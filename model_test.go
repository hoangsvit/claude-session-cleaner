package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func makeTestModel(sessions []Session) model {
	m := newModel("/tmp/.claude", "/tmp/.claude.json", "/tmp/.claude/projects")
	m.sessions = sessions
	m.state = stateList
	return m
}

func fakeSessions(n int) []Session {
	s := make([]Session, n)
	for i := range s {
		s[i] = Session{Index: i + 1, Name: "session" + string(rune('A'+i)), Size: 1024}
	}
	return s
}

// realisticSessions returns sessions resembling real Claude project data:
// mix of HasData/no-data, varied token counts, realistic project paths.
func realisticSessions() []Session {
	ago := func(minutes int) time.Time {
		return time.Now().Add(-time.Duration(minutes) * time.Minute)
	}
	day := func(days int) time.Time {
		return time.Now().AddDate(0, 0, -days)
	}

	return []Session{
		{
			Index: 1, Name: encodePath("/home/user/projects/webapp-frontend"),
			ProjectPath: "/home/user/projects/webapp-frontend",
			Modified: ago(9), Size: 1_887_437,
			TotalTokens: 0, HasTokenData: true, HasData: true,
		},
		{
			Index: 2, Name: encodePath("/home/user/projects/api-service"),
			ProjectPath: "/home/user/projects/api-service",
			Modified: ago(25), Size: 72_704_000,
			TotalTokens: 154200, HasTokenData: true, HasData: true,
		},
		{
			Index: 3, Name: encodePath("/home/user/projects/mobile-app"),
			ProjectPath: "/home/user/projects/mobile-app",
			Modified: day(1), Size: 806_912,
			TotalTokens: 531800, HasTokenData: true, HasData: true,
		},
		{
			Index: 4, Name: encodePath("/home/user/projects/data-pipeline"),
			ProjectPath: "/home/user/projects/data-pipeline",
			Modified: day(25), Size: 194_355,
			TotalTokens: 0, HasTokenData: true, HasData: true,
		},
		{
			Index: 5, Name: encodePath("/home/user/projects/infra-scripts"),
			ProjectPath: "/home/user/projects/infra-scripts",
			Modified: day(25), Size: 419_635,
			TotalTokens: 213100, HasTokenData: true, HasData: true,
		},
		{
			Index: 6, Name: encodePath("/home/user/projects/design-system"),
			ProjectPath: "/home/user/projects/design-system",
			HasData: false, HasTokenData: false,
		},
		{
			Index: 7, Name: encodePath("/home/user/projects/archived-tool"),
			ProjectPath: "/home/user/projects/archived-tool",
			HasData: false, HasTokenData: false,
		},
	}
}

func pressKey(m model, key string) model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	if key == "enter" {
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	}
	if key == "esc" {
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	}
	if key == " " {
		msg = tea.KeyMsg{Type: tea.KeySpace}
	}
	next, _ := m.Update(msg)
	return next.(model)
}

// --- list key tests ---

func TestNavigateDown(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m = pressKey(m, "down")
	if m.cursor != 1 {
		t.Errorf("cursor want 1, got %d", m.cursor)
	}
}

func TestNavigateUpBoundary(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m = pressKey(m, "up") // wraps to last item
	if m.cursor != 2 {
		t.Errorf("cursor should wrap to 2, got %d", m.cursor)
	}
}

func TestNavigateDownBoundary(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.cursor = 2
	m = pressKey(m, "down") // wraps to first item
	if m.cursor != 0 {
		t.Errorf("cursor should wrap to 0, got %d", m.cursor)
	}
}

func TestSpaceToggle(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m = pressKey(m, " ")
	if !m.selected[1] {
		t.Error("expected session 1 selected")
	}
	m = pressKey(m, " ")
	if m.selected[1] {
		t.Error("expected session 1 deselected")
	}
}

func TestSelectAllFromNone(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m = pressKey(m, "a")
	for _, s := range m.sessions {
		if !m.selected[s.Index] {
			t.Errorf("session %d not selected after 'a'", s.Index)
		}
	}
}

