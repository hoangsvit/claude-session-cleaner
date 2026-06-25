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
	m = pressKey(m, "up")
	if m.cursor != 0 {
		t.Errorf("cursor should stay 0, got %d", m.cursor)
	}
}

func TestNavigateDownBoundary(t *testing.T) {
	m := makeTestModel(fakeSessions(3))
	m.cursor = 2
	m = pressKey(m, "down")
	if m.cursor != 2 {
		t.Errorf("cursor should stay 2, got %d", m.cursor)
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
