# enforced-thought

Go CLI + Claude Code hooks. No external dependencies — keep it that way
unless there's a strong reason (this includes the TOML config reader and
paste-similarity check, both hand-rolled on purpose).

## Build/test

```bash
just build          # builds ./think, symlinks ./tnk to it
go vet ./...
gofmt -l .           # must print nothing before committing
```

There's no test suite yet — verify by hand: `go build`, then drive `think`
directly and pipe JSON into `think hook-user-prompt-submit` /
`think hook-pre-tool-use` to check the block/allow contract.

## Layout

- `cmd/think` — CLI entrypoint (`think`, aliased as `tnk` via symlink).
- `internal/state` — checkpoint state (`.enforced-thought/state.json`) and
  expiry policy.
- `internal/config` — `.enforced-thought/config.toml` loading.
- `internal/hook` — the `UserPromptSubmit` / `PreToolUse` hook contract
  (stdin JSON in, block/allow JSON out).
- `internal/audit` — heuristic diff-vs-checkpoint scope check (`think audit`).
- `internal/paste` — flags checkpoint answers pasted from the transcript.

## Conventions

- `gofmt` clean, `go vet` clean, before every commit.
- No new abstractions beyond what's needed — state is a single JSON file
  per project, config is a fixed-shape TOML subset, not a general parser.
- `think`/`tnk` dispatch by subcommand, not by `os.Args[0]` — don't make
  behavior depend on which name it was invoked as.
