package agent

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
)

type ProcessConfig struct {
	Command        string        // e.g. "pi"
	Args           []string      // e.g. ["--mode", "rpc"]
	RestartDelay   time.Duration // delay before restarting crashed process
	PeriodicRestart time.Duration // restart interval (0 = disabled)
	PromptTimeout  time.Duration // max time per prompt
}

// ProcessManager manages a persistent CLI agent subprocess.
// Concrete adapters (pi, claude, etc.) embed this and implement the Agent interface.
type ProcessManager struct {
	cfg     ProcessConfig
	mu      sync.Mutex
	cmd     *exec.Cmd
	running bool
	stopCh  chan struct{}
}

func NewProcessManager(cfg ProcessConfig) *ProcessManager {
	return &ProcessManager{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

func (pm *ProcessManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.running {
		return fmt.Errorf("process already running")
	}
	if err := pm.spawn(); err != nil {
		return err
	}
	go pm.watchLoop()
	if pm.cfg.PeriodicRestart > 0 {
		go pm.periodicRestartLoop()
	}
	return nil
}

func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	close(pm.stopCh)
	if pm.cmd != nil && pm.cmd.Process != nil {
		return pm.cmd.Process.Kill()
	}
	return nil
}

func (pm *ProcessManager) Cmd() *exec.Cmd {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.cmd
}

func (pm *ProcessManager) spawn() error {
	cmd := exec.Command(pm.cfg.Command, pm.cfg.Args...)
	pm.cmd = cmd
	pm.running = true
	log.Printf("agent: spawning %s %v", pm.cfg.Command, pm.cfg.Args)
	return nil
}

func (pm *ProcessManager) Restart() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.cmd != nil && pm.cmd.Process != nil {
		_ = pm.cmd.Process.Kill()
		_ = pm.cmd.Wait()
	}
	pm.running = false
	return pm.spawn()
}

// watchLoop monitors the child process and restarts on crash.
func (pm *ProcessManager) watchLoop() {
	for {
		select {
		case <-pm.stopCh:
			return
		default:
		}

		pm.mu.Lock()
		cmd := pm.cmd
		pm.mu.Unlock()

		if cmd == nil || cmd.Process == nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// wait for process to exit
		err := cmd.Wait()
		select {
		case <-pm.stopCh:
			return
		default:
		}

		if err != nil {
			log.Printf("agent: process exited: %v, restarting in %v", err, pm.cfg.RestartDelay)
		} else {
			log.Printf("agent: process exited cleanly, restarting in %v", pm.cfg.RestartDelay)
		}

		time.Sleep(pm.cfg.RestartDelay)

		pm.mu.Lock()
		pm.running = false
		if err := pm.spawn(); err != nil {
			log.Printf("agent: restart failed: %v", err)
		}
		pm.mu.Unlock()
	}
}

func (pm *ProcessManager) periodicRestartLoop() {
	ticker := time.NewTicker(pm.cfg.PeriodicRestart)
	defer ticker.Stop()
	for {
		select {
		case <-pm.stopCh:
			return
		case <-ticker.C:
			log.Printf("agent: periodic restart")
			if err := pm.Restart(); err != nil {
				log.Printf("agent: periodic restart failed: %v", err)
			}
		}
	}
}
