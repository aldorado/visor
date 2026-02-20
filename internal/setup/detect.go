package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type State struct {
	FirstRun    bool
	Missing     []string
	ProjectRoot string
}

func Detect(projectRoot, dataDir string) (State, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return State{}, fmt.Errorf("project root is required")
	}
	if strings.TrimSpace(dataDir) == "" {
		dataDir = "data"
	}

	missing := make([]string, 0)
	envPath := filepath.Join(projectRoot, ".env")
	if _, err := os.Stat(envPath); err != nil {
		if os.IsNotExist(err) {
			missing = append(missing, ".env")
		} else {
			return State{}, fmt.Errorf("stat %s: %w", envPath, err)
		}
	}

	dataPath := dataDir
	if !filepath.IsAbs(dataPath) {
		dataPath = filepath.Join(projectRoot, dataDir)
	}
	if _, err := os.Stat(dataPath); err != nil {
		if os.IsNotExist(err) {
			missing = append(missing, "data_dir")
		} else {
			return State{}, fmt.Errorf("stat %s: %w", dataPath, err)
		}
	}

	bootstrapEnvMissing := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")) == "" || strings.TrimSpace(os.Getenv("USER_PHONE_NUMBER")) == ""
	firstRun := len(missing) > 0 && bootstrapEnvMissing
	return State{FirstRun: firstRun, Missing: missing, ProjectRoot: projectRoot}, nil
}

func BuildContext(state State) string {
	if !state.FirstRun {
		return ""
	}
	missing := strings.Join(state.Missing, ", ")
	return strings.TrimSpace(`
[first-run setup mode]
project is not fully initialized yet (` + missing + `).

you must guide the user through setup in small steps:
1) collect required env vars (telegram token, owner chat id, agent backend) and write/update .env
2) validate telegram token
3) set webhook url + optional secret
4) verify /health is reachable

if you need to execute setup actions, include one dedicated setup action json block only in your final response.
`)
}
