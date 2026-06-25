package main

import "testing"

func TestSemverGT(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.1.0", "1.0.0", true},
		{"1.0.1", "1.0.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.1.0", false},
		{"1.0.0", "2.0.0", false},
		{"1.2.3-beta", "1.2.2", true},
	}
	for _, c := range cases {
		if got := semverGT(c.a, c.b); got != c.want {
			t.Errorf("semverGT(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestSplitVer(t *testing.T) {
	cases := []struct {
		in   string
		want [3]int
	}{
		{"1.2.3", [3]int{1, 2, 3}},
		{"10.0.1", [3]int{10, 0, 1}},
		{"1.0.0-beta", [3]int{1, 0, 0}},
		{"2", [3]int{2, 0, 0}},
	}
	for _, c := range cases {
		if got := splitVer(c.in); got != c.want {
			t.Errorf("splitVer(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestUpdateCheckMsgHandled(t *testing.T) {
	m := makeTestModel(fakeSessions(2))
	next, _ := m.Update(updateCheckMsg{latest: "1.1.0", hasUpdate: true})
	m = next.(model)
	if !m.hasUpdate {
		t.Error("hasUpdate should be true")
	}
	if m.latestVersion != "1.1.0" {
		t.Errorf("latestVersion want '1.1.0', got %q", m.latestVersion)
	}
}

func TestUpdateCheckMsgNoUpdate(t *testing.T) {
	m := makeTestModel(fakeSessions(2))
	next, _ := m.Update(updateCheckMsg{latest: "1.0.0", hasUpdate: false})
	m = next.(model)
	if m.hasUpdate {
		t.Error("hasUpdate should be false")
	}
}
