package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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
	InputTokens  int64
	OutputTokens int64
	HasData      bool // false = directory absent or empty (no session files found)
}

// encodePath converts an actual project path to the hashed directory name
// that Claude Code uses under ~/.claude/projects/.
func encodePath(path string) string {
	r := strings.NewReplacer(":", "-", "/", "-", "\\", "-")
	return strings.ToLower(r.Replace(path))
}

// projectStats reads a flat project directory: sums file sizes and parses
// JSONL session files for token usage in one pass. No recursive walk.
func projectStats(dirPath string) (size, inputTokens, outputTokens int64, modified time.Time) {
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
		if strings.HasSuffix(e.Name(), ".jsonl") {
			in, out := countTokensInFile(filepath.Join(dirPath, e.Name()))
			inputTokens += in
			outputTokens += out
		}
	}
	return
}

// countTokensInFile parses a JSONL session file and sums input/output tokens
// from assistant message usage fields. Skips lines without "usage" fast.
func countTokensInFile(path string) (inputTokens, outputTokens int64) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MB per line

	var entry struct {
		Message struct {
			Usage struct {
				InputTokens  int64 `json:"input_tokens"`
				OutputTokens int64 `json:"output_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte(`"usage"`)) {
			continue
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		inputTokens += entry.Message.Usage.InputTokens
		outputTokens += entry.Message.Usage.OutputTokens
	}
	return
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
		Projects map[string]json.RawMessage `json:"projects"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil || len(cfg.Projects) == 0 {
		return scanFromDir(projectsDir)
	}

	ch := make(chan Session, len(cfg.Projects))
	var wg sync.WaitGroup

	for projectPath := range cfg.Projects {
		wg.Add(1)
		go func(projPath string) {
			defer wg.Done()
			encoded := encodePath(projPath)
			dirPath := filepath.Join(projectsDir, encoded)

			size, inputTokens, outputTokens, modified := projectStats(dirPath)

			ch <- Session{
				Name:         encoded,
				Path:         dirPath,
				ProjectPath:  projPath,
				Modified:     modified,
				Size:         size,
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				HasData:      !modified.IsZero(),
			}
		}(projectPath)
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
			size, inputTokens, outputTokens, modified := projectStats(dirPath)
			ch <- Session{
				Name:         e.Name(),
				Path:         dirPath,
				Modified:     modified,
				Size:         size,
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
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
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
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
