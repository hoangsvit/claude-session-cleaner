# Claude Session Cleaner

Interactive CLI for safely finding and deleting selected Claude Code project
session folders.

It supports Windows, macOS, and Linux with Node.js 18 or newer.

<img width="1125" height="621" alt="Claude Session Cleaner preview" src="https://github.com/user-attachments/assets/89094ee1-2a41-4aec-b47e-a14425b57d4c" />

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

- Node.js 18, 20, 22, or newer
- npm, included with Node.js
- A terminal that supports interactive input

The GitHub Actions test suite runs on:

| Operating system | Runner |
| --- | --- |
| Windows | `windows-latest` |
| macOS | `macos-latest` |
| Linux | `ubuntu-latest` |

## Install and run

### Run without installing

```bash
npx claude-session-cleaner
```

### Install globally

```bash
npm install --global claude-session-cleaner
claude-session-cleaner
```

### Run from the source code

```bash
git clone https://github.com/mic1/claude-session-cleaner.git
cd claude-session-cleaner
npm install
npm start
```

Platform helper scripts are also included.

Windows Command Prompt or PowerShell:

```powershell
.\run.cmd
```

macOS or Linux:

```bash
chmod +x ./run.sh
./run.sh
```

Both scripts forward CLI arguments. For example:

```powershell
.\run.cmd --claude-dir "D:\custom\.claude"
```

```bash
./run.sh --claude-dir "/custom/path/.claude"
```

## Command options

```text
claude-session-cleaner [options]

--claude-dir <path>   Use a custom Claude configuration directory
-h, --help            Show help
-v, --version         Show the installed version
```

Examples:

```bash
claude-session-cleaner
claude-session-cleaner --help
claude-session-cleaner --version
claude-session-cleaner --claude-dir "/path/to/.claude"
```

On Windows:

```powershell
claude-session-cleaner --claude-dir "C:\Users\YourName\.claude"
```

## Configure a custom Claude directory

The `--claude-dir` option has the highest priority. If it is omitted, the tool
uses `CLAUDE_CONFIG_DIR`. Otherwise it falls back to `~/.claude`.

Windows PowerShell:

```powershell
$env:CLAUDE_CONFIG_DIR = "D:\ClaudeData"
claude-session-cleaner
```

Windows Command Prompt:

```bat
set CLAUDE_CONFIG_DIR=D:\ClaudeData
claude-session-cleaner
```

macOS or Linux:

```bash
export CLAUDE_CONFIG_DIR="/mnt/data/claude"
claude-session-cleaner
```

## Selecting sessions

At the selection prompt, enter:

| Input | Action |
| --- | --- |
| `1` | Select one session |
| `1,3,5` | Select several sessions |
| `1-3` | Select a range |
| `3-1` | Select a reversed range |
| `all` or `a` | Select all listed sessions |
| `r` | Refresh the list |
| `q` | Quit without deleting |

Example:

```text
Your selection: 1,3-5
Type "DELETE" to confirm: DELETE
```

Any confirmation other than the exact uppercase word `DELETE` cancels the
operation.

## Troubleshooting

### Claude directory was not found

Confirm that Claude Code has been run at least once and that the displayed
Claude directory is correct:

```bash
claude-session-cleaner --claude-dir "/correct/path/.claude"
```

### Permission denied

Run the command as the same operating-system user that owns the Claude
configuration directory. Avoid running as administrator or root unless that
directory genuinely requires it.

### `node` or `npm` was not found

Install a supported Node.js version, then open a new terminal and verify:

```bash
node --version
npm --version
```

## Development

Install dependencies:

```bash
npm ci
```

Run unit tests:

```bash
npm test
```

Run the CLI version smoke test:

```bash
npm run test:smoke
```

Inspect the npm package contents:

```bash
npm run pack:dry
```

## Continuous integration

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs unit and smoke tests
for Node.js 18, 20, and 22 on Windows, macOS, and Linux for pushes and pull
requests.

## Creating a GitHub Release

[`.github/workflows/release.yml`](.github/workflows/release.yml) creates a
GitHub Release when a tag matching `v*` is pushed. It first tests the package on
Windows, macOS, and Linux.

The tag must match the version in `package.json`. For example, version `1.1.0`
must use tag `v1.1.0`.

Release steps:

```bash
npm version patch
git push
git push origin v1.0.1
```

Use `minor` or `major` instead of `patch` when appropriate:

```bash
npm version minor
npm version major
```

After all platform tests pass, the workflow:

1. Verifies that the tag matches `package.json`.
2. Builds the npm `.tgz` archive.
3. Creates a GitHub Release with generated release notes.
4. Uploads the archive to the release.

Publishing to npm remains a separate explicit operation:

```bash
npm login
npm publish
```

## License

MIT