func TestSelectAllFromPartial(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true // 1 of 3 selected
	m = pressKey(m, "a")
	for _, s := range m.sessions {
		if !m.selected[s.Index] {
			t.Errorf("session %d not selected; 'a' from partial should select all", s.Index)
		}
	}
}

func TestSelectAllToggleOff(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	for _, s := range m.sessions {
		m.selected[s.Index] = true
	}
	m = pressKey(m, "a")
	for _, s := range m.sessions {
		if m.selected[s.Index] {
			t.Errorf("session %d still selected; 'a' from all-selected should deselect", s.Index)
		}
	}
}

func TestQuitFromList(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("expected quit cmd")
	}
}

// --- confirm state tests ---

func TestEnterWithSelectionGoesToConfirm(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "enter")
	if m.state != stateConfirm {
		t.Errorf("want stateConfirm, got %v", m.state)
	}
}

func TestEnterWithNoSelectionStaysList(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m = pressKey(m, "enter")
	if m.state != stateList {
		t.Errorf("want stateList, got %v", m.state)
	}
}

func TestConfirmDefaultIsNo(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "enter")
	if m.confirmIdx != 0 {
		t.Errorf("default confirm should be No (0), got %d", m.confirmIdx)
	}
}

func TestConfirmRightMovesToYes(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "enter") // go to confirm
	m = pressKey(m, "right")
	if m.confirmIdx != 1 {
		t.Errorf("want confirmIdx 1 (Yes), got %d", m.confirmIdx)
	}
}

func TestConfirmLeftMovesToNo(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "enter")
	m = pressKey(m, "right") // → Yes
	m = pressKey(m, "left")  // ← No
	if m.confirmIdx != 0 {
		t.Errorf("want confirmIdx 0 (No), got %d", m.confirmIdx)
	}
}

func TestConfirmEscReturnsToList(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "enter")
	m = pressKey(m, "esc")
	if m.state != stateList {
		t.Errorf("esc should return to stateList, got %v", m.state)
	}
}

func TestConfirmEnterNoReturnsToList(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "enter") // confirm screen, No selected
	m = pressKey(m, "enter") // confirm No → back to list
	if m.state != stateList {
		t.Errorf("confirming No should return stateList, got %v", m.state)
	}
}

func TestSelectedClearedAfterDone(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m.selected[2] = true
	// simulate done message
	next, _ := m.Update(deleteDoneMsg{deleted: []string{"sessionA"}, failed: nil})
	m = next.(model)
	for k, v := range m.selected {
		if v {
			t.Errorf("selected[%d] should be cleared after done", k)
		}
	}
}

func TestDoneEnterReloads(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.state = stateDone
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != stateLoading {
		t.Errorf("Enter in stateDone should go stateLoading, got %v", m.state)
	}
	if cmd == nil {
		t.Error("Enter in stateDone should return reload cmd")
	}
}

func TestDoneQQuits(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.state = stateDone
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("q in stateDone should quit")
	}
}

// --- vim key navigation ---

func TestJKNavigation(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m = pressKey(m, "j")
	if m.cursor != 1 {
		t.Errorf("j should move cursor down, got %d", m.cursor)
	}
	m = pressKey(m, "k")
	if m.cursor != 0 {
		t.Errorf("k should move cursor up, got %d", m.cursor)
	}
}

// --- purge mode ---

func TestPKeyEntersPurgeModeConfirm(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "p")
	if m.state != stateConfirm {
		t.Errorf("p should go stateConfirm, got %v", m.state)
	}
	if !m.purgeMode {
		t.Error("purgeMode should be true after p")
	}
}

func TestPKeyNoSelectionStaysList(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m = pressKey(m, "p")
	if m.state != stateList {
		t.Errorf("p with no selection should stay stateList, got %v", m.state)
	}
}

func TestDeleteModeNotPurge(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "enter")
	if m.purgeMode {
		t.Error("enter/delete should set purgeMode=false")
	}
}

// --- force-purge (x key) ---

