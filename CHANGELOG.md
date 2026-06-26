# Changelog

## Unreleased

- Token column now falls back to summing `message.usage` from session `.jsonl` files when `~/.claude.json` does not contain `lastTotal*` token fields (common on newer Claude Code installs).
- UI Project column shows only the last folder name (e.g. `g-front`) instead of the full path; full path is still used internally for correct deletion.
- Bumped minimum Go version to 1.25 (go.mod).
- Updated CI matrix to Go 1.25 / 1.26 across Windows, macOS, and Linux.
- Updated all workflows (ci, demo, release) to Go 1.25.
- Added Snyk security scanning workflow (push / PR + weekly schedule).
- Upgraded dependencies to fix HIGH/MEDIUM Snyk findings:
  - `golang.org/x/text` v0.3.8 → v0.38.0 (CWE-1327)
  - `golang.org/x/sys` v0.27.0 → v0.46.0 (CWE-190)
  - `github.com/charmbracelet/bubbletea` v1.2.4 → v1.3.10
  - `github.com/charmbracelet/bubbles` v0.20.0 → v1.0.0
  - `github.com/charmbracelet/lipgloss` v1.0.0 → v1.1.0
- Restructured README: install section promoted, dev content moved to CONTRIBUTING.md.
- Improved asynchronous directory scanning and deletion safety.
- Added automated tests on Windows, macOS, and Linux.
- Added OIDC-based npm publishing and tag-based GitHub Release automation.
- Added optional `NPM_TOKEN` bootstrap support for the first npm publication.
- Removed generated npm `always-auth` configuration from the release workflow.
- Expanded installation, usage, troubleshooting, and release documentation.

## 1.0.0

- Initial npm-ready release.
- Interactive project session selection.
- Cross-platform support for Windows, macOS, and Linux.
- Supports `--claude-dir`, `--help`, and `--version`.
