// Package state manages checkpoint state: read/write and expiry-policy
// evaluation.
//
// v1 keeps one active task per project directory (identified by the
// project's .enforced-thought/state.json). "think start" replaces it;
// there's no multi-task registry because nothing in v1 needs more than one
// active task at a time.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"enforced-thought/internal/config"
)

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func minutesSince(iso string) float64 {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return 0
	}
	return time.Since(t).Minutes()
}

func GitBranch(projectRoot string) string {
	out, err := exec.Command("git", "-C", projectRoot, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func GitHead(projectRoot string) string {
	out, err := exec.Command("git", "-C", projectRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

type Task struct {
	ID            string   `json:"id"`
	Problem       string   `json:"problem"`
	Hypothesis    string   `json:"hypothesis"`
	CreatedAt     string   `json:"created_at"`
	Branch        string   `json:"branch"`
	Head          string   `json:"head"`
	DeclaredPaths []string `json:"declared_paths"`
}

type State struct {
	Task                 *Task  `json:"task"`
	LastCheckpointAt     string `json:"last_checkpoint_at"`
	LastReaffirmAt       string `json:"last_reaffirm_at"`
	PromptsSinceReaffirm int    `json:"prompts_since_reaffirm"`
	LastTranscriptPath   string `json:"last_transcript_path"`
}

func Load(projectRoot string) State {
	var s State
	data, err := os.ReadFile(config.StatePath(projectRoot))
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}
	}
	return s
}

func Save(projectRoot string, s State) error {
	if _, err := config.InitProject(projectRoot); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(config.StatePath(projectRoot), data, 0o644)
}

func AppendLog(projectRoot, event string, detail map[string]any) error {
	if _, err := config.InitProject(projectRoot); err != nil {
		return err
	}
	entry := map[string]any{"ts": now(), "event": event}
	for k, v := range detail {
		entry[k] = v
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(config.LogPath(projectRoot), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

func StartTask(projectRoot, problem, hypothesis string, declaredPaths []string) (Task, error) {
	task := Task{
		ID:            time.Now().UTC().Format("20060102-150405"),
		Problem:       problem,
		Hypothesis:    hypothesis,
		CreatedAt:     now(),
		Branch:        GitBranch(projectRoot),
		Head:          GitHead(projectRoot),
		DeclaredPaths: declaredPaths,
	}
	s := Load(projectRoot)
	s.Task = &task
	s.LastCheckpointAt = now()
	s.LastReaffirmAt = now()
	s.PromptsSinceReaffirm = 0
	if err := Save(projectRoot, s); err != nil {
		return task, err
	}
	return task, AppendLog(projectRoot, "checkpoint", map[string]any{
		"problem": problem, "hypothesis": hypothesis,
	})
}

func Reaffirm(projectRoot, note string) error {
	s := Load(projectRoot)
	s.LastReaffirmAt = now()
	s.PromptsSinceReaffirm = 0
	if err := Save(projectRoot, s); err != nil {
		return err
	}
	return AppendLog(projectRoot, "reaffirm", map[string]any{"note": note})
}

// CheckResult.Status is one of "ok", "need_checkpoint", "need_reaffirm".
type CheckResult struct {
	Status string
	Reason string
}

func CheckCheckpoint(projectRoot string) CheckResult {
	cfg := config.Load(projectRoot)
	s := Load(projectRoot)

	if s.Task == nil {
		return CheckResult{"need_checkpoint", "No checkpoint recorded yet for this project."}
	}

	currentBranch := GitBranch(projectRoot)
	if s.Task.Branch != "" && currentBranch != "" && currentBranch != s.Task.Branch {
		return CheckResult{
			"need_checkpoint",
			fmt.Sprintf("Git branch changed (%s -> %s) since last checkpoint.", s.Task.Branch, currentBranch),
		}
	}

	reaffirmBase := s.LastReaffirmAt
	if reaffirmBase == "" {
		reaffirmBase = s.LastCheckpointAt
	}
	if reaffirmBase != "" {
		elapsed := minutesSince(reaffirmBase)
		if elapsed >= float64(cfg.ExpiryMicroMinutes) {
			return CheckResult{
				"need_reaffirm",
				fmt.Sprintf("%d min since last check-in (limit %d).", int(elapsed), cfg.ExpiryMicroMinutes),
			}
		}
	}
	if s.PromptsSinceReaffirm >= cfg.ExpiryMicroRequests {
		return CheckResult{
			"need_reaffirm",
			fmt.Sprintf("%d prompts since last check-in (limit %d).", s.PromptsSinceReaffirm, cfg.ExpiryMicroRequests),
		}
	}

	return CheckResult{"ok", ""}
}

func RecordPromptPassed(projectRoot string) error {
	s := Load(projectRoot)
	s.PromptsSinceReaffirm++
	return Save(projectRoot, s)
}

func RecordTranscriptPath(projectRoot, transcriptPath string) error {
	if transcriptPath == "" {
		return nil
	}
	s := Load(projectRoot)
	s.LastTranscriptPath = transcriptPath
	return Save(projectRoot, s)
}

// AbsRoot resolves a possibly-relative project root the way the CLI/hooks pass it.
func AbsRoot(path string) string {
	if path == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
