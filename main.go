package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "1.0.1"

func printHelp() {
	fmt.Printf(`Claude Cleaner v%s  —  ePlus.DEV

Interactive TUI to safely delete selected Claude Code project session logs.

Usage:
  claude-cleaner
  claude-cleaner --claude-dir <path>
  claude-cleaner --help
  claude-cleaner --version

Options:
  --claude-dir <path>   Custom Claude config directory (default: ~/.claude)
  --dry-run             Preview what would be deleted without touching any files
  --mock-update         Simulate a newer version available (for testing)
  -h, --help            Show help
  -v, --version         Show version

Key bindings:
  ↑/↓ or j/k   Navigate list
  g / G         Jump to top / bottom
  space         Toggle selection
  a             Select / deselect all (visible)
  n             Unselect all
  o             Select all orphaned projects (○)
  d             Reset sort / filter / search / selection to defaults
  enter         Confirm — show delete screen (when items selected)
  p             Purge selected (confirm screen)
  x             Force-purge item at cursor — no confirm
  s             Cycle sort: recent → size → tokens → name
  f             Cycle filter: all → has data → orphaned
  e             Cycle expiry: off → 7d → 14d → 30d → 60d → 90d
  c             Open category cleanup (debug logs, telemetry, history…)
  /             Search by project name / path
  r             Rescan / refresh project list
  u             Update claude-cleaner in-place (when update available)
  ?             Show key bindings
  esc           Go back / clear search / cancel
  q / ctrl+c    Quit (any screen)

Safety:
  Only session folders inside ~/.claude/projects are deleted.
  Source code directories are never touched.
  --dry-run shows exactly what would be deleted without modifying anything.
`, version)
}

func resolveClaudeDir(dir string) (string, error) {
	if strings.TrimSpace(dir) != "" {
		return filepath.Clean(dir), nil
	}
	if env := os.Getenv("CLAUDE_CONFIG_DIR"); strings.TrimSpace(env) != "" {
		return filepath.Clean(env), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}

func main() {
	var claudeDirFlag string
	var helpFlag, versionFlag, mockUpdateFlag, dryRunFlag bool

	flag.StringVar(&claudeDirFlag, "claude-dir", "", "Custom Claude config directory")
	flag.BoolVar(&helpFlag, "help", false, "Show help")
	flag.BoolVar(&helpFlag, "h", false, "Show help")
	flag.BoolVar(&versionFlag, "version", false, "Show version")
	flag.BoolVar(&versionFlag, "v", false, "Show version")
	flag.BoolVar(&mockUpdateFlag, "mock-update", false, "Simulate a newer version available (for testing)")
	flag.BoolVar(&dryRunFlag, "dry-run", false, "Preview deletions without modifying any files")
	flag.Parse()

	if versionFlag {
		fmt.Println(version)
		return
	}
	if helpFlag {
		printHelp()
		return
	}

	cleanupOldBinary()

	claudeDir, err := resolveClaudeDir(claudeDirFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(claudeDir); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot find Claude directory: %s\n", claudeDir)
		os.Exit(1)
	}

	projectsDir := filepath.Join(claudeDir, "projects")
	if _, err := os.Stat(projectsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot find Claude projects directory: %s\n", projectsDir)
		os.Exit(1)
	}

	// ~/.claude.json is one level above claudeDir (~/.claude → ~)
	claudeJSONPath := filepath.Join(filepath.Dir(claudeDir), ".claude.json")

	prefs := loadPrefs(claudeDir)

	m := newModel(claudeDir, claudeJSONPath, projectsDir)
	m.sortMode = sortMode(prefs.SortMode)
	m.filterMode = filterMode(prefs.FilterMode)
	m.expiryDays = prefs.ExpiryDays
	// restore expiryIdx so cycling works correctly
	for i, v := range []int{0, 7, 14, 30, 60, 90} {
		if v == prefs.ExpiryDays {
			m.expiryIdx = i
			break
		}
	}
	if dryRunFlag {
		m.dryRun = true
	}
	if mockUpdateFlag {
		m.latestVersion = "99.0.0"
		m.hasUpdate = true
		m.updateChecked = true
		m.pendingUpdatePrompt = true
		m.skipUpdateCheck = true
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fm, ok := finalModel.(model); ok && fm.restartAfterUpdate {
		exe, exeErr := os.Executable()
		if exeErr == nil && !isTempBuild(exe) {
			// Installed binary: re-exec self without --mock-update
			var args []string
			for _, a := range os.Args[1:] {
				if a != "--mock-update" {
					args = append(args, a)
				}
			}
			cmd := exec.Command(exe, args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if cmd.Start() == nil {
				return
			}
		}
		// Fallback: dev mode (go run .) — recompile and restart
		cmd := exec.Command("go", "run", ".")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if cmd.Start() != nil {
			fmt.Println("Update complete. Restart with: claude-cleaner")
		}
	}
}

// isTempBuild returns true when running via "go run" (binary lives in a temp dir).
func isTempBuild(exe string) bool {
	return strings.Contains(exe, "go-build") ||
		strings.Contains(exe, string(os.PathSeparator)+"T"+string(os.PathSeparator)) // macOS /var/folders/T/
}

// prepareWindowsUpdate renames the running exe to .old on Windows so npm can
// overwrite it (Windows locks running executables from being replaced in-place).
// The .old file is cleaned up on the next startup via cleanupOldBinary.
func prepareWindowsUpdate() {
	if runtime.GOOS != "windows" {
		return
	}
	exe, err := os.Executable()
	if err != nil || isTempBuild(exe) {
		return
	}
	_ = os.Rename(exe, exe+".old")
}

// cleanupOldBinary removes any leftover .old binary from a previous update.
func cleanupOldBinary() {
	exe, err := os.Executable()
	if err == nil {
		_ = os.Remove(exe + ".old")
	}
}
