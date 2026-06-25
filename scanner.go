package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Session struct {
	Index        int
	Name         string
	Path         string
	ProjectPath  string // actual working dir from ~/.claude.json; empty if unknown
	Modified     time.Time
	Size         int64
	TotalTokens  int64
	HasTokenData bool // false = token fields absent in ~/.claude.json (show "—")
	HasData      bool // false = directory absent or empty (no session files found)
}

// projectEntry mirrors the token fields stored per-project in ~/.claude.json.
type projectEntry struct {
	LastTotalInputTokens              *int64 `json:"lastTotalInputTokens"`
	LastTotalOutputTokens             *int64 `json:"lastTotalOutputTokens"`
	LastTotalCacheCreationInputTokens *int64 `json:"lastTotalCacheCreationInputTokens"`
	LastTotalCacheReadInputTokens     *int64 `json:"lastTotalCacheReadInputTokens"`
}

func (e projectEntry) total() int64 {
	var n int64
	if e.LastTotalInputTokens != nil {
		n += *e.LastTotalInputTokens
	}
	if e.LastTotalOutputTokens != nil {
		n += *e.LastTotalOutputTokens
	}
	if e.LastTotalCacheCreationInputTokens != nil {
		n += *e.LastTotalCacheCreationInputTokens
	}
	if e.LastTotalCacheReadInputTokens != nil {
		n += *e.LastTotalCacheReadInputTokens
	}
	return n
}

func (e projectEntry) hasAnyField() bool {
	return e.LastTotalInputTokens != nil ||
		e.LastTotalOutputTokens != nil ||
		e.LastTotalCacheCreationInputTokens != nil ||
		e.LastTotalCacheReadInputTokens != nil
}

// normalizePath lowercases and normalises separators so Windows paths are
// compared case-insensitively (d:/Foo and D:\foo → d:/foo).
func normalizePath(p string) string {
	return strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
}

// encodePath converts an actual project path to the hashed directory name
// that Claude Code uses under ~/.claude/projects/.
func encodePath(path string) string {
	r := strings.NewReplacer(":", "-", "/", "-", "\\", "-")
	return strings.ToLower(r.Replace(path))
}

// projectStats reads a flat project directory: returns file size sum and most
// recent file mtime. No recursive walk — session files are always at top level.
func projectStats(dirPath string) (size int64, modified time.Time) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		size += info.Size()
		if info.ModTime().After(modified) {
			modified = info.ModTime()
		}
	}
	return
}

// deduplicateProjects merges duplicate project paths that differ only in
// case (Windows). Keeps the entry with the higher token total.
func deduplicateProjects(raw map[string]projectEntry) map[string]projectEntry {
	type best struct {
		path  string
		entry projectEntry
		total int64
	}
	seen := make(map[string]best, len(raw))
	for path, entry := range raw {
		norm := normalizePath(path)
		total := entry.total()
		if b, ok := seen[norm]; !ok || total > b.total {
			seen[norm] = best{path: path, entry: entry, total: total}
		}
	}
	out := make(map[string]projectEntry, len(seen))
	for _, b := range seen {
		out[b.path] = b.entry
	}
	return out
}

