package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dracula palette
var (
	clrPurple  = lipgloss.Color("#BD93F9")
	clrCyan    = lipgloss.Color("#8BE9FD")
	clrGreen   = lipgloss.Color("#50FA7B")
	clrRed     = lipgloss.Color("#FF5555")
	clrFg      = lipgloss.Color("#F8F8F2")
	clrComment = lipgloss.Color("#6272A4")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(clrPurple).
			Background(clrBg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrPurple).
			Padding(0, 2)

	clrBg         = lipgloss.Color("#282A36")
	clrSelection  = lipgloss.Color("#44475A")
	clrCursor     = lipgloss.Color("#6272A4")

	dimStyle      = lipgloss.NewStyle().Foreground(clrComment)
	nameStyle     = lipgloss.NewStyle().Foreground(clrFg)
	timeStyle     = lipgloss.NewStyle().Foreground(clrComment)
	sizeStyle     = lipgloss.NewStyle().Foreground(clrCyan)
	checkOnStyle  = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)
	checkOffStyle = lipgloss.NewStyle().Foreground(clrComment)
	cursorStyle   = lipgloss.NewStyle().Foreground(clrPurple).Bold(true)
	dangerStyle   = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	successStyle  = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)
	countStyle    = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)

	rowNormalStyle   = lipgloss.NewStyle().Background(clrBg)
	rowCursorStyle   = lipgloss.NewStyle().Background(clrCursor)
	rowSelectedStyle = lipgloss.NewStyle().Background(clrSelection)

	helpStyle = lipgloss.NewStyle().
			Foreground(clrComment).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(clrComment).
			MarginTop(1).
			PaddingTop(1)
)

type appState int

const (
	stateLoading appState = iota
	stateUpdatePrompt
	stateList
	stateConfirm
	stateDeleting
	stateDone
)

const bannerLogo = ` ██████╗██╗      █████╗ ██╗   ██╗██████╗ ███████╗
██╔════╝██║     ██╔══██╗██║   ██║██╔══██╗██╔════╝
██║     ██║     ███████║██║   ██║██║  ██║█████╗
██║     ██║     ██╔══██║██║   ██║██║  ██║██╔══╝
╚██████╗███████╗██║  ██║╚██████╔╝██████╔╝███████╗
 ╚═════╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═════╝╚══════╝
           C  L  E  A  N  E  R`

type sessionsLoadedMsg struct {
	sessions []Session
	err      error
}

type claudeCLIMsg struct{ version string }
type updateCheckMsg struct {
	latest    string
	hasUpdate bool
}
type updateDoneMsg struct{ err error }
type rescanDoneMsg struct {
	sessions   []Session
	err        error
	cliVersion string
}

type deleteDoneMsg struct {
	deleted []string
	failed  []string
}

type deleteItemMsg struct {
	done    int
	total   int
	deleted []string
	failed  []string
	nextIdx int
}

type model struct {
	state          appState
	claudeDir      string
	claudeJSONPath string
	projectsDir    string
	sessions       []Session
	selected       map[int]bool
	cursor         int
	spinner        spinner.Model
	confirmIdx     int  // 0 = No (default), 1 = Yes
	purgeMode      bool // true = full purge via claude CLI, false = session files only
	deleted        []string
	failed         []string
	width          int
	deleteTotal        int
	deleteProgress     int
	deleteSelectedSnap map[int]bool
	claudeCLIVersion   string // "" = not yet detected
	claudeCLIDetected  bool
	latestVersion        string
	hasUpdate            bool
	updateChecked        bool
	pendingUpdatePrompt  bool   // update arrived before sessions loaded
	sessionsReady        bool   // sessions loaded flag
	lastScanTime         time.Time
	rescanning           bool   // manual rescan in progress — keep list visible
}

