# Contributing to Claude Cleaner

Thank you for your interest in contributing!

## Tech stack

| Layer | Tool |
| --- | --- |
| CLI / TUI | Go 1.22+, [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| npm wrapper | Node.js 20+ (thin shim — downloads Go binary on install) |
| Releases | [GoReleaser](https://goreleaser.com) + GitHub Actions |
| Demo GIFs | [VHS](https://github.com/charmbracelet/vhs) |

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
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

## Tests

```bash
go test -v ./...
```

## Project structure

```
main.go          — entry point, flag parsing, resolveClaudeDir
scanner.go       — Session type, filesystem scanning, safeRemove, helpers
model.go         — Bubble Tea TUI model (loading → list → confirm → deleting → done)
scanner_test.go  — unit tests

scripts/
  install.js     — npm postinstall: downloads correct Go binary from GitHub Releases
  run.js         — npm bin shim: finds and executes the downloaded binary
  sync-version.js — syncs package.json version → main.go

demo/
  seed.js        — creates fake ~/.claude/projects data for VHS recordings
  *.tape         — VHS scripts for demo GIFs

.github/workflows/
  ci.yml         — Go tests on every push / PR
  release.yml    — GoReleaser (binaries) + npm publish (wrapper)
  demo.yml       — regenerate demo GIFs on change
```

## Releasing a new version

Version is sourced from `package.json`. Running `npm version` syncs it to `main.go` automatically.

```bash
# Patch release (2.0.0 → 2.0.1)
npm version patch

# Minor release (2.0.0 → 2.1.0)
npm version minor

# Major release (2.0.0 → 3.0.0)
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
