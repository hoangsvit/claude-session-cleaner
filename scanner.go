package main

import (
	"bufio"
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
	HasTokenData bool // false = no token data in ~/.claude.json or session .jsonl files (show "—")
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

// jsonlTokens holds just the four token fields from message.usage in a .jsonl line.
type jsonlTokens struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// scanProjectTokens sums token usage from all top-level .jsonl session files in
// dirPath. Returns (total, hasData); hasData is true when at least one usage
// record was found.
func scanProjectTokens(dirPath string) (int64, bool) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, false
	}
	var total int64
	var hasData bool
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		f, err := os.Open(filepath.Join(dirPath, e.Name()))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20) // 1 MB per line – handles large tool outputs
		for sc.Scan() {
			var row struct {
				Type    string `json:"type"`
				Message struct {
					Usage *jsonlTokens `json:"usage"`
				} `json:"message"`
			}
			if json.Unmarshal(sc.Bytes(), &row) != nil {
				continue
			}
			if row.Type != "assistant" || row.Message.Usage == nil {
				continue
			}
			u := row.Message.Usage
			total += u.InputTokens + u.OutputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
			hasData = true
		}
		f.Close()
	}
	return total, hasData
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

			totalTokens := e.total()
			hasTokenData := e.hasAnyField()
			if !hasTokenData {
				totalTokens, hasTokenData = scanProjectTokens(dirPath)
			}
			ch <- Session{
				Name:         encoded,
				Path:         dirPath,
				ProjectPath:  projPath,
				Modified:     modified,
				Size:         size,
				TotalTokens:  totalTokens,
				HasTokenData: hasTokenData,
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
// Token data is read from .jsonl session files since ~/.claude.json is absent.
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
			totalTokens, hasTokenData := scanProjectTokens(dirPath)
			ch <- Session{
				Name:         e.Name(),
				Path:         dirPath,
				Modified:     modified,
				Size:         size,
				TotalTokens:  totalTokens,
				HasTokenData: hasTokenData,
				HasData:      !modified.IsZero(),
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

// ── Category cleanup ──────────────────────────────────────────────────────────

// Category represents a cleanable data directory or special cleanup operation.
type Category struct {
	Key       string
	Label     string
	Path      string
	Size      int64
	FileCount int
	Exists    bool
	Special   bool // JSON orphan cleanup / history trim — not a plain RemoveAll
}

var cleanableDirs = []struct {
	key   string
	label string
	dir   string
}{
	{"debug", "Debug logs", "debug"},
	{"file-history", "File history", "file-history"},
	{"telemetry", "Telemetry", "telemetry"},
	{"shell-snapshots", "Shell snapshots", "shell-snapshots"},
	{"transcripts", "Transcripts", "transcripts"},
	{"todos", "Todos", "todos"},
	{"plans", "Plans", "plans"},
	{"usage-data", "Usage data", "usage-data"},
	{"tasks", "Tasks", "tasks"},
	{"paste-cache", "Paste cache", "paste-cache"},
	{"plugins", "Plugins cache", "plugins"},
}

// dirSizeCount returns total size and file count for a flat directory.
func dirSizeCount(path string) (size int64, count int) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !e.IsDir() {
			size += info.Size()
			count++
		}
	}
	return
}