func newModel(claudeDir, claudeJSONPath, projectsDir string) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(clrPurple)

	return model{
		state:          stateLoading,
		claudeDir:      claudeDir,
		claudeJSONPath: claudeJSONPath,
		projectsDir:    projectsDir,
		selected:       make(map[int]bool),
		spinner:        sp,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			sessions, err := scanSessions(m.claudeJSONPath, m.projectsDir)
			return sessionsLoadedMsg{sessions, err}
		},
		func() tea.Msg {
			return claudeCLIMsg{DetectClaudeCLI()}
		},
		func() tea.Msg {
			latest, hasUpdate := CheckLatestVersion(version)
			return updateCheckMsg{latest, hasUpdate}
		},
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
			return m, tea.Quit
		}
		if m.state == stateUpdatePrompt {
			return m.handleUpdatePromptKey(msg)
		}
		if m.state == stateConfirm {
			return m.handleConfirmKey(msg)
		}
		return m.handleListKey(msg)

	case sessionsLoadedMsg:
		if msg.err != nil {
			return m, tea.Quit
		}
		wasRescanning := m.rescanning
		m.sessions = msg.sessions
		m.sessionsReady = true
		m.rescanning = false
		m.lastScanTime = time.Now()
		if wasRescanning {
			// manual rescan: reset cursor/selection, go straight to list
			m.cursor = 0
			m.selected = make(map[int]bool)
			m.state = stateList
		} else if m.pendingUpdatePrompt {
			m.state = stateUpdatePrompt
		} else {
			m.state = stateList
		}
		return m, nil

	case deleteItemMsg:
		m.deleteProgress = msg.done
		return m, nextDeleteCmd(m.sessions, m.deleteSelectedSnap, msg.nextIdx, msg.done, msg.total, msg.deleted, msg.failed, m.projectsDir)

	case rescanDoneMsg:
		if msg.err != nil {
			return m, tea.Quit
		}
		m.sessions = msg.sessions
		m.sessionsReady = true
		m.rescanning = false
		m.lastScanTime = time.Now()
		m.cursor = 0
		m.selected = make(map[int]bool)
		m.claudeCLIVersion = msg.cliVersion
		m.claudeCLIDetected = true
		m.state = stateList
		return m, nil

	case claudeCLIMsg:
		m.claudeCLIVersion = msg.version
		m.claudeCLIDetected = true
		return m, nil

	case updateCheckMsg:
		m.latestVersion = msg.latest
		m.hasUpdate = msg.hasUpdate
		m.updateChecked = true
		if msg.hasUpdate {
			if m.sessionsReady {
				m.state = stateUpdatePrompt
			} else {
				m.pendingUpdatePrompt = true
			}
		}
		return m, nil

	case updateDoneMsg:
		// npm finished — quit, user must restart to use new binary
		if msg.err == nil {
			fmt.Print("\n  Update complete. Please restart claude-cleaner.\n")
		}
		return m, tea.Quit

	case deleteDoneMsg:
		m.deleted = msg.deleted
		m.failed = msg.failed
		m.state = stateDone
		m.selected = make(map[int]bool)
		m.deleteSelectedSnap = nil
		m.deleteTotal = 0
		m.deleteProgress = 0
		m.purgeMode = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}


	return m, nil
}

func (m model) handleUpdatePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		cmd := exec.Command("npm", "install", "-g", "claude-cleaner@latest")
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return updateDoneMsg{err}
		})
	case "n", "N", "esc":
		m.pendingUpdatePrompt = false // clear so rescan won't re-show prompt
		m.state = stateList
		return m, nil
	}
	return m, nil
}