func TestXKeyGoesToDeleting(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = next.(model)
	if m.state != stateDeleting {
		t.Errorf("x should go stateDeleting, got %v", m.state)
	}
}

func TestXKeyNoSessionsStaysList(t *testing.T) {
	m := makeTestModel(nil)
	m = pressKey(m, "x")
	if m.state != stateList {
		t.Errorf("x with no sessions should stay stateList, got %v", m.state)
	}
}

// --- confirm screen keys ---

func TestConfirmNReturnsToList(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "enter")
	m = pressKey(m, "n")
	if m.state != stateList {
		t.Errorf("n in confirm should return stateList, got %v", m.state)
	}
}

func TestConfirmHLNavigation(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "enter")
	m = pressKey(m, "l") // vim right → Yes
	if m.confirmIdx != 1 {
		t.Errorf("l should move to Yes (1), got %d", m.confirmIdx)
	}
	m = pressKey(m, "h") // vim left → No
	if m.confirmIdx != 0 {
		t.Errorf("h should move to No (0), got %d", m.confirmIdx)
	}
}

func TestConfirmTabNavigation(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.selected[1] = true
	m = pressKey(m, "enter")
	// tab moves to No (left)
	msg := tea.KeyMsg{Type: tea.KeyTab}
	next, _ := m.Update(msg)
	m = next.(model)
	if m.confirmIdx != 0 {
		t.Errorf("tab should set confirmIdx 0, got %d", m.confirmIdx)
	}
}

// --- q quits from all states ---

func TestQQuitFromConfirm(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.state = stateConfirm
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("q should quit from stateConfirm")
	}
}

func TestQQuitFromDeleting(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.state = stateDeleting
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("q should quit from stateDeleting")
	}
}

func TestQQuitFromLoading(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.state = stateLoading
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("q should quit from stateLoading")
	}
}

// --- misc messages ---

func TestWindowSizeMsgUpdatesWidth(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(model)
	if m.width != 120 {
		t.Errorf("width should be 120, got %d", m.width)
	}
}

func TestClaudeCLIMsgStoresVersion(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	next, _ := m.Update(claudeCLIMsg{version: "1.2.3"})
	m = next.(model)
	if m.claudeCLIVersion != "1.2.3" {
		t.Errorf("claudeCLIVersion want '1.2.3', got %q", m.claudeCLIVersion)
	}
	if !m.claudeCLIDetected {
		t.Error("claudeCLIDetected should be true")
	}
}

func TestClaudeCLIMsgNotFound(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	next, _ := m.Update(claudeCLIMsg{version: ""})
	m = next.(model)
	if !m.claudeCLIDetected {
		t.Error("claudeCLIDetected should be true even when CLI not found")
	}
	if m.claudeCLIVersion != "" {
		t.Errorf("claudeCLIVersion should be empty, got %q", m.claudeCLIVersion)
	}
}

func TestDeleteItemMsgUpdatesProgress(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.state = stateDeleting
	m.deleteTotal = 3
	m.deleteSelectedSnap = map[int]bool{1: true, 2: true, 3: true}
	next, _ := m.Update(deleteItemMsg{done: 1, total: 3, deleted: []string{"a"}, failed: nil, nextIdx: 1})
	m = next.(model)
	if m.deleteProgress != 1 {
		t.Errorf("deleteProgress want 1, got %d", m.deleteProgress)
	}
}

func TestDeleteProgressResetOnDone(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.deleteTotal = 5
	m.deleteProgress = 3
	next, _ := m.Update(deleteDoneMsg{deleted: []string{"a"}, failed: nil})
	m = next.(model)
	if m.deleteTotal != 0 || m.deleteProgress != 0 {
		t.Errorf("deleteTotal/Progress should reset, got total=%d progress=%d", m.deleteTotal, m.deleteProgress)
	}
}

// --- empty sessions edge cases ---

func TestEmptySessionsNoActionOnEnter(t *testing.T) {
	m := makeTestModel(nil)
	m = pressKey(m, "enter")
	if m.state != stateList {
		t.Errorf("enter on empty list should stay stateList, got %v", m.state)
	}
}

