# enforced-thought

A local Claude Code hook that blocks a prompt until you've stated, in your
own words, the problem you're solving and your hypothesis for the approach.
Re-checkpoints are required periodically (micro) and on scope/branch change
(macro), so an agentic coding session can't run unattended past the point
where you've stopped tracking it.

## Why

Prompt to accept to next. No hypothesis, no explanation, no idea what
breaks if the requirement was wrong — just code that ships and a
comprehension bill that lands on you or whoever inherits it. The evidence
that this actually costs something is no longer anecdotal:

- **A 52-developer RCT** [[1]](#sources) found the AI-assisted group scored
  **17% lower** on comprehension of their own code, measured minutes later
  (d=0.738, p=0.01) — with no time savings to show for it, and the largest
  gap on debugging specifically. The single strongest predictor of outcome
  wasn't whether someone used AI, it was whether they asked follow-up
  questions instead of going straight from prompt to accept.
- **A metacognitive-script study** [[2]](#sources) put a number on the fix:
  engineers gated by a teach-back step failed a later AI-blackout
  maintenance task on their own code **39%** of the time, versus **77%**
  for the ungated group — for about 14 minutes of friction per gate.
- **A field study of professional developers** [[3]](#sources) — 13 field
  observations plus a 99-developer survey — found the ones who actually
  ship reliably "don't vibe, they control": roughly 69% review every
  AI-generated change, 75% read every line before it merges.
- **"Comprehension Debt in GenAI-Assisted Software Engineering Projects"**
  [[4]](#sources) names the failure mode this tool targets and its four
  accumulation patterns: AI-as-black-box acceptance, context-mismatch debt,
  dependency-induced atrophy, verification-bypass.

This tool takes that same control and moves it one step earlier: instead
of explaining the diff after the model writes it, you state the problem
and your hypothesis *before* it does. No stated hypothesis, no prompt.

The payoff isn't just fewer bugs:

- **The model does better work.** A stated hypothesis is a sharper target
  than a bare prompt — it gives the model something concrete to confirm,
  refute, or push back on, instead of guessing at intent from a one-line ask.
- **You stay in the loop, not just in the room.** Reviewing output someone
  else generated is a different job from thinking through a problem
  yourself, and only one of those keeps you sharp, engaged, and growing —
  the other is just rubber-stamping on a schedule.
- **Your team's trust in your name on a PR stays earned.** "I don't know
  why this works" code and unvalidated requirements are cheap to write and
  expensive to discover later, usually by someone else, usually at the
  worst time.

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

## Sources

1. Shen & Tamkin, Anthropic, "How AI assistance impacts the formation of
   coding skills" (Jan 2026) — <https://www.anthropic.com/research/AI-assistance-coding-skills>
   · paper: <https://arxiv.org/abs/2601.20245>
2. "Mitigating Epistemic Debt in AI-Assisted Programming using
   Metacognitive Scripts" — <https://arxiv.org/abs/2602.20206>
3. Huang, Reyna et al. (Cornell/UCSD), "Professional developers don't vibe,
   they control" — <https://arxiv.org/abs/2512.14012>
4. "Comprehension Debt in GenAI-Assisted Software Engineering Projects" —
   <https://arxiv.org/abs/2604.13277>
