# enforced-thought

A local Claude Code hook that blocks a prompt until you've stated, in your
own words, the problem you're solving and your hypothesis for the approach.
Re-checkpoints are required periodically (micro) and on scope/branch change
(macro), so an agentic coding session can't run unattended past the point
where you've stopped tracking it.

## Why

LLMs let you ship code you never actually understood, built on requirements
nobody validated — pushing the comprehension burden onto whoever reviews it
now, and onto whoever maintains it later, once it's too late to cheaply
recover. Research on this ("comprehension debt") names four ways it
accumulates:

- **AI-as-black-box acceptance** — merging output you couldn't explain if asked.
- **Context-mismatch debt** — code that's correct for a requirement nobody checked was the right one.
- **Dependency-induced atrophy** — the muscle for solving it yourself quietly weakens.
- **Verification-bypass** — "it works" standing in for "I know why it works."

A metacognitive-script study measured the cost of skipping this: engineers
who used AI ungated failed a later AI-blackout maintenance task on their own
code **77% of the time**, versus **39%** for a group forced through a
teach-back gate first — at roughly 14 minutes of friction per gate. This
tool is the same bet, moved one step earlier: instead of explaining the
diff after the model writes it, you state the problem and your hypothesis
*before* it does, so the prompt itself has to be grounded in something you
actually thought through.

Past correctness, the point is who you stay as while doing this job: an
engineer who can explain not just why a change works but why it's wrong, or
one whose name is on code they never really held in their head — and a
team's trust erodes the same way individual comprehension does, quietly,
one ungated merge at a time.

## Install

```bash
just build     # builds ./think, symlinks ./tnk to it
just install   # installs both into your Go bin dir
```

`think` and `tnk` are the same binary — `tnk` is just a shorthand alias.

## Use

```bash
think init                 # set up .enforced-thought/ in a project
think checkpoint           # state your problem + hypothesis
think reaffirm             # lightweight periodic check-in
think audit                # heuristic diff-vs-checkpoint scope check
think status                # show current checkpoint + gate status
```

Wire the hooks into Claude Code by merging [settings.json.example](settings.json.example)
into `~/.claude/settings.json` (or a project-level `.claude/settings.json`).
This covers both the CLI and the desktop app, since they share the same
hooks config.

## How it works

- `UserPromptSubmit` hook: blocks every prompt until `.enforced-thought/state.json`
  has a valid, unexpired checkpoint for the current project + git branch.
- `PreToolUse` hook (`Edit|Write`): same check, as defense-in-depth against
  a checkpoint expiring mid-session.
- State is a single JSON file per project — no database, no server.

See [go.mod](go.mod) for the module layout: `cmd/think` is the CLI entrypoint,
`internal/{state,config,hook,audit,paste}` hold the logic.