func (m model) viewUpdatePrompt() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(clrGreen).Bold(true).Render("⬆  New version available: v"+m.latestVersion) + "\n\n")
	sb.WriteString("  " + dimStyle.Render("Current: v"+version) + "\n\n")
	sb.WriteString("  Update now via npm install -g claude-cleaner@latest?\n\n")

	no := dimStyle.Render("[ N ]  No, skip")
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(clrGreen).Bold(true).Render("[ Y ]  Yes, update now") + "      " + no + "\n\n")
	sb.WriteString("  " + dimStyle.Render("y/enter update  n/esc skip"))
	return sb.String()
}

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.sessions)

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < n-1 {
			m.cursor++
		}

	case "a":
		allOn := n > 0
		for _, s := range m.sessions {
			if !m.selected[s.Index] {
				allOn = false
				break
			}
		}
		for _, s := range m.sessions {
			m.selected[s.Index] = !allOn
		}

	case " ":
		// Toggle selection on current row
		if n > 0 {
			idx := m.sessions[m.cursor].Index
			m.selected[idx] = !m.selected[idx]
		}
		return m, nil

	case "enter":
		if m.state == stateDone {
			m.deleted = nil
			m.failed = nil
			return m.doRescan()
		}
		count := 0
		for _, v := range m.selected {
			if v {
				count++
			}
		}
		if count > 0 {
			m.purgeMode = false
			m.state = stateConfirm
			m.confirmIdx = 0
			return m, nil
		}
		return m, nil

	case "p", "P":
		count := 0
		for _, v := range m.selected {
			if v {
				count++
			}
		}
		if count > 0 {
			m.purgeMode = true
			m.state = stateConfirm
			m.confirmIdx = 0
			return m, nil
		}

	case "r", "R":
		return m.doRescan()

	case "x", "X":
		// Force purge project at cursor — no confirm screen, single project only.
		if n > 0 {
			return m.doPurgeDirect(m.sessions[m.cursor])
		}

	case "u", "U":
		if m.hasUpdate {
			cmd := exec.Command("npm", "install", "-g", "claude-cleaner@latest")
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return updateDoneMsg{err}
			})
		}
	}

	return m, nil
}

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.state = stateList
		m.confirmIdx = 0
		m.purgeMode = false
		return m, nil

	case "left", "h", "tab":
		m.confirmIdx = 0
		return m, nil

	case "right", "l":
		m.confirmIdx = 1
		return m, nil

	case "y":
		m.confirmIdx = 1
		if m.purgeMode {
			return m.doPurge()
		}
		return m.doDelete()

	case "enter":
		if m.confirmIdx == 1 {
			if m.purgeMode {
				return m.doPurge()
			}
			return m.doDelete()
		}
		m.state = stateList
		m.confirmIdx = 0
		m.purgeMode = false
		return m, nil
	}
	return m, nil
}

func (m model) doRescan() (tea.Model, tea.Cmd) {
	m.sessionsReady = false
	m.pendingUpdatePrompt = false
	claudeJSONPath := m.claudeJSONPath
	projectsDir := m.projectsDir

	if m.state == stateList {
		// Keep list visible, overlay spinner. Bundle with CLI detect so
		// goroutine takes ~100-200ms (subprocess) — long enough to see.
		m.rescanning = true
		return m, tea.Batch(
			m.spinner.Tick,
			func() tea.Msg {
				sessions, err := scanSessions(claudeJSONPath, projectsDir)
				cliVersion := DetectClaudeCLI() // adds real latency
				return rescanDoneMsg{sessions, err, cliVersion}
			},
		)
	}

	// stateDone or other: full loading screen
	m.state = stateLoading
	m.cursor = 0
	m.selected = make(map[int]bool)
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			sessions, err := scanSessions(claudeJSONPath, projectsDir)
			return sessionsLoadedMsg{sessions, err}
		},
	)
}

func (m model) doDelete() (tea.Model, tea.Cmd) {
	m.state = stateDeleting

	snap := make(map[int]bool, len(m.selected))
	for k, v := range m.selected {
		snap[k] = v
	}

	total := 0
	for _, v := range snap {
		if v {
			total++
		}
	}
	m.deleteTotal = total
	m.deleteProgress = 0
	m.deleteSelectedSnap = snap

	// All selected: use RunDelete (--all optimization, single shot)
	allSelected := total == len(m.sessions) && total > 0
	if allSelected {
		sessions := m.sessions
		projectsDir := m.projectsDir
		return m, tea.Batch(
			m.spinner.Tick,
			func() tea.Msg {
				deleted, failed := RunDelete(sessions, snap, projectsDir)
				return deleteDoneMsg{deleted, failed}
			},
		)
	}

	// Partial selection: per-item sequential cmd for live progress bar.
	return m, tea.Batch(
		m.spinner.Tick,
		nextDeleteCmd(m.sessions, snap, 0, 0, total, nil, nil, m.projectsDir),
	)
}