// scanSessions reads projects from claudeJSONPath (primary source of truth).
// Falls back to scanning projectsDir directly when claudeJSONPath is absent
// (e.g. custom --claude-dir for demos).
func scanSessions(claudeJSONPath, projectsDir string) ([]Session, error) {
	data, err := os.ReadFile(claudeJSONPath)
	if os.IsNotExist(err) {
		return scanFromDir(projectsDir)
	}
	if err != nil {
		return nil, err
	}

	var cfg struct {
		Projects map[string]projectEntry `json:"projects"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil || len(cfg.Projects) == 0 {
		return scanFromDir(projectsDir)
	}

	projects := deduplicateProjects(cfg.Projects)

	ch := make(chan Session, len(projects))
	var wg sync.WaitGroup

	for projectPath, entry := range projects {
		wg.Add(1)
		go func(projPath string, e projectEntry) {
			defer wg.Done()
			encoded := encodePath(projPath)
			dirPath := filepath.Join(projectsDir, encoded)
			size, modified := projectStats(dirPath)

			ch <- Session{
				Name:         encoded,
				Path:         dirPath,
				ProjectPath:  projPath,
				Modified:     modified,
				Size:         size,
				TotalTokens:  e.total(),
				HasTokenData: e.hasAnyField(),
				HasData:      !modified.IsZero(),
			}
		}(projectPath, entry)
	}

	wg.Wait()
	close(ch)

	var sessions []Session
	for s := range ch {
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})
	for i := range sessions {
		sessions[i].Index = i + 1
	}

	return sessions, nil
}

// scanFromDir is the fallback: enumerate subdirectories of projectsDir directly.
// Token data is unavailable without ~/.claude.json.
func scanFromDir(projectsDir string) ([]Session, error) {
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	ch := make(chan Session, len(entries))
	var wg sync.WaitGroup

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wg.Add(1)
		go func(e os.DirEntry) {
			defer wg.Done()
			dirPath := filepath.Join(projectsDir, e.Name())
			size, modified := projectStats(dirPath)
			ch <- Session{
				Name:     e.Name(),
				Path:     dirPath,
				Modified: modified,
				Size:     size,
				HasData:  !modified.IsZero(),
			}
		}(entry)
	}

	wg.Wait()
	close(ch)

	var sessions []Session
	for s := range ch {
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})
	for i := range sessions {
		sessions[i].Index = i + 1
	}

	return sessions, nil
}

// DetectClaudeCLI returns the claude CLI version string, or empty string if not found.
func DetectClaudeCLI() string {
	path, err := exec.LookPath("claude")
	if err != nil {
		return ""
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return "found"
	}
	return strings.TrimSpace(string(out))
}

// smartDelete tries claude project purge first (when CLI is available and
// ProjectPath is known), then falls back to direct directory removal if the
// folder still exists afterwards.
func smartDelete(s Session, projectsDir string) error {
	if s.ProjectPath != "" {
		if _, err := exec.LookPath("claude"); err == nil {
			cmd := exec.Command("claude", "project", "purge", "-y", s.ProjectPath)
			_ = cmd.Run() // ignore error — check folder next
		}
	}
	// Folder already gone (claude handled it) — success.
	if _, err := os.Stat(s.Path); os.IsNotExist(err) {
		return nil
	}
	return safeRemove(projectsDir, s.Path)
}

// RunDelete executes deletion for the given sessions snapshot.
// selected is a snapshot (caller must deep-copy before passing to avoid races).
// If all sessions are selected and claude CLI is available, uses --all for efficiency.
func RunDelete(sessions []Session, selected map[int]bool, projectsDir string) (deleted, failed []string) {
	// Check if all sessions are selected
	allSelected := len(sessions) > 0
	for _, s := range sessions {
		if !selected[s.Index] {
			allSelected = false
			break
		}
	}

	if allSelected {
		if _, err := exec.LookPath("claude"); err == nil {
			cmd := exec.Command("claude", "project", "purge", "--all", "-y")
			if cmd.Run() == nil {
				// Verify each folder; clean up any that remain
				for _, s := range sessions {
					if _, statErr := os.Stat(s.Path); os.IsNotExist(statErr) {
						deleted = append(deleted, s.Name)
					} else {
						if err := safeRemove(projectsDir, s.Path); err != nil {
							failed = append(failed, s.Name)
						} else {
							deleted = append(deleted, s.Name)
						}
					}
				}
				return
			}
			// --all failed, fall through to per-project
		}
	}

	for _, s := range sessions {
		if !selected[s.Index] {
			continue
		}
		if err := smartDelete(s, projectsDir); err != nil {
			failed = append(failed, s.Name)
		} else {
			deleted = append(deleted, s.Name)
		}
	}
	return
}

func safeRemove(projectsDir, targetPath string) error {
	rel, err := filepath.Rel(filepath.Clean(projectsDir), filepath.Clean(targetPath))
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if rel == "." ||
		rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
		strings.Contains(rel, string(filepath.Separator)) {
		return fmt.Errorf("refusing to delete path outside projects directory")
	}
	return os.RemoveAll(targetPath)
}

func formatSize(b int64) string {
	const (
		gb = 1 << 30
		mb = 1 << 20
		kb = 1 << 10
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/gb)
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/mb)
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/kb)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatTokens(n int64) string {
	const (
		e = 1_000_000_000_000_000_000 // exa  (int64 max ~9.2E)
		p = 1_000_000_000_000_000     // peta
		t = 1_000_000_000_000         // tera
		b = 1_000_000_000             // billion
		m = 1_000_000                 // million
		k = 1_000                     // kilo
	)
	switch {
	case n >= e:
		return fmt.Sprintf("%.1fE", float64(n)/e)
	case n >= p:
		return fmt.Sprintf("%.1fP", float64(n)/p)
	case n >= t:
		return fmt.Sprintf("%.1fT", float64(n)/t)
	case n >= b:
		return fmt.Sprintf("%.1fB", float64(n)/b)
	case n >= m:
		return fmt.Sprintf("%.1fM", float64(n)/m)
	case n >= k:
		return fmt.Sprintf("%.1fK", float64(n)/k)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func humanTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < 2*time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hr ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
