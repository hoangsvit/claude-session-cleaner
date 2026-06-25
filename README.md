# Claude Session Cleaner

[![CI](https://github.com/ePlus-DEV/claude-session-cleaner/actions/workflows/ci.yml/badge.svg)](https://github.com/ePlus-DEV/claude-session-cleaner/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ePlus-DEV/claude-session-cleaner)](https://github.com/ePlus-DEV/claude-session-cleaner/releases)
[![npm version](https://img.shields.io/npm/v/claude-session-cleaner.svg)](https://www.npmjs.com/package/claude-session-cleaner)
[![Go version](https://img.shields.io/github/go-mod/go-version/ePlus-DEV/claude-session-cleaner)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/ePlus-DEV/claude-session-cleaner)](https://goreportcard.com/report/github.com/ePlus-DEV/claude-session-cleaner)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Claude Session Cleaner** is an interactive terminal UI — built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) — that inspects Claude Code project session history, displays disk usage, and safely deletes only the sessions you select.

Runs on Windows, macOS, and Linux. No runtime required when using a pre-built binary.

![Full demo](demo/full.gif)

## Demos

| Scenario | Preview |
| --- | --- |
| `--help` | ![Help](demo/help.gif) |
| Delete a session | ![Full flow](demo/full.gif) |
| Cancel confirmation | ![Cancel](demo/cancel.gif) |

## Features

- Finds Claude Code project sessions automatically.
- Shows session name, last modification time, and disk usage.
- Multi-select with `space`, select all with `a`.
- Requires explicit `DELETE` confirmation before any deletion.
- Rejects paths outside the Claude `projects` directory.
- Concurrent filesystem scanning.
- Supports custom Claude configuration directories via `--claude-dir` or `CLAUDE_CONFIG_DIR`.

## What it deletes

Only project session folders directly inside `~/.claude/projects` (or `$CLAUDE_CONFIG_DIR/projects`).

These folders contain Claude Code session and conversation history. Source code directories are never touched.

Before deleting, the tool:

1. Lists every session with size and last modified time.
2. Shows the exact folders selected for deletion.
3. Requires you to type `DELETE` to confirm.
4. Validates that all paths are inside the Claude projects directory.

Deleted session history cannot be restored by this tool.

## Install

### Run without installing

```bash
npx claude-session-cleaner
```

### Install globally

```bash
npm install --global claude-session-cleaner
claude-session-cleaner
```

> The npm package is a thin wrapper. On install it automatically downloads the correct pre-built binary for your platform from GitHub Releases. No Go required.

### Download binary directly

Go to [Releases](https://github.com/ePlus-DEV/claude-session-cleaner/releases), download the archive for your platform, extract, and run.

| Platform | File |
| --- | --- |
| Linux x64 | `claude-session-cleaner_*_linux_amd64.tar.gz` |
| Linux ARM64 | `claude-session-cleaner_*_linux_arm64.tar.gz` |
| macOS x64 | `claude-session-cleaner_*_darwin_amd64.tar.gz` |
| macOS Apple Silicon | `claude-session-cleaner_*_darwin_arm64.tar.gz` |
| Windows x64 | `claude-session-cleaner_*_windows_amd64.zip` |
| Windows ARM64 | `claude-session-cleaner_*_windows_arm64.zip` |

### Install with Go

```bash
go install github.com/ePlus-DEV/claude-session-cleaner@latest
```

### Build from source

```bash
git clone https://github.com/ePlus-DEV/claude-session-cleaner.git
cd claude-session-cleaner
go build -o claude-session-cleaner .
./claude-session-cleaner
```

## Usage

```bash
claude-session-cleaner
claude-session-cleaner --claude-dir "/path/to/.claude"
claude-session-cleaner --help
claude-session-cleaner --version
```

### Options

```text
--claude-dir <path>   Custom Claude config directory (default: ~/.claude)
-h, --help            Show help
-v, --version         Show version
```

### Key bindings

| Key | Action |
| --- | --- |
| `↑` / `↓` or `j` / `k` | Navigate list |
| `space` | Toggle selection |
| `a` | Select / deselect all |
| `enter` | Confirm selection |
| `esc` | Go back |
| `q` / `ctrl+c` | Quit |

## Configure a custom Claude directory

Priority order: `--claude-dir` > `CLAUDE_CONFIG_DIR` > `~/.claude`

```bash
# macOS / Linux
export CLAUDE_CONFIG_DIR="/mnt/data/claude"
claude-session-cleaner
```

```powershell
# Windows PowerShell
$env:CLAUDE_CONFIG_DIR = "D:\ClaudeData"
claude-session-cleaner
```

## Troubleshooting

**Claude directory not found** — Run Claude Code at least once so the directory is created, or point to the correct path:

```bash
claude-session-cleaner --claude-dir "/correct/path/.claude"
```

**Permission denied** — Run as the same OS user that owns the Claude config directory.

**Binary not found after `npx`** — Try reinstalling:

```bash
npm install --global claude-session-cleaner
```

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for full setup, build, and release instructions.

Quick reference:

```bash
go mod tidy          # install dependencies
go build -v ./...    # build
go test -v ./...     # run tests
go run . --version   # smoke test
```

## CI / CD

| Workflow | Trigger | What it does |
| --- | --- | --- |
| [ci.yml](.github/workflows/ci.yml) | push / PR | Go tests on 1.22, 1.23, 1.24 × Windows, macOS, Linux |
| [release.yml](.github/workflows/release.yml) | push `v*` tag | GoReleaser builds binaries → GitHub Release → npm publish |
| [demo.yml](.github/workflows/demo.yml) | push to main (Go / tape files) | Regenerates demo GIFs via VHS |

### Publishing a release

```bash
npm version patch     # or minor / major
git push --follow-tags
```

`npm version` automatically syncs the version to `main.go` and creates a git tag. Pushing the tag triggers GoReleaser and npm publish.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE) © ePlus.DEV
