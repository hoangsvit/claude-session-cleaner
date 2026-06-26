package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Preferences persisted between sessions at ~/.claude/cleaner-preferences.json
type Preferences struct {
	SortMode   int `json:"sort_mode"`
	FilterMode int `json:"filter_mode"`
	ExpiryDays int `json:"expiry_days"`
}

func prefsPath(claudeDir string) string {
	return filepath.Join(claudeDir, "cleaner-preferences.json")
}

func loadPrefs(claudeDir string) Preferences {
	var p Preferences
	data, err := os.ReadFile(prefsPath(claudeDir))
	if err != nil {
		return p
	}
	_ = json.Unmarshal(data, &p)
	return p
}

func writePrefs(claudeDir string, p Preferences) {
	data, err := json.Marshal(p)
	if err != nil {
		return
	}
	_ = os.WriteFile(prefsPath(claudeDir), data, 0644)
}