// scanCategories returns cleanable categories inside claudeDir plus special entries.
func scanCategories(claudeDir string) []Category {
	var result []Category

	for _, cat := range cleanableDirs {
		path := filepath.Join(claudeDir, cat.dir)
		_, statErr := os.Stat(path)
		size, count := dirSizeCount(path)
		result = append(result, Category{
			Key:       cat.key,
			Label:     cat.label,
			Path:      path,
			Size:      size,
			FileCount: count,
			Exists:    statErr == nil,
		})
	}

	// Config backups: ~/.claude.json.backup* (sibling of claudeDir)
	parentDir := filepath.Dir(claudeDir)
	backups, _ := filepath.Glob(filepath.Join(parentDir, ".claude.json.backup*"))
	var backupSize int64
	for _, b := range backups {
		if info, err := os.Stat(b); err == nil {
			backupSize += info.Size()
		}
	}
	result = append(result, Category{
		Key:       "config-backups",
		Label:     "Config backups (.claude.json.backup*)",
		Path:      parentDir,
		Size:      backupSize,
		FileCount: len(backups),
		Exists:    len(backups) > 0,
	})

	// Orphan project entries in ~/.claude.json
	claudeJSONPath := filepath.Join(parentDir, ".claude.json")
	orphanCount, jsonExists := countOrphanEntries(claudeJSONPath)
	result = append(result, Category{
		Key:       "json-orphans",
		Label:     "Orphan project entries in ~/.claude.json",
		Path:      claudeJSONPath,
		Size:      0,
		FileCount: orphanCount,
		Exists:    jsonExists && orphanCount > 0,
		Special:   true,
	})

	// History trim: ~/.claude/history.jsonl
	histPath := filepath.Join(claudeDir, "history.jsonl")
	histSize := int64(0)
	histExists := false
	if info, err := os.Stat(histPath); err == nil {
		histSize = info.Size()
		histExists = histSize > 0
	}
	result = append(result, Category{
		Key:     "history-trim",
		Label:   "Trim history.jsonl (keep last 500 lines)",
		Path:    histPath,
		Size:    histSize,
		Exists:  histExists,
		Special: true,
	})

	return result
}

// countOrphanEntries returns the number of project paths in ~/.claude.json
// whose directories no longer exist, plus whether the file was readable.
func countOrphanEntries(claudeJSONPath string) (count int, exists bool) {
	data, err := os.ReadFile(claudeJSONPath)
	if err != nil {
		return 0, false
	}
	var cfg struct {
		Projects map[string]json.RawMessage `json:"projects"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0, true
	}
	for path := range cfg.Projects {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			count++
		}
	}
	return count, true
}

// cleanCategory executes the cleanup operation for a category.
func cleanCategory(cat Category, claudeDir string) error {
	switch cat.Key {
	case "json-orphans":
		parentDir := filepath.Dir(claudeDir)
		return cleanOrphanEntries(filepath.Join(parentDir, ".claude.json"))
	case "history-trim":
		return trimHistory(cat.Path, 500)
	case "config-backups":
		parentDir := filepath.Dir(claudeDir)
		backups, _ := filepath.Glob(filepath.Join(parentDir, ".claude.json.backup*"))
		for _, b := range backups {
			_ = os.Remove(b)
		}
		return nil
	default:
		if cat.Path == "" || !cat.Exists {
			return nil
		}
		return os.RemoveAll(cat.Path)
	}
}

// cleanOrphanEntries removes project entries from ~/.claude.json where the
// project directory no longer exists, writing back atomically.
func cleanOrphanEntries(claudeJSONPath string) error {
	data, err := os.ReadFile(claudeJSONPath)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	projectsRaw, ok := raw["projects"]
	if !ok {
		return nil
	}
	var projects map[string]json.RawMessage
	if err := json.Unmarshal(projectsRaw, &projects); err != nil {
		return err
	}
	changed := false
	for path := range projects {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			delete(projects, path)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	newProjects, err := json.Marshal(projects)
	if err != nil {
		return err
	}
	raw["projects"] = newProjects
	newData, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := claudeJSONPath + ".tmp"
	if err := os.WriteFile(tmpPath, newData, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, claudeJSONPath)
}

// trimHistory keeps only the last keepLines lines of histPath, writing atomically.
func trimHistory(histPath string, keepLines int) error {
	data, err := os.ReadFile(histPath)
	if err != nil {
		return err
	}
	content := strings.TrimRight(string(data), "\n")
	lines := strings.Split(content, "\n")
	if len(lines) <= keepLines {
		return nil
	}
	kept := lines[len(lines)-keepLines:]
	newContent := strings.Join(kept, "\n") + "\n"
	tmpPath := histPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, histPath)
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
