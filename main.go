package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "2.0.0"

func printHelp() {
	fmt.Printf(`Claude Session Cleaner v%s  —  ePlus.DEV

Interactive TUI to safely delete selected Claude Code project session logs.

Usage:
  claude-cleaner
  claude-cleaner --claude-dir <path>
  claude-cleaner --help
  claude-cleaner --version

Options:
  --claude-dir <path>   Custom Claude config directory (default: ~/.claude)
  -h, --help            Show help
  -v, --version         Show version

Key bindings:
  ↑/↓ or j/k   Navigate
  space         Toggle selection
  a             Select / deselect all
  enter         Confirm selection
  esc           Go back
  q / ctrl+c    Quit

Safety:
  Only session folders inside ~/.claude/projects are deleted.
  Source code directories are never touched.
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
	var helpFlag, versionFlag bool

	flag.StringVar(&claudeDirFlag, "claude-dir", "", "Custom Claude config directory")
	flag.BoolVar(&helpFlag, "help", false, "Show help")
	flag.BoolVar(&helpFlag, "h", false, "Show help")
	flag.BoolVar(&versionFlag, "version", false, "Show version")
	flag.BoolVar(&versionFlag, "v", false, "Show version")
	flag.Parse()

	if versionFlag {
		fmt.Println(version)
		return
	}
	if helpFlag {
		printHelp()
		return
	}

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

	p := tea.NewProgram(newModel(claudeDir, projectsDir), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
