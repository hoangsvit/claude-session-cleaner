package main

import (
	"fmt"
	"strings"

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

type deleteDoneMsg struct {
	deleted []string
	failed  []string
}

type model struct {
	state        appState
	claudeDir    string
	projectsDir  string
	sessions     []Session
	selected     map[int]bool
	cursor       int
	spinner      spinner.Model
	confirmIdx   int // 0 = No (default), 1 = Yes
	deleted      []string
	failed       []string
	width        int
}

func newModel(claudeDir, projectsDir string) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(clrPurple)

	return model{
		state:       stateLoading,
		claudeDir:   claudeDir,
		projectsDir: projectsDir,
		selected:    make(map[int]bool),
		spinner:     sp,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			sessions, err := scanSessions(m.projectsDir)
			return sessionsLoadedMsg{sessions, err}
		},
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.state == stateConfirm {
			return m.handleConfirmKey(msg)
		}
		return m.handleListKey(msg)

	case sessionsLoadedMsg:
		if msg.err != nil {
			return m, tea.Quit
		}
		m.sessions = msg.sessions
		m.state = stateList
		return m, nil

	case deleteDoneMsg:
		m.deleted = msg.deleted
		m.failed = msg.failed
		m.state = stateDone
		m.selected = make(map[int]bool)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}


	return m, nil
}

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.sessions)

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < n-1 {
			m.cursor++
		}

	case " ":
		if n > 0 {
			idx := m.sessions[m.cursor].Index
			m.selected[idx] = !m.selected[idx]
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

	case "enter":
		if m.state == stateDone {
			return m, tea.Quit
		}
		count := 0
		for _, v := range m.selected {
			if v {
				count++
			}
		}
		if count > 0 {
			m.state = stateConfirm
			m.confirmIdx = 0
			return m, nil
		}
	}

	return m, nil
}

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "n":
		m.state = stateList
		m.confirmIdx = 0
		return m, nil

	case "left", "h", "tab":
		m.confirmIdx = 0
		return m, nil

	case "right", "l":
		m.confirmIdx = 1
		return m, nil

	case "y":
		m.confirmIdx = 1
		return m.doDelete()

	case "enter":
		if m.confirmIdx == 1 {
			return m.doDelete()
		}
		m.state = stateList
		m.confirmIdx = 0
		return m, nil
	}
	return m, nil
}

func (m model) doDelete() (tea.Model, tea.Cmd) {
	m.state = stateDeleting
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			var deleted, failed []string
			for _, s := range m.sessions {
				if !m.selected[s.Index] {
					continue
				}
				if err := safeRemove(m.projectsDir, s.Path); err != nil {
					failed = append(failed, s.Name)
				} else {
					deleted = append(deleted, s.Name)
				}
			}
			return deleteDoneMsg{deleted, failed}
		},
	)
}

func (m model) View() string {
	header := m.renderHeader()

	var body string
	switch m.state {
	case stateLoading:
		body = "\n  " + m.spinner.View() + " Scanning sessions…\n"
	case stateList:
		body = m.viewList()
	case stateConfirm:
		body = m.viewConfirm()
	case stateDeleting:
		body = "\n  " + m.spinner.View() + " Deleting…\n"
	case stateDone:
		body = m.viewDone()
	}

	return header + body
}

func (m model) viewList() string {
	if len(m.sessions) == 0 {
		return "\n  " + dimStyle.Render("No Claude project sessions found.") + "\n" +
			"\n  " + dimStyle.Render("q quit")
	}

	const (
		nameW = 44
		timeW = 14
		sizeW = 10
	)

	var sb strings.Builder
	sb.WriteString("\n")

	// Header row
	sb.WriteString(dimStyle.Render(fmt.Sprintf("       %-*s  %-*s  %s",
		nameW, "Name",
		timeW, "Last modified",
		"Size",
	)) + "\n")
	sb.WriteString(dimStyle.Render("  "+strings.Repeat("─", nameW+timeW+sizeW+12)) + "\n")

	rowW := m.width
	if rowW < 80 {
		rowW = 80
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

		name := lipgloss.NewStyle().Foreground(clrFg).Background(bg).Width(nameW).Render(truncate(s.Name, nameW))
		t := lipgloss.NewStyle().Foreground(clrComment).Background(bg).Width(timeW).Render(humanTime(s.Modified))
		sz := lipgloss.NewStyle().Foreground(clrCyan).Background(bg).Render(formatSize(s.Size))

		content := cur + check + "  " + name + "  " + t + "  " + sz
		sb.WriteString(rowStyle.Width(rowW).Render(content) + "\n")
	}

	selected := 0
	for _, v := range m.selected {
		if v {
			selected++
		}
	}

	footer := fmt.Sprintf(
		"↑/↓ navigate  space toggle  a select all  enter confirm  q quit    %s selected",
		countStyle.Render(fmt.Sprintf("%d", selected)),
	)
	sb.WriteString(helpStyle.Render(footer))

	return sb.String()
}

func (m model) viewConfirm() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  " + dangerStyle.Render("⚠  Will delete:") + "\n\n")

	var total int64
	for _, s := range m.sessions {
		if !m.selected[s.Index] {
			continue
		}
		total += s.Size
		sb.WriteString(fmt.Sprintf("    %s  %s  %s\n",
			checkOnStyle.Render("✓"),
			nameStyle.Render(truncate(s.Name, 44)),
			sizeStyle.Render(formatSize(s.Size)),
		))
	}

	sb.WriteString(fmt.Sprintf("\n  Total: %s\n\n", sizeStyle.Render(formatSize(total))))
	sb.WriteString("  " + dimStyle.Render("Deletes session history only. Source code is NOT affected.") + "\n\n")

	noStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 3)
	yesStyle := noStyle

	if m.confirmIdx == 0 {
		noStyle = noStyle.BorderForeground(clrPurple).Foreground(clrFg).Bold(true)
		yesStyle = yesStyle.BorderForeground(clrComment).Foreground(clrComment)
	} else {
		noStyle = noStyle.BorderForeground(clrComment).Foreground(clrComment)
		yesStyle = yesStyle.BorderForeground(clrRed).Foreground(clrRed).Bold(true)
	}

	btnNo := noStyle.Render("✗  No, cancel")
	btnYes := yesStyle.Render("✓  Yes, delete")
	sb.WriteString("  " + lipgloss.JoinHorizontal(lipgloss.Center, btnNo, "   ", btnYes) + "\n\n")
	sb.WriteString("  " + dimStyle.Render("←/→ select  enter confirm  esc back"))

	return sb.String()
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
	sb.WriteString("\n  " + dimStyle.Render("press enter or q to exit"))

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
	statusLine := label("Status") + lipgloss.NewStyle().Foreground(clrGreen).Bold(true).Render("● Ready")
	verLine := label("Version") + lipgloss.NewStyle().Foreground(clrComment).Render(version)

	info := strings.Join([]string{title, divider, dirLine, statusLine, verLine}, "\n")
	infoPanel := lipgloss.NewStyle().Padding(0, 2).Render(info)

	if m.width > 0 && m.width < 90 {
		return lipgloss.JoinVertical(lipgloss.Left, logoPanel, infoPanel) + "\n"
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, logoPanel, infoPanel) + "\n"
}