func TestEmptySessionsNoActionOnSpace(t *testing.T) {
	m := makeTestModel(nil)
	m = pressKey(m, " ")
	if m.state != stateList {
		t.Errorf("space on empty list should stay stateList, got %v", m.state)
	}
}

// --- sort ---

func TestSortKeyCyclesModes(t *testing.T) {
	m := makeTestModel(realisticSessions())
	if m.sortMode != sortRecent {
		t.Fatalf("initial sortMode want sortRecent, got %d", m.sortMode)
	}
	m = pressKey(m, "s")
	if m.sortMode != sortSize {
		t.Errorf("after 1×s want sortSize, got %d", m.sortMode)
	}
	m = pressKey(m, "s")
	if m.sortMode != sortTokens {
		t.Errorf("after 2×s want sortTokens, got %d", m.sortMode)
	}
	m = pressKey(m, "s")
	if m.sortMode != sortName {
		t.Errorf("after 3×s want sortName, got %d", m.sortMode)
	}
	m = pressKey(m, "s")
	if m.sortMode != sortRecent {
		t.Errorf("after 4×s should wrap to sortRecent, got %d", m.sortMode)
	}
}

func TestSortResetsCursor(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.cursor = 3
	m = pressKey(m, "s")
	if m.cursor != 0 {
		t.Errorf("sort should reset cursor to 0, got %d", m.cursor)
	}
}

func TestSortByTokensDescending(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.sortMode = sortTokens
	result := m.filteredSessions()
	for i := 1; i < len(result); i++ {
		if result[i].TotalTokens > result[i-1].TotalTokens {
			t.Errorf("sortTokens: index %d (%d tok) > index %d (%d tok)", i, result[i].TotalTokens, i-1, result[i-1].TotalTokens)
		}
	}
}

func TestSortBySizeDescending(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.sortMode = sortSize
	result := m.filteredSessions()
	for i := 1; i < len(result); i++ {
		if result[i].Size > result[i-1].Size {
			t.Errorf("sortSize: index %d (%d B) > index %d (%d B)", i, result[i].Size, i-1, result[i-1].Size)
		}
	}
}

func TestSortByNameAscending(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.sortMode = sortName
	result := m.filteredSessions()
	for i := 1; i < len(result); i++ {
		ni := result[i-1].ProjectPath
		nj := result[i].ProjectPath
		if ni > nj {
			t.Errorf("sortName: %q should come before %q", ni, nj)
		}
	}
}

// --- filter ---

func TestFilterKeyCyclesModes(t *testing.T) {
	m := makeTestModel(realisticSessions())
	if m.filterMode != filterAll {
		t.Fatalf("initial filterMode want filterAll, got %d", m.filterMode)
	}
	m = pressKey(m, "f")
	if m.filterMode != filterHasData {
		t.Errorf("after 1×f want filterHasData, got %d", m.filterMode)
	}
	m = pressKey(m, "f")
	if m.filterMode != filterOrphaned {
		t.Errorf("after 2×f want filterOrphaned, got %d", m.filterMode)
	}
	m = pressKey(m, "f")
	if m.filterMode != filterAll {
		t.Errorf("after 3×f should wrap to filterAll, got %d", m.filterMode)
	}
}

func TestFilterResetsCursor(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.cursor = 2
	m = pressKey(m, "f")
	if m.cursor != 0 {
		t.Errorf("filter should reset cursor to 0, got %d", m.cursor)
	}
}

func TestFilterHasDataExcludesOrphaned(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.filterMode = filterHasData
	result := m.filteredSessions()
	for _, s := range result {
		if !s.HasData {
			t.Errorf("filterHasData: session %q has HasData=false, should be excluded", s.Name)
		}
	}
}

func TestFilterOrphanedExcludesWithData(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.filterMode = filterOrphaned
	result := m.filteredSessions()
	for _, s := range result {
		if s.HasData {
			t.Errorf("filterOrphaned: session %q has HasData=true, should be excluded", s.Name)
		}
	}
}

