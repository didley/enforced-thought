// Command think is the enforced-thought CLI: init, start, checkpoint,
// reaffirm, audit, status, and the Claude Code hook entrypoints.
//
// The binary is also installed under the "tnk" name as a shorthand alias
// (see Makefile) — behavior is identical either way since dispatch is by
// subcommand, not by the invoked binary name.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"enforced-thought/internal/audit"
	"enforced-thought/internal/config"
	"enforced-thought/internal/hook"
	"enforced-thought/internal/paste"
	"enforced-thought/internal/state"
)

func projectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func promptLine(question string) string {
	fmt.Println(question)
	fmt.Print("> ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func validateAnswer(label, answer string) string {
	words := len(strings.Fields(answer))
	if words < paste.MinWords {
		return fmt.Sprintf("'%s' answer is too short (%d words, minimum %d).", label, words, paste.MinWords)
	}
	return ""
}

func runInit() int {
	root := projectRoot()
	dir, err := config.InitProject(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	fmt.Println("Initialized enforced-thought in", dir)
	return 0
}

func runCheckpoint(args []string) int {
	fs := flag.NewFlagSet("checkpoint", flag.ExitOnError)
	problemFlag := fs.String("problem", "", "")
	hypothesisFlag := fs.String("hypothesis", "", "")
	pathsFlag := fs.String("paths", "", "comma-separated path prefixes this task will touch")
	fs.Parse(args)

	root := projectRoot()
	s := state.Load(root)
	transcriptTexts := paste.RecentAssistantTexts(s.LastTranscriptPath, 200)

	problem := *problemFlag
	if problem == "" {
		problem = promptLine("What problem/requirement are you solving right now?")
	}
	if errMsg := validateAnswer("problem", problem); errMsg != "" {
		fmt.Fprintln(os.Stderr, "error:", errMsg)
		return 1
	}
	if warn := paste.Warning(problem, transcriptTexts); warn != "" {
		fmt.Fprintln(os.Stderr, "warning:", warn)
	}

	hypothesis := *hypothesisFlag
	if hypothesis == "" {
		hypothesis = promptLine("What's your hypothesis for the approach, or what do you expect to change?")
	}
	if errMsg := validateAnswer("hypothesis", hypothesis); errMsg != "" {
		fmt.Fprintln(os.Stderr, "error:", errMsg)
		return 1
	}
	if warn := paste.Warning(hypothesis, transcriptTexts); warn != "" {
		fmt.Fprintln(os.Stderr, "warning:", warn)
	}

	var declaredPaths []string
	if *pathsFlag != "" {
		declaredPaths = strings.Split(*pathsFlag, ",")
	}

	task, err := state.StartTask(root, problem, hypothesis, declaredPaths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	fmt.Printf("Checkpoint recorded (task %s).\n", task.ID)
	return 0
}

func runReaffirm(args []string) int {
	fs := flag.NewFlagSet("reaffirm", flag.ExitOnError)
	noteFlag := fs.String("note", "", "")
	fs.Parse(args)

	root := projectRoot()
	note := *noteFlag
	if note == "" {
		note = promptLine("Still on track? What, if anything, changed?")
	}
	if len(strings.Fields(note)) < 3 {
		fmt.Fprintln(os.Stderr, "error: reaffirm note is too short — give at least a few words.")
		return 1
	}
	if err := state.Reaffirm(root, note); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	fmt.Println("Check-in recorded.")
	return 0
}

func runAudit() int {
	root := projectRoot()
	report := audit.Run(root)
	fmt.Println(report.Render())
	return 0
}

func runStatus() int {
	root := projectRoot()
	result := state.CheckCheckpoint(root)
	s := state.Load(root)
	if s.Task != nil {
		fmt.Printf("Task %s (branch %s):\n", s.Task.ID, s.Task.Branch)
		fmt.Printf("  problem:    %s\n", s.Task.Problem)
		fmt.Printf("  hypothesis: %s\n", s.Task.Hypothesis)
	} else {
		fmt.Println("No active checkpoint.")
	}
	if result.Reason != "" {
		fmt.Printf("Status: %s — %s\n", result.Status, result.Reason)
	} else {
		fmt.Printf("Status: %s\n", result.Status)
	}
	return 0
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: think <command> [flags]   (or: tnk <command> [flags])

commands:
  init                       set up .enforced-thought/ in the current project
  start                      start a new task checkpoint (alias for checkpoint)
  checkpoint                 record a problem/hypothesis checkpoint
  reaffirm                   light one-line check-in to clear a micro expiry
  audit                      compare the current diff against the stated checkpoint
  status                     show the current checkpoint and gate status
  hook-user-prompt-submit    Claude Code UserPromptSubmit hook
  hook-pre-tool-use          Claude Code PreToolUse hook`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var code int
	switch cmd {
	case "init":
		code = runInit()
	case "start", "checkpoint":
		code = runCheckpoint(args)
	case "reaffirm":
		code = runReaffirm(args)
	case "audit":
		code = runAudit()
	case "status":
		code = runStatus()
	case "hook-user-prompt-submit":
		hook.UserPromptSubmit()
		return
	case "hook-pre-tool-use":
		hook.PreToolUse()
		return
	default:
		usage()
		code = 1
	}
	os.Exit(code)
}
