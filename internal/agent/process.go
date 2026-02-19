package agent

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"visor/internal/observability"
)

type ProcessConfig struct {
	Command         string
	Args            []string
	RestartDelay    time.Duration
	PeriodicRestart time.Duration // 0 = disabled
	PromptTimeout   time.Duration
}

// ProcessManager manages a persistent CLI agent subprocess with stdin/stdout pipes.
type ProcessManager struct {
	cfg     ProcessConfig
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	running bool
	stopCh  chan struct{}
	log     *observability.Logger
}

func NewProcessManager(cfg ProcessConfig) *ProcessManager {
	return &ProcessManager{
		cfg:    cfg,
		stopCh: make(chan struct{}),
		log:    observability.Component("agent.process"),
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
	if pm.stdin != nil {
		pm.stdin.Close()
	}
	if pm.cmd != nil && pm.cmd.Process != nil {
		return pm.cmd.Process.Kill()
	}
	return nil
}

func (pm *ProcessManager) spawn() error {
	cmd := exec.Command(pm.cfg.Command, pm.cfg.Args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return fmt.Errorf("start process: %w", err)
	}

	pm.cmd = cmd
	pm.stdin = stdin
	pm.scanner = bufio.NewScanner(stdout)
	pm.scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	pm.running = true
	pm.log.Info(nil, "agent process spawned", "command", pm.cfg.Command, "args", pm.cfg.Args, "pid", cmd.Process.Pid)
	return nil
}

// Stdin returns the writer to the child process stdin.
func (pm *ProcessManager) Stdin() io.Writer {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.stdin
}

// Scanner returns the line scanner on child process stdout.
func (pm *ProcessManager) Scanner() *bufio.Scanner {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.scanner
}

func (pm *ProcessManager) Restart() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.stdin != nil {
		pm.stdin.Close()
	}
	if pm.cmd != nil && pm.cmd.Process != nil {
		_ = pm.cmd.Process.Kill()
		_ = pm.cmd.Wait()
	}
	pm.running = false
	return pm.spawn()
}

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

		err := cmd.Wait()
		select {
		case <-pm.stopCh:
			return
		default:
		}

		if err != nil {
			pm.log.Warn(nil, "agent process exited", "error", err.Error(), "restart_delay", pm.cfg.RestartDelay.String())
		} else {
			pm.log.Info(nil, "agent process exited cleanly", "restart_delay", pm.cfg.RestartDelay.String())
		}

		time.Sleep(pm.cfg.RestartDelay)

		pm.mu.Lock()
		pm.running = false
		if err := pm.spawn(); err != nil {
			pm.log.Error(nil, "agent restart failed", "error", err.Error())
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
			pm.log.Info(nil, "agent periodic restart")
			if err := pm.Restart(); err != nil {
				pm.log.Error(nil, "agent periodic restart failed", "error", err.Error())
			}
		}
	}
}