func TestFilterAllReturnsEverything(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.filterMode = filterAll
	result := m.filteredSessions()
	if len(result) != len(m.sessions) {
		t.Errorf("filterAll want %d sessions, got %d", len(m.sessions), len(result))
	}
}

// --- search ---

func TestSlashEntersSearchMode(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m = pressKey(m, "/")
	if !m.searching {
		t.Error("'/' should set searching=true")
	}
	if m.searchQuery != "" {
		t.Errorf("searchQuery should be empty on entry, got %q", m.searchQuery)
	}
}

func TestSlashResetsCursor(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.cursor = 3
	m = pressKey(m, "/")
	if m.cursor != 0 {
		t.Errorf("'/' should reset cursor to 0, got %d", m.cursor)
	}
}

func TestSearchAppendRune(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.searching = true
	m.searchQuery = "api"
	m = pressKey(m, "-")
	if m.searchQuery != "api-" {
		t.Errorf("searchQuery want 'api-', got %q", m.searchQuery)
	}
}

func TestSearchBackspace(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.searching = true
	m.searchQuery = "api"
	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	next, _ := m.Update(msg)
	m = next.(model)
	if m.searchQuery != "ap" {
		t.Errorf("backspace want 'ap', got %q", m.searchQuery)
	}
}

func TestSearchBackspaceEmptyNoOp(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.searching = true
	m.searchQuery = ""
	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	next, _ := m.Update(msg)
	m = next.(model)
	if m.searchQuery != "" {
		t.Errorf("backspace on empty query should stay empty, got %q", m.searchQuery)
	}
}

func TestSearchEnterExitsKeepsQuery(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.searching = true
	m.searchQuery = "api"
	m = pressKey(m, "enter")
	if m.searching {
		t.Error("enter should exit search mode")
	}
	if m.searchQuery != "api" {
		t.Errorf("enter should keep query, got %q", m.searchQuery)
	}
}

func TestSearchEscClearsAndExits(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.searching = true
	m.searchQuery = "api"
	m = pressKey(m, "esc")
	if m.searching {
		t.Error("esc should exit search mode")
	}
	if m.searchQuery != "" {
		t.Errorf("esc should clear query, got %q", m.searchQuery)
	}
}

func TestSearchFiltersSessionsByName(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.searchQuery = "api"
	result := m.filteredSessions()
	for _, s := range result {
		found := false
		if s.Name != "" {
			found = found || len(s.Name) > 0 && containsCI(s.Name, "api")
		}
		if s.ProjectPath != "" {
			found = found || containsCI(s.ProjectPath, "api")
		}
		if !found {
			t.Errorf("session %q should not match query 'api'", s.Name)
		}
	}
}

