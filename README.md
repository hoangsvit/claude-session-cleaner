# Claude Session Cleaner

[![CI](https://github.com/hoangsvit/claude-session-cleaner/actions/workflows/ci.yml/badge.svg)](https://github.com/hoangsvit/claude-session-cleaner/actions/workflows/ci.yml)
[![Release](https://github.com/hoangsvit/claude-session-cleaner/actions/workflows/release.yml/badge.svg)](https://github.com/hoangsvit/claude-session-cleaner/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Claude Session Cleaner** is an interactive terminal UI — built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) — that inspects Claude Code project session history, displays disk usage, and safely deletes only the sessions you select.

It runs on Windows, macOS, and Linux.

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
- Multi-select with space bar, range select with `a`.
- Requires an explicit `DELETE` confirmation.
- Rejects deletion paths outside the Claude `projects` directory.
- Concurrent filesystem scanning.
- Supports custom Claude configuration directories.
- Works on Windows, macOS, and Linux.

## What it deletes

The tool only deletes project session folders directly inside:

```text
~/.claude/projects
```

If `CLAUDE_CONFIG_DIR` is configured, it uses:

```text
$CLAUDE_CONFIG_DIR/projects
```

These folders contain Claude Code session and conversation history. The tool
does not delete your actual source code directories.

Before deletion, it:

1. Lists every detected project session with its size and modification time.
2. Shows the exact folders selected for deletion.
3. Requires the user to type `DELETE`.
4. Rejects deletion paths outside the Claude `projects` directory.

Deleted session history cannot be restored by this tool.

## Requirements

- Go 1.22 or newer (to build from source)
- Or download a pre-built binary from [Releases](https://github.com/hoangsvit/claude-session-cleaner/releases)

The CI test suite runs on:

| Operating system | Runner |
| --- | --- |
| Windows | `windows-latest` |
| macOS | `macos-latest` |
| Linux | `ubuntu-latest` |

## Install

### Download pre-built binary

Go to [Releases](https://github.com/hoangsvit/claude-session-cleaner/releases) and download the archive for your platform.

### Install with Go

```bash
go install github.com/hoangsvit/claude-session-cleaner@latest
```

### Build from source

```bash
git clone https://github.com/hoangsvit/claude-session-cleaner.git
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

## Command options

```text
claude-session-cleaner [options]

--claude-dir <path>   Use a custom Claude configuration directory
-h, --help            Show help
-v, --version         Show version
```

## Key bindings

| Key | Action |
| --- | --- |
| `↑` / `↓` or `j` / `k` | Navigate |
| `space` | Toggle selection |
| `a` | Select / deselect all |
| `enter` | Confirm selection |
| `esc` | Go back |
| `q` / `ctrl+c` | Quit |

## Configure a custom Claude directory

The `--claude-dir` option has the highest priority. If omitted, the tool uses
`CLAUDE_CONFIG_DIR`. Otherwise it falls back to `~/.claude`.

Windows PowerShell:

```powershell
$env:CLAUDE_CONFIG_DIR = "D:\ClaudeData"
claude-session-cleaner
```

macOS or Linux:

```bash
export CLAUDE_CONFIG_DIR="/mnt/data/claude"
claude-session-cleaner
```

## Troubleshooting

### Claude directory was not found

Confirm that Claude Code has been run at least once and that the displayed
Claude directory is correct:

```bash
claude-session-cleaner --claude-dir "/correct/path/.claude"
```

### Permission denied

Run the command as the same operating-system user that owns the Claude
configuration directory.

## Development

Build:

```bash
go build -v ./...
```

Run tests:

```bash
go test -v ./...
```

Smoke test:

```bash
go run . --version
```

## Continuous integration

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs tests for Go 1.22,
1.23, and 1.24 on Windows, macOS, and Linux for every push and pull request.

## Release

[`.github/workflows/release.yml`](.github/workflows/release.yml) uses
[GoReleaser](https://goreleaser.com) to build cross-platform binaries and
create a GitHub Release when a tag matching `v*` is pushed.

```bash
git tag v2.0.0
git push --follow-tags
```

GoReleaser builds for Linux, macOS, and Windows on amd64 and arm64.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, build instructions, project structure, and release process.

## License

MIT
