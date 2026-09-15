// Package audit implements `think audit`: a heuristic scope/requirements
// check.
//
// v1 note: the plan calls for an LLM-judge comparing the checkpoint against
// the real diff. Shipping that requires deciding on a provider/key/model
// now, which is a separate decision from the gate mechanism itself. v1
// ships a heuristic version (declared-path scope check + weak-checkpoint
// detection) that needs no API key, and RunAudit is structured so an LLM
// judge is a drop-in replacement for the flagging logic later.
package audit

import (
	"os/exec"
	"strings"

	"enforced-thought/internal/state"
)

type Report struct {
	Flags        []string
	ChangedFiles []string
}

func (r Report) Render() string {
	var b strings.Builder
	b.WriteString("Changed files:\n")
	if len(r.ChangedFiles) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, f := range r.ChangedFiles {
			b.WriteString("  " + f + "\n")
		}
	}
	b.WriteString("\n")
	if len(r.Flags) == 0 {
		b.WriteString("No flags.\n")
	} else {
		b.WriteString("Flags:\n")
		for _, f := range r.Flags {
			b.WriteString("  - " + f + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func changedFiles(projectRoot string) []string {
	out, err := exec.Command("git", "-C", projectRoot, "diff", "--name-only", "HEAD").Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

func Run(projectRoot string) Report {
	s := state.Load(projectRoot)
	changed := changedFiles(projectRoot)
	var flags []string

	if s.Task == nil {
		flags = append(flags, "No checkpoint on record for this project — nothing to audit against.")
		return Report{Flags: flags, ChangedFiles: changed}
	}

	if len(s.Task.DeclaredPaths) > 0 {
		var undeclared []string
		for _, f := range changed {
			matched := false
			for _, p := range s.Task.DeclaredPaths {
				if strings.HasPrefix(f, p) {
					matched = true
					break
				}
			}
			if !matched {
				undeclared = append(undeclared, f)
			}
		}
		if len(undeclared) > 0 {
			flags = append(flags, "Diff touches paths not mentioned at checkpoint time: "+strings.Join(undeclared, ", "))
		}
	}

	for _, pair := range []struct{ label, text string }{
		{"problem", s.Task.Problem},
		{"hypothesis", s.Task.Hypothesis},
	} {
		words := len(strings.Fields(pair.text))
		if words < 6 {
			flags = append(flags, "Checkpoint '"+pair.label+"' is very short — may not reflect real understanding.")
		}
	}

	return Report{Flags: flags, ChangedFiles: changed}
}
