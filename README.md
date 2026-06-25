# Claude Session Cleaner

[![npm version](https://img.shields.io/npm/v/claude-session-cleaner.svg)](https://www.npmjs.com/package/claude-session-cleaner)
[![CI](https://github.com/hoangsvit/claude-session-cleaner/actions/workflows/ci.yml/badge.svg)](https://github.com/hoangsvit/claude-session-cleaner/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Claude Session Cleaner is an interactive CLI that inspects Claude Code project
session history, displays disk usage, and safely deletes only the sessions you
select.

It runs on Windows, macOS, and Linux with Node.js 20 or newer.

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
- Supports individual, multiple, range, and all-session selection.
- Requires an explicit `DELETE` confirmation.
- Rejects deletion paths outside the Claude `projects` directory.
- Uses asynchronous, concurrency-limited filesystem scanning.
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

- Node.js 20, 22, 24, or newer
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
git clone https://github.com/hoangsvit/claude-session-cleaner.git
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
for Node.js 20, 22, and 24 on Windows, macOS, and Linux for pushes and pull
requests.

## Automated npm and GitHub release

[`.github/workflows/release.yml`](.github/workflows/release.yml) publishes the
package to npm and creates a GitHub Release when a tag matching `v*` is pushed.
It first tests the package on Windows, macOS, and Linux.

The tag must match the version in `package.json`. For example, version `1.1.0`
must use tag `v1.1.0`.

### Configure npm Trusted Publishing

The workflow uses OpenID Connect (OIDC), so it does not need an `NPM_TOKEN`,
`.env` file, or long-lived GitHub Actions secret.

For an existing npm package:

1. Sign in to [npmjs.com](https://www.npmjs.com/).
2. Open the `claude-session-cleaner` package.
3. Open **Settings**, then locate **Trusted Publisher**.
4. Select **GitHub Actions**.
5. Configure these exact values:

| npm setting | Value |
| --- | --- |
| Organization or user | `hoangsvit` |
| Repository | `claude-session-cleaner` |
| Workflow filename | `release.yml` |
| Environment | Leave empty |
| Allowed action | `npm publish` |

Trusted Publishing can only be configured after the package exists on npm. An
attempt to use OIDC before the first publication usually fails with npm error
`E404` even when the package name is available.

### First publication

Choose one of these bootstrap methods.

Publish from a trusted local machine:

```bash
npm login
npm publish
```

Or publish through GitHub Actions:

1. Create a granular npm access token with read/write permission for packages.
2. Open the GitHub repository.
3. Go to **Settings → Secrets and variables → Actions**.
4. Create a repository secret named `NPM_TOKEN`.
5. Commit the workflow changes and push a new version tag.

Do not simply re-run an older failed job after changing the workflow. GitHub
uses the workflow stored at that job's original tag. For example, after a
failed `v1.0.1` bootstrap release:

```bash
git add .
git commit -m "Fix npm publishing bootstrap"
npm version patch
git push --follow-tags
```

This creates `v1.0.2`, which includes the corrected workflow.

The release workflow passes this optional secret as `NODE_AUTH_TOKEN`. After
the first version exists on npm:

1. Configure Trusted Publishing using the table above.
2. Delete the `NPM_TOKEN` GitHub secret.
3. Use OIDC for all later releases.

Never commit an npm token to `.env`, `.npmrc`, source code, or workflow files.

### `always-auth` warning

The workflow does not generate an npm registry `.npmrc`, so current npm
versions will not receive the deprecated `always-auth` option from
`actions/setup-node`. The npm registry is selected through `publishConfig` in
`package.json`.

### Publish a new version

Release steps:

```bash
npm version patch
git push --follow-tags
```

Use `minor` or `major` instead of `patch` when appropriate:

```bash
npm version minor
npm version major
```

After all platform tests pass, the workflow:

1. Verifies that the tag matches `package.json`.
2. Builds the npm `.tgz` archive.
3. Publishes the package to npm using short-lived OIDC credentials.
4. Creates a GitHub Release with generated release notes.
5. Uploads the archive to the GitHub Release.

If npm publishing fails, the GitHub Release is not created.

## License

MIT