// nextDeleteCmd processes one selected session starting from startIdx and
// returns either a deleteItemMsg (more items remain) or deleteDoneMsg (all done).
func nextDeleteCmd(sessions []Session, selected map[int]bool, startIdx, done, total int, deleted, failed []string, projectsDir string) tea.Cmd {
	return func() tea.Msg {
		for i := startIdx; i < len(sessions); i++ {
			s := sessions[i]
			if !selected[s.Index] {
				continue
			}
			err := smartDelete(s, projectsDir)
			newDone := done + 1
			newDel := append([]string(nil), deleted...)
			newFail := append([]string(nil), failed...)
			if err != nil {
				newFail = append(newFail, s.Name)
			} else {
				newDel = append(newDel, s.Name)
			}
			if newDone >= total {
				return deleteDoneMsg{newDel, newFail}
			}
			return deleteItemMsg{
				done: newDone, total: total,
				deleted: newDel, failed: newFail,
				nextIdx: i + 1,
			}
		}
		return deleteDoneMsg{deleted, failed}
	}
}

func (m model) doPurge() (tea.Model, tea.Cmd) {
	return m.doDelete()
}

func (m model) doPurgeDirect(s Session) (tea.Model, tea.Cmd) {
	m.state = stateDeleting
	projectsDir := m.projectsDir
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			if err := smartDelete(s, projectsDir); err != nil {
				return deleteDoneMsg{failed: []string{s.Name}}
			}
			return deleteDoneMsg{deleted: []string{s.Name}}
		},
	)
}

func (m model) View() string {
	header := m.renderHeader()

	var body string
	switch m.state {
	case stateLoading:
		body = "\n  " + m.spinner.View() + " Scanning sessions…\n"
	case stateUpdatePrompt:
		body = m.viewUpdatePrompt()
	case stateList:
		body = m.viewList()
	case stateConfirm:
		body = m.viewConfirm()
	case stateDeleting:
		body = m.viewDeleting()
	case stateDone:
		body = m.viewDone()
	}

	return header + body
}

func (m model) viewList() string {
	if len(m.sessions) == 0 {
		if m.rescanning {
			return "\n  " + m.spinner.View() + " Rescanning…\n"
		}
		return "\n  " + dimStyle.Render("No Claude project sessions found.") + "\n" +
			"\n  " + dimStyle.Render("r rescan  q quit")
	}

	const (
		nameW   = 36
		timeW   = 12
		tokensW = 10
		sizeW   = 8
	)

	var sb strings.Builder
	sb.WriteString("\n")

	if m.rescanning {
		sb.WriteString("  " + m.spinner.View() + " " +
			lipgloss.NewStyle().Foreground(clrPurple).Render("Rescanning…") + "\n\n")
	}

	// Header row
	sb.WriteString(dimStyle.Render(fmt.Sprintf("        %-*s  %-*s  %-*s  %s",
		nameW, "Name",
		timeW, "Last modified",
		tokensW, "Tokens",
		"Size",
	)) + "\n")
	sb.WriteString(dimStyle.Render("  "+strings.Repeat("─", nameW+timeW+tokensW+sizeW+18)) + "\n")

	rowW := m.width
	if rowW < 82 {
		rowW = 82
	}

	for _, s := range m.sessions {
		isCursor := m.cursor == s.Index-1
		isSelected := m.selected[s.Index]

		var bg lipgloss.Color
		var rowStyle lipgloss.Style
		switch {
		case isCursor:
			bg = clrCursor
			rowStyle = rowCursorStyle
		case isSelected:
			bg = clrSelection
			rowStyle = rowSelectedStyle
		default:
			bg = clrBg
			rowStyle = rowNormalStyle
		}

		cur := lipgloss.NewStyle().Background(bg).Render("  ")
		if isCursor {
			cur = lipgloss.NewStyle().Foreground(clrPurple).Background(bg).Bold(true).Render("▶ ")
		}

		check := lipgloss.NewStyle().Foreground(clrComment).Background(bg).Render("[ ]")
		if isSelected {
			check = lipgloss.NewStyle().Foreground(clrGreen).Background(bg).Bold(true).Render("[✓]")
		}

		// ● green = has session data  ○ dim = no local data (can still purge config)
		var status string
		if s.HasData {
			status = lipgloss.NewStyle().Foreground(clrGreen).Background(bg).Render("●")
		} else {
			status = lipgloss.NewStyle().Foreground(clrComment).Background(bg).Render("○")
		}

		displayName := s.Name
		if s.ProjectPath != "" {
			displayName = s.ProjectPath
		}

		nameFg := clrFg
		if !s.HasData {
			nameFg = clrComment
		}
		name := lipgloss.NewStyle().Foreground(nameFg).Background(bg).Width(nameW).Render(truncate(displayName, nameW))

		var timeStr, tokStr, szStr string
		if s.HasData {
			timeStr = humanTime(s.Modified)
			if s.HasTokenData {
				tokStr = formatTokens(s.TotalTokens)
			} else {
				tokStr = "—"
			}
			szStr = formatSize(s.Size)
		} else {
			timeStr = "—"
			tokStr = "—"
			szStr = "—"
		}
		t := lipgloss.NewStyle().Foreground(clrComment).Background(bg).Width(timeW).Render(timeStr)
		tok := lipgloss.NewStyle().Foreground(clrPurple).Background(bg).Width(tokensW).Render(tokStr)
		sz := lipgloss.NewStyle().Foreground(clrCyan).Background(bg).Render(szStr)

		content := cur + check + " " + status + " " + name + "  " + t + "  " + tok + "  " + sz
		sb.WriteString(rowStyle.Width(rowW).Render(content) + "\n")
	}

	selected := 0
	for _, v := range m.selected {
		if v {
			selected++
		}
	}

	footer := fmt.Sprintf(
		"↑/↓ navigate  space select  a select all  enter delete  p purge  x force-purge  q quit    %s selected",
		countStyle.Render(fmt.Sprintf("%d", selected)),
	)
	sb.WriteString(helpStyle.Render(footer))

	return sb.String()
}

