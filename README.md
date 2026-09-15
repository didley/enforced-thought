# enforced-thought

A local Claude Code hook that blocks a prompt until you've stated, in your
own words, the problem you're solving and your hypothesis for the approach.
Re-checkpoints are required periodically (micro) and on scope/branch change
(macro), so an agentic coding session can't run unattended past the point
where you've stopped tracking it.

## Why

LLMs make it easy to ship code you don't actually understand, built on
requirements nobody validated. This gate forces a stated hypothesis before
each prompt goes to the model — cheap insurance against comprehension debt,
and a sharper target for the model to work against.

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
