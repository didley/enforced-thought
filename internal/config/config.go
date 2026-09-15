// Package config loads per-project settings for enforced-thought.
//
// State lives under <project_root>/.enforced-thought/. The project root is
// whatever `cwd` Claude Code reports (hooks) or the shell's cwd (CLI
// invoked manually).
package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultConfigTOML = `[expiry]
# Minutes of inactivity before a lightweight "still on track?" reaffirm is required.
micro_minutes = 25
# Number of allowed prompts between reaffirms, whichever threshold hits first.
micro_requests = 20

[audit]
enabled = true
`

type Config struct {
	ExpiryMicroMinutes  int
	ExpiryMicroRequests int
	AuditEnabled        bool
}

func Defaults() Config {
	return Config{ExpiryMicroMinutes: 25, ExpiryMicroRequests: 20, AuditEnabled: true}
}

func StateDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".enforced-thought")
}

func ConfigPath(projectRoot string) string {
	return filepath.Join(StateDir(projectRoot), "config.toml")
}

func StatePath(projectRoot string) string {
	return filepath.Join(StateDir(projectRoot), "state.json")
}

func LogPath(projectRoot string) string {
	return filepath.Join(StateDir(projectRoot), "log.jsonl")
}

// Load reads config.toml if present, otherwise returns defaults. It only
// understands the small fixed shape this project uses ([expiry], [audit]
// sections with int/bool values) — not a general TOML parser.
func Load(projectRoot string) Config {
	cfg := Defaults()
	f, err := os.Open(ConfigPath(projectRoot))
	if err != nil {
		return cfg
	}
	defer f.Close()

	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if idx := strings.Index(val, "#"); idx != -1 {
			val = strings.TrimSpace(val[:idx])
		}

		switch {
		case section == "expiry" && key == "micro_minutes":
			if n, err := strconv.Atoi(val); err == nil {
				cfg.ExpiryMicroMinutes = n
			}
		case section == "expiry" && key == "micro_requests":
			if n, err := strconv.Atoi(val); err == nil {
				cfg.ExpiryMicroRequests = n
			}
		case section == "audit" && key == "enabled":
			cfg.AuditEnabled = val == "true"
		}
	}
	return cfg
}

// InitProject creates .enforced-thought/ with a default config if missing.
func InitProject(projectRoot string) (string, error) {
	dir := StateDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := ConfigPath(projectRoot)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(DefaultConfigTOML), 0o644); err != nil {
			return "", err
		}
	}
	return dir, nil
}
