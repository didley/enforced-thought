// Package hook implements the Claude Code hook entrypoints
// (UserPromptSubmit, PreToolUse).
//
// Contract (per Claude Code's hooks reference): the hook receives a JSON
// payload on stdin (fields include at least `cwd`, `hook_event_name`, and
// `transcript_path`). To block, print a JSON decision to stdout and exit 0.
// To allow, exit 0 (optionally with JSON carrying `additionalContext`).
package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"enforced-thought/internal/state"
)

type input struct {
	Cwd            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	TranscriptPath string `json:"transcript_path"`
}

func readInput() input {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return input{}
	}
	var in input
	_ = json.Unmarshal(raw, &in) // malformed/empty stdin just yields a zero-value input
	return in
}

func block(reason string) {
	out, _ := json.Marshal(map[string]string{"decision": "block", "reason": reason})
	fmt.Println(string(out))
	os.Exit(0)
}

func allow(additionalContext string) {
	if additionalContext != "" {
		out, _ := json.Marshal(map[string]any{
			"hookSpecificOutput": map[string]string{
				"hookEventName":     "UserPromptSubmit",
				"additionalContext": additionalContext,
			},
		})
		fmt.Println(string(out))
	}
	os.Exit(0)
}

func UserPromptSubmit() {
	in := readInput()
	root := state.AbsRoot(in.Cwd)
	_ = state.RecordTranscriptPath(root, in.TranscriptPath)

	result := state.CheckCheckpoint(root)
	switch result.Status {
	case "need_checkpoint":
		_ = state.AppendLog(root, "block", map[string]any{"hook": "UserPromptSubmit", "reason": result.Reason})
		block(fmt.Sprintf(
			"[enforced-thought] %s Run `think checkpoint` in this project directory before continuing.",
			result.Reason,
		))
		return
	case "need_reaffirm":
		_ = state.AppendLog(root, "block", map[string]any{"hook": "UserPromptSubmit", "reason": result.Reason})
		block(fmt.Sprintf(
			"[enforced-thought] %s Run `think reaffirm` for a quick one-line check-in, then retry.",
			result.Reason,
		))
		return
	}

	_ = state.RecordPromptPassed(root)
	s := state.Load(root)
	context := ""
	if s.Task != nil {
		context = fmt.Sprintf("[enforced-thought] Stated hypothesis for this task: %s", s.Task.Hypothesis)
	}
	allow(context)
}

func PreToolUse() {
	in := readInput()
	root := state.AbsRoot(in.Cwd)

	result := state.CheckCheckpoint(root)
	if result.Status == "need_checkpoint" {
		_ = state.AppendLog(root, "block", map[string]any{"hook": "PreToolUse", "reason": result.Reason})
		block(fmt.Sprintf(
			"[enforced-thought] %s Run `think checkpoint` before further edits.",
			result.Reason,
		))
		return
	}
	// A micro reaffirm expiry is not enforced at the tool-call level: it
	// would fire mid-edit on a timer even when the prompt gate already let
	// the turn through, blocking tool calls the user never got a chance to
	// avoid triggering. Only the harder "no checkpoint at all" or "branch
	// changed" conditions gate tool use directly.
	os.Exit(0)
}