func (m model) viewConfirm() string {
	var sb strings.Builder
	sb.WriteString("\n")
	if m.purgeMode {
		sb.WriteString("  " + dangerStyle.Render("⚠  Full purge — will delete ALL project data:") + "\n\n")
	} else {
		sb.WriteString("  " + dangerStyle.Render("⚠  Will delete session files:") + "\n\n")
	}

	var totalSize, totalTokens int64
	var anyTokenData bool
	for _, s := range m.sessions {
		if !m.selected[s.Index] {
			continue
		}
		totalSize += s.Size
		if s.HasTokenData {
			totalTokens += s.TotalTokens
			anyTokenData = true
		}
		displayName := s.Name
		if s.ProjectPath != "" {
			displayName = s.ProjectPath
		}
		tokLabel := "—"
		if s.HasTokenData {
			tokLabel = formatTokens(s.TotalTokens) + " tok"
		}
		sb.WriteString(fmt.Sprintf("    %s  %s  %s  %s\n",
			checkOnStyle.Render("✓"),
			nameStyle.Render(truncate(displayName, 38)),
			lipgloss.NewStyle().Foreground(clrPurple).Render(tokLabel),
			sizeStyle.Render(formatSize(s.Size)),
		))
	}

	totalTokStr := "—"
	if anyTokenData {
		totalTokStr = formatTokens(totalTokens) + " tokens"
	}
	sb.WriteString(fmt.Sprintf("\n  Total: %s  %s\n\n",
		sizeStyle.Render(formatSize(totalSize)),
		lipgloss.NewStyle().Foreground(clrPurple).Render(totalTokStr),
	))
	if m.purgeMode {
		sb.WriteString("  " + dangerStyle.Render("Deletes transcripts, tasks, file history, and config. Cannot undo.") + "\n\n")
	} else {
		sb.WriteString("  " + dimStyle.Render("Deletes session history only. Source code is NOT affected.") + "\n\n")
	}

	no := dimStyle.Render("[ N ]  No, cancel")
	yes := dimStyle.Render("[ Y ]  Yes, delete")
	if m.confirmIdx == 0 {
		no = lipgloss.NewStyle().Foreground(clrFg).Bold(true).Render("[ N ]  No, cancel")
	} else {
		yes = lipgloss.NewStyle().Foreground(clrRed).Bold(true).Render("[ Y ]  Yes, delete")
	}
	sb.WriteString("  " + no + "      " + yes + "\n\n")
	sb.WriteString("  " + dimStyle.Render("←/→ or y/n select  enter confirm  esc back"))

	return sb.String()
}

