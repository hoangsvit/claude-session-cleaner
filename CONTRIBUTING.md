# Contributing to Claude Cleaner

Thank you for your interest in contributing!

## Tech stack

| Layer | Tool |
| --- | --- |
| CLI / TUI | Go 1.25+, [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| npm wrapper | Node.js 20+ (thin shim — downloads Go binary on install) |
| Releases | [GoReleaser](https://goreleaser.com) + GitHub Actions |
| Demo GIFs | [VHS](https://github.com/charmbracelet/vhs) |

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/) (for npm wrapper scripts)
- Git

## Set up

```bash
git clone https://github.com/ePlus-DEV/claude-cleaner.git
cd claude-cleaner
go mod tidy
```

## Build & run

```bash
go build -o claude-cleaner .
./claude-cleaner --help
```

Or without building:

```bash
go run . --help
go run . --claude-dir /path/to/.claude
```

### Test the update flow

```bash
go run . --mock-update
```

Injects a fake `v99.0.0` — triggers the update prompt on startup without publishing a real release. Press `n` to skip into the list; header shows `⬆ v99.0.0 available  press u to update`.

## Run with fake data (no real Claude install needed)

Use the included seed script to create fake session data:

```bash
# macOS / Linux
node demo/seed.js /tmp/claude-demo
go run . --claude-dir /tmp/claude-demo
```

```powershell
# Windows
node demo/seed.js $env:TEMP\claude-demo
go run . --claude-dir $env:TEMP\claude-demo
```

Creates 5 fake project sessions of various sizes — enough to test all TUI flows (navigate, select, delete, cancel) without touching real Claude data.

## Tests

```bash
go test -v ./...
go test -v -run TestFormatTokens ./...   # run a specific test
```

The test suite covers: TUI state transitions, all key bindings, progress bar messages, CLI detection messages, update check messages, scanner helpers (token fields, dedup, path encoding), deletion safety (`safeRemove`), and edge cases (empty list, malformed JSON, nonexistent paths).

### Test helpers

`model_test.go` exports two session factories:

| Helper | Use |
| --- | --- |
| `fakeSessions(n)` | n minimal sessions — fast, for state/key tests |
| `realisticSessions()` | 7 sessions with varied tokens, sizes, HasData flags — mirrors real data shape |

## Project structure

```
main.go           — entry point, flag parsing, resolveClaudeDir, --mock-update
scanner.go        — Session type, scanning from ~/.claude.json, safeRemove,
                    smartDelete, RunDelete, DetectClaudeCLI, formatTokens, helpers
updater.go        — CheckLatestVersion (npm registry), semver comparison
model.go          — Bubble Tea TUI model
                    states: loading → updatePrompt → list → confirm → deleting → done
                    key handlers, viewList, viewDeleting (progress bar), viewDone,
                    renderHeader (claude CLI status, version + update badge, scanned time)
scanner_test.go   — unit tests for scanning, tokens, dedup, deletion helpers
model_test.go     — unit tests for all TUI states, key bindings, message handling
updater_test.go   — unit tests for semver comparison, update message handling

scripts/
  install.js      — npm postinstall: downloads correct Go binary from GitHub Releases
  run.js          — npm bin shim: finds and executes the downloaded binary
  sync-version.js — syncs package.json version → main.go

demo/
  seed.js         — creates fake ~/.claude/projects data for VHS recordings
  *.tape          — VHS scripts for demo GIFs

.github/workflows/
  ci.yml          — Go tests (1.25 / 1.26 × Windows / macOS / Linux) on push / PR
  release.yml     — GoReleaser (binaries) + npm publish (wrapper)
  demo.yml        — regenerate demo GIFs on change
  snyk.yml        — dependency vulnerability scan (push / PR + weekly)
```

### Key data flow

```
~/.claude.json  ──→  scanSessions()  ──→  []Session  ──→  model.sessions
                      └─ token fields: lastTotalInputTokens etc.
                      └─ deduplicateProjects (Windows path case-insensitive)
                      └─ encodePath → ~/.claude/projects/<hash>/ for HasData/size/mtime

Deletion chain (smartDelete):
  1. exec.LookPath("claude") → claude project purge -y <path>
  2. os.Stat(s.Path) — if still exists → safeRemove (os.RemoveAll, path-validated)

All-selected optimisation:
  RunDelete detects len(selected)==len(sessions) → claude project purge --all -y
  → verify each folder → safeRemove remaining
```

## Releasing a new version

Version is sourced from `package.json`. Running `npm version` syncs it to `main.go` automatically.

```bash
# Patch release (1.0.0 → 1.0.1)
npm version patch

# Minor release (1.0.0 → 1.1.0)
npm version minor

# Major release (1.0.0 → 2.0.0)
npm version major
```

Then push with tags:

```bash
git push --follow-tags
```

GitHub Actions will:
1. Build cross-platform Go binaries via GoReleaser → create GitHub Release
2. Publish the npm wrapper (needs `NPM_TOKEN` secret)

## Submitting changes

1. Fork the repository.
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes and add tests where applicable.
4. Ensure `go test ./...` passes.
5. Open a pull request against `main`.

### Commit style

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add session filtering by size
fix: prevent deletion outside projects directory
chore: update dependencies
docs: improve README install section
```

## Reporting issues

Open an issue at [github.com/ePlus-DEV/claude-cleaner/issues](https://github.com/ePlus-DEV/claude-cleaner/issues).

Include:
- OS and architecture
- `claude-cleaner --version` output
- Steps to reproduce
- Expected vs actual behavior

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