func TestSearchFiltersSessionsByPath(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.searchQuery = "webapp"
	result := m.filteredSessions()
	if len(result) != 1 {
		t.Errorf("query 'webapp' should match 1 session, got %d", len(result))
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.searchQuery = "WEBAPP"
	upper := m.filteredSessions()
	m.searchQuery = "webapp"
	lower := m.filteredSessions()
	if len(upper) != len(lower) {
		t.Errorf("search should be case-insensitive: WEBAPP=%d, webapp=%d", len(upper), len(lower))
	}
}

func TestSearchResetsCursorOnAppend(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.searching = true
	m.cursor = 2
	m = pressKey(m, "a")
	if m.cursor != 0 {
		t.Errorf("typing in search should reset cursor, got %d", m.cursor)
	}
}

// containsCI is a case-insensitive contains helper used only in tests.
func containsCI(s, sub string) bool {
	return len(s) >= len(sub) &&
		func() bool {
			sl := make([]rune, 0, len(s))
			subl := make([]rune, 0, len(sub))
			for _, r := range s {
				if r >= 'A' && r <= 'Z' {
					sl = append(sl, r+32)
				} else {
					sl = append(sl, r)
				}
			}
			for _, r := range sub {
				if r >= 'A' && r <= 'Z' {
					subl = append(subl, r+32)
				} else {
					subl = append(subl, r)
				}
			}
			s2 := string(sl)
			sub2 := string(subl)
			for i := 0; i <= len(s2)-len(sub2); i++ {
				if s2[i:i+len(sub2)] == sub2 {
					return true
				}
			}
			return false
		}()
}

// --- expiry ---

func TestExpiryKeyCycles(t *testing.T) {
	m := makeTestModel(realisticSessions())
	if m.expiryDays != 0 {
		t.Fatalf("initial expiryDays want 0, got %d", m.expiryDays)
	}
	expected := []int{7, 14, 30, 60, 90, 0}
	for i, want := range expected {
		m = pressKey(m, "e")
		if m.expiryDays != want {
			t.Errorf("after %d×e want expiryDays=%d, got %d", i+1, want, m.expiryDays)
		}
	}
}

func TestExpiryFiltersRecentSessions(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.expiryDays = 7 // only show sessions older than 7 days
	result := m.filteredSessions()
	for _, s := range result {
		if !s.HasData {
			continue // orphaned sessions have zero mtime, skip
		}
		cutoff := time.Now().AddDate(0, 0, -7)
		if s.Modified.After(cutoff) {
			t.Errorf("session %q modified %v is within 7-day window, should be excluded", s.Name, s.Modified)
		}
	}
}

func TestExpiryZeroShowsAll(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.expiryDays = 0
	withExpiry := m.filteredSessions()
	m.expiryDays = 7
	withoutRecent := m.filteredSessions()
	if len(withExpiry) <= len(withoutRecent) {
		t.Errorf("expiryDays=0 should show more sessions than expiryDays=7 (got %d vs %d)", len(withExpiry), len(withoutRecent))
	}
}

// --- esc reset ---

func TestEscResetsSearchAndFilters(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.searchQuery = "api"
	m.filterMode = filterHasData
	m.sortMode = sortSize
	m = pressKey(m, "esc")
	if m.searchQuery != "" {
		t.Errorf("esc should clear searchQuery, got %q", m.searchQuery)
	}
	if m.filterMode != filterAll {
		t.Errorf("esc should reset filterMode to filterAll, got %d", m.filterMode)
	}
	if m.sortMode != sortRecent {
		t.Errorf("esc should reset sortMode to sortRecent, got %d", m.sortMode)
	}
}

func TestDKeyResetsAll(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m.sortMode = sortName
	m.filterMode = filterOrphaned
	m.searchQuery = "api"
	m.cursor = 3
	m.selected = map[int]bool{1: true, 2: true}
	m = pressKey(m, "d")
	if m.sortMode != sortRecent {
		t.Errorf("d should reset sortMode, got %d", m.sortMode)
	}
	if m.filterMode != filterAll {
		t.Errorf("d should reset filterMode, got %d", m.filterMode)
	}
	if m.searchQuery != "" {
		t.Errorf("d should clear searchQuery, got %q", m.searchQuery)
	}
	if m.cursor != 0 {
		t.Errorf("d should reset cursor, got %d", m.cursor)
	}
	if len(m.selected) != 0 {
		t.Errorf("d should clear selected map, got %v", m.selected)
	}
}

// --- orphaned select ---

func TestOKeySelectsOnlyOrphaned(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m = pressKey(m, "o")
	for _, s := range m.sessions {
		selected := m.selected[s.Index]
		if !s.HasData && !selected {
			t.Errorf("session %q (orphaned) should be selected after 'o'", s.Name)
		}
		if s.HasData && selected {
			t.Errorf("session %q (has data) should NOT be selected after 'o'", s.Name)
		}
	}
}

// --- category ---

func TestCKeyOpensCategories(t *testing.T) {
	m := makeTestModel(realisticSessions())
	m = pressKey(m, "c")
	if m.state != stateCategories {
		t.Errorf("'c' should set state to stateCategories, got %v", m.state)
	}
	if m.categoryCursor != 0 {
		t.Errorf("categoryCursor should reset to 0, got %d", m.categoryCursor)
	}
}

func fakeCategories() []Category {
	return []Category{
		{Key: "debug", Label: "Debug logs", Exists: true, Size: 1024},
		{Key: "telemetry", Label: "Telemetry", Exists: true, Size: 2048},
		{Key: "transcripts", Label: "Transcripts", Exists: false, Size: 0},
	}
}

func makeCategoryModel() model {
	m := makeTestModel(realisticSessions())
	m.state = stateCategories
	m.categories = fakeCategories()
	m.categorySelected = make(map[string]bool)
	m.categoryCursor = 0
	return m
}

func TestCategoryNavigateDown(t *testing.T) {
	m := makeCategoryModel()
	m = pressKey(m, "j")
	if m.categoryCursor != 1 {
		t.Errorf("j should move categoryCursor to 1, got %d", m.categoryCursor)
	}
}

func TestCategoryNavigateUp(t *testing.T) {
	m := makeCategoryModel()
	m.categoryCursor = 1
	m = pressKey(m, "k")
	if m.categoryCursor != 0 {
		t.Errorf("k should move categoryCursor to 0, got %d", m.categoryCursor)
	}
}

func TestCategoryBoundaryWrapDown(t *testing.T) {
	m := makeCategoryModel()
	m.categoryCursor = len(m.categories) - 1
	m = pressKey(m, "j")
	if m.categoryCursor != 0 {
		t.Errorf("j at last item should wrap to 0, got %d", m.categoryCursor)
	}
}

func TestCategoryBoundaryWrapUp(t *testing.T) {
	m := makeCategoryModel()
	m.categoryCursor = 0
	m = pressKey(m, "k")
	if m.categoryCursor != len(m.categories)-1 {
		t.Errorf("k at first item should wrap to last (%d), got %d", len(m.categories)-1, m.categoryCursor)
	}
}

func TestCategoryGJumpsToTop(t *testing.T) {
	m := makeCategoryModel()
	m.categoryCursor = 2
	m = pressKey(m, "g")
	if m.categoryCursor != 0 {
		t.Errorf("g should jump to top, got %d", m.categoryCursor)
	}
}

func TestCategoryGShiftJumpsToBottom(t *testing.T) {
	m := makeCategoryModel()
	m.categoryCursor = 0
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")}
	next, _ := m.Update(msg)
	m = next.(model)
	if m.categoryCursor != len(m.categories)-1 {
		t.Errorf("G should jump to bottom (%d), got %d", len(m.categories)-1, m.categoryCursor)
	}
}

func TestCategorySpaceTogglesExisting(t *testing.T) {
	m := makeCategoryModel()
	m.categoryCursor = 0 // "debug", Exists=true
	m = pressKey(m, " ")
	if !m.categorySelected["debug"] {
		t.Error("space should select 'debug'")
	}
	m = pressKey(m, " ")
	if m.categorySelected["debug"] {
		t.Error("space again should deselect 'debug'")
	}
}

func TestCategorySpaceSkipsNonExistent(t *testing.T) {
	m := makeCategoryModel()
	m.categoryCursor = 2 // "transcripts", Exists=false
	m = pressKey(m, " ")
	if m.categorySelected["transcripts"] {
		t.Error("space on non-existent category should not select it")
	}
}

func TestCategorySelectAll(t *testing.T) {
	m := makeCategoryModel()
	m = pressKey(m, "a")
	for _, cat := range m.categories {
		if cat.Exists && !m.categorySelected[cat.Key] {
			t.Errorf("'a' should select existing category %q", cat.Key)
		}
		if !cat.Exists && m.categorySelected[cat.Key] {
			t.Errorf("'a' should not select non-existent category %q", cat.Key)
		}
	}
}

func TestCategorySelectAllThenDeselectAll(t *testing.T) {
	m := makeCategoryModel()
	m = pressKey(m, "a") // select all
	m = pressKey(m, "a") // toggle: deselect all
	for _, cat := range m.categories {
		if m.categorySelected[cat.Key] {
			t.Errorf("second 'a' should deselect %q", cat.Key)
		}
	}
}

func TestCategoryNDeselectsAll(t *testing.T) {
	m := makeCategoryModel()
	m.categorySelected["debug"] = true
	m.categorySelected["telemetry"] = true
	m = pressKey(m, "n")
	if len(m.categorySelected) != 0 {
		t.Errorf("'n' should clear categorySelected, got %v", m.categorySelected)
	}
}

func TestCategoryEnterWithSelectionGoesToConfirm(t *testing.T) {
	m := makeCategoryModel()
	m.categorySelected["debug"] = true
	m = pressKey(m, "enter")
	if m.state != stateCategoryConfirm {
		t.Errorf("enter with selection should go to stateCategoryConfirm, got %v", m.state)
	}
	if m.confirmIdx != 0 {
		t.Errorf("confirmIdx should be 0 (No default), got %d", m.confirmIdx)
	}
}

func TestCategoryEnterNoSelectionStays(t *testing.T) {
	m := makeCategoryModel()
	// no selection
	m = pressKey(m, "enter")
	if m.state != stateCategories {
		t.Errorf("enter with no selection should stay stateCategories, got %v", m.state)
	}
}

func TestCategoryEscReturnsToList(t *testing.T) {
	m := makeCategoryModel()
	m = pressKey(m, "esc")
	if m.state != stateList {
		t.Errorf("esc in stateCategories should return to stateList, got %v", m.state)
	}
	if len(m.categorySelected) != 0 {
		t.Error("esc should clear categorySelected")
	}
}

func TestCategoryConfirmEscReturnsToCategories(t *testing.T) {
	m := makeCategoryModel()
	m.state = stateCategoryConfirm
	m.confirmIdx = 1
	m = pressKey(m, "esc")
	if m.state != stateCategories {
		t.Errorf("esc in stateCategoryConfirm should go to stateCategories, got %v", m.state)
	}
	if m.confirmIdx != 0 {
		t.Errorf("confirmIdx should reset to 0, got %d", m.confirmIdx)
	}
}

func TestCategoryConfirmNReturnsToCategories(t *testing.T) {
	m := makeCategoryModel()
	m.state = stateCategoryConfirm
	m = pressKey(m, "n")
	if m.state != stateCategories {
		t.Errorf("'n' in stateCategoryConfirm should go to stateCategories, got %v", m.state)
	}
}

func TestCategoryConfirmRightMovesToYes(t *testing.T) {
	m := makeCategoryModel()
	m.state = stateCategoryConfirm
	msg := tea.KeyMsg{Type: tea.KeyRight}
	next, _ := m.Update(msg)
	m = next.(model)
	if m.confirmIdx != 1 {
		t.Errorf("right should set confirmIdx=1 (Yes), got %d", m.confirmIdx)
	}
}

func TestCategoryConfirmLeftMovesToNo(t *testing.T) {
	m := makeCategoryModel()
	m.state = stateCategoryConfirm
	m.confirmIdx = 1
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	next, _ := m.Update(msg)
	m = next.(model)
	if m.confirmIdx != 0 {
		t.Errorf("left should set confirmIdx=0 (No), got %d", m.confirmIdx)
	}
}

func TestCategoryConfirmTabMovesToNo(t *testing.T) {
	m := makeCategoryModel()
	m.state = stateCategoryConfirm
	m.confirmIdx = 1
	msg := tea.KeyMsg{Type: tea.KeyTab}
	next, _ := m.Update(msg)
	m = next.(model)
	if m.confirmIdx != 0 {
		t.Errorf("tab should set confirmIdx=0 (No), got %d", m.confirmIdx)
	}
}

func TestCategoryConfirmEnterOnNoReturnsToCategories(t *testing.T) {
	m := makeCategoryModel()
	m.state = stateCategoryConfirm
	m.confirmIdx = 0 // No
	m = pressKey(m, "enter")
	if m.state != stateCategories {
		t.Errorf("enter on No should return to stateCategories, got %v", m.state)
	}
}