func (m model) viewDeleting() string {
	if m.deleteTotal <= 0 || m.deleteProgress <= 0 {
		return "\n  " + m.spinner.View() + " Deleting…\n"
	}
	const barWidth = 28
	pct := float64(m.deleteProgress) / float64(m.deleteTotal)
	filled := int(pct * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("\n  %s Deleting…  %s  %s\n",
		m.spinner.View(),
		lipgloss.NewStyle().Foreground(clrPurple).Render("["+bar+"]"),
		lipgloss.NewStyle().Foreground(clrCyan).Render(fmt.Sprintf("%d / %d", m.deleteProgress, m.deleteTotal)),
	)
}

func (m model) viewDone() string {
	var sb strings.Builder
	sb.WriteString("\n")

	for _, name := range m.deleted {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", successStyle.Render("✓"), name))
	}
	for _, name := range m.failed {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", dangerStyle.Render("✗"), name))
	}

	if len(m.deleted) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s\n",
			successStyle.Render(fmt.Sprintf("%d session(s) deleted", len(m.deleted))),
		))
	}
	sb.WriteString("\n  " + dimStyle.Render("enter back to list  q quit"))

	return sb.String()
}

func (m model) renderHeader() string {
	logoLines := strings.Split(bannerLogo, "\n")
	art := strings.Join(logoLines[:6], "\n")
	sub := logoLines[6]

	logo := lipgloss.NewStyle().Foreground(clrPurple).Render(art) + "\n" +
		lipgloss.NewStyle().Foreground(clrCyan).Bold(true).Render(sub)
	logoPanel := lipgloss.NewStyle().Padding(0, 1).Render(logo)

	lw := 10
	label := func(s string) string {
		return lipgloss.NewStyle().Foreground(clrCyan).Bold(true).Width(lw).Render(s + ":")
	}
	title := lipgloss.NewStyle().Foreground(clrPurple).Bold(true).Render("ePlus.DEV") +
		lipgloss.NewStyle().Foreground(clrFg).Render("/claude-cleaner")
	divider := lipgloss.NewStyle().Foreground(clrPurple).Render(strings.Repeat("─", 36))
	dirLine := label("Dir") + lipgloss.NewStyle().Foreground(clrFg).Render(m.claudeDir)
	webLine := label("Web") + lipgloss.NewStyle().Foreground(clrCyan).Render("https://eplus.dev")
	var verBadge string
	switch {
	case !m.updateChecked:
		verBadge = dimStyle.Render("(checking…)")
	case m.hasUpdate:
		verBadge = lipgloss.NewStyle().Foreground(clrGreen).Bold(true).Render("⬆ v"+m.latestVersion+" available") +
			"  " + dimStyle.Render("press u to update")
	case m.latestVersion == "":
		verBadge = dimStyle.Render("(offline)")
	default:
		verBadge = dimStyle.Render("(latest)")
	}
	verLine := label("Version") + lipgloss.NewStyle().Foreground(clrComment).Render(version) + "  " + verBadge

	var claudeLine string
	if !m.claudeCLIDetected {
		claudeLine = label("Claude") + dimStyle.Render("detecting…")
	} else if m.claudeCLIVersion == "" {
		claudeLine = label("Claude") + lipgloss.NewStyle().Foreground(clrComment).Render("○ not found")
	} else {
		claudeLine = label("Claude") + lipgloss.NewStyle().Foreground(clrGreen).Bold(true).Render("● "+m.claudeCLIVersion)
	}

	scanLabel := "—"
	if !m.lastScanTime.IsZero() {
		scanLabel = humanTime(m.lastScanTime) + "  " + dimStyle.Render("r rescan")
	}
	scanLine := label("Scanned") + lipgloss.NewStyle().Foreground(clrComment).Render(scanLabel)

	info := strings.Join([]string{title, divider, dirLine, webLine, claudeLine, verLine, scanLine}, "\n")
	infoPanel := lipgloss.NewStyle().Padding(0, 2).Render(info)

	if m.width > 0 && m.width < 90 {
		return lipgloss.JoinVertical(lipgloss.Left, logoPanel, infoPanel) + "\n"
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, logoPanel, infoPanel) + "\n"
}
