package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProjectStats(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.txt"), []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	// JSONL with one usage line and one non-usage line
	jsonl := `{"type":"user","message":{"content":"hello"}}
{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50}}}
`
	if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte(jsonl), 0644); err != nil {
		t.Fatal(err)
	}

	size, in, out, mod := projectStats(dir)
	if size == 0 {
		t.Error("size should be > 0")
	}
	if in != 100 {
		t.Errorf("input_tokens want 100, got %d", in)
	}
	if out != 50 {
		t.Errorf("output_tokens want 50, got %d", out)
	}
	if mod.IsZero() {
		t.Error("modified should not be zero")
	}
}

func TestSafeRemove(t *testing.T) {
	root := t.TempDir()

	child := filepath.Join(root, "session-1")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := safeRemove(root, child); err != nil {
		t.Errorf("expected no error for direct child, got: %v", err)
	}

	if err := safeRemove(root, root); err == nil {
		t.Error("expected error when target equals projectsDir")
	}

	if err := safeRemove(root, filepath.Dir(root)); err == nil {
		t.Error("expected error for path above projectsDir")
	}

	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := safeRemove(root, nested); err == nil {
		t.Error("expected error for nested path")
	}
}

func TestScanSessionsFromDir(t *testing.T) {
	projectsDir := t.TempDir()
	names := []string{"proj-a", "proj-b", "proj-c"}
	for _, name := range names {
		dir := filepath.Join(projectsDir, name)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	sessions, err := scanSessions("/nonexistent/.claude.json", projectsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != len(names) {
		t.Errorf("expected %d sessions, got %d", len(names), len(sessions))
	}
	for i, s := range sessions {
		if s.Index != i+1 {
			t.Errorf("session[%d].Index = %d, want %d", i, s.Index, i+1)
		}
		if s.Size == 0 {
			t.Errorf("session[%d].Size should be > 0", i)
		}
	}
}

func TestScanSessionsFromClaudeJSON(t *testing.T) {
	projectsDir := t.TempDir()

	projects := map[string]string{
		"/home/user/my-app":     encodePath("/home/user/my-app"),
		"/home/user/other-proj": encodePath("/home/user/other-proj"),
	}
	for _, encoded := range projects {
		dir := filepath.Join(projectsDir, encoded)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	claudeJSON := `{"projects":{"/home/user/my-app":{},"/home/user/other-proj":{}}}`
	jsonPath := filepath.Join(t.TempDir(), ".claude.json")
	if err := os.WriteFile(jsonPath, []byte(claudeJSON), 0644); err != nil {
		t.Fatal(err)
	}

	sessions, err := scanSessions(jsonPath, projectsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
	for _, s := range sessions {
		if s.ProjectPath == "" {
			t.Errorf("session %q should have ProjectPath set", s.Name)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{1_000_000, "1.0M"},
		{2_500_000, "2.5M"},
	}
	for _, c := range cases {
		if got := formatTokens(c.n); got != c.want {
			t.Errorf("formatTokens(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}
	for _, tt := range tests {
		if got := formatSize(tt.bytes); got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestHumanTime(t *testing.T) {
	if got := humanTime(time.Now().Add(-30 * time.Second)); got != "just now" {
		t.Errorf("got %q, want 'just now'", got)
	}
	if got := humanTime(time.Now().Add(-25 * time.Hour)); got != "yesterday" {
		t.Errorf("got %q, want 'yesterday'", got)
	}
	if got := humanTime(time.Time{}); got != "—" {
		t.Errorf("zero time want '—', got %q", got)
	}
}

func TestEncodePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"d:/laragon/www/dev/claude-session-cleaner", "d--laragon-www-dev-claude-session-cleaner"},
		{"/Users/foo/myproject", "-users-foo-myproject"},
		{"D:\\laragon\\www\\myapp", "d--laragon-www-myapp"},
	}
	for _, c := range cases {
		if got := encodePath(c.in); got != c.want {
			t.Errorf("encodePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("got %q, want 'hello'", got)
	}
	if got := truncate("hello world", 8); got != "hello w…" {
		t.Errorf("got %q, want 'hello w…'", got)
	}
}
