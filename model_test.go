package main

import (
	"testing"

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
