package main

import (
	"fmt"
	"os/exec"
	"sort"
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

	clrBg        = lipgloss.Color("#282A36")
	clrSelection = lipgloss.Color("#44475A")
	clrCursor    = lipgloss.Color("#6272A4")

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

// ── Sort / filter modes ──────────────────────────────────────────────────────

type sortMode int

const (
	sortRecent sortMode = iota
	sortSize
	sortTokens
	sortName
)

func (s sortMode) label() string {
	switch s {
	case sortSize:
		return "size ↓"
	case sortTokens:
		return "tokens ↓"
	case sortName:
		return "name A–Z"
	default:
		return "recent"
	}
}

type filterMode int

const (
	filterAll filterMode = iota
	filterHasData
	filterOrphaned
)

func (f filterMode) label() string {
	switch f {
	case filterHasData:
		return "has data"
	case filterOrphaned:
		return "orphaned"
	default:
		return "all"
	}
}

// ── App state ────────────────────────────────────────────────────────────────

type appState int

const (
	stateLoading appState = iota
	stateUpdatePrompt
	stateUpdating
	stateList
	stateConfirm
	stateDeleting
	stateDone
	stateCategories
	stateCategoryConfirm
)

const bannerLogo = ` ██████╗██╗      █████╗ ██╗   ██╗██████╗ ███████╗
██╔════╝██║     ██╔══██╗██║   ██║██╔══██╗██╔════╝
██║     ██║     ███████║██║   ██║██║  ██║█████╗
██║     ██║     ██╔══██║██║   ██║██║  ██║██╔══╝
╚██████╗███████╗██║  ██║╚██████╔╝██████╔╝███████╗
 ╚═════╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚═════╝╚══════╝
           C  L  E  A  N  E  R`

// ── Messages ─────────────────────────────────────────────────────────────────

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

type categoriesLoadedMsg struct{ categories []Category }

type categoryCleanDoneMsg struct {
	cleaned []string
	failed  []string
}

// ── Model ────────────────────────────────────────────────────────────────────

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
	purgeMode      bool // true = full purge via claude CLI
	deleted        []string
	failed         []string
	width          int
	deleteTotal        int
	deleteProgress     int
	deleteSelectedSnap map[int]bool
	claudeCLIVersion   string
	claudeCLIDetected  bool
	latestVersion       string
	hasUpdate           bool
	updateChecked       bool
	pendingUpdatePrompt bool
	sessionsReady       bool
	lastScanTime        time.Time
	rescanning          bool
	skipUpdateCheck     bool
	updatePromptIdx     int
	restartAfterUpdate  bool

	// sort / filter / search / help
	sortMode    sortMode
	filterMode  filterMode
	searchQuery string
	searching   bool
	showHelp    bool

	// expiry threshold (0 = no filter)
	expiryDays int
	expiryIdx  int // index into expiryOptions

	// category cleanup
	categories       []Category
	categorySelected map[string]bool
	categoryCursor   int
	categoryMode     bool // true when done screen shows category cleanup result

	dryRun bool // --dry-run: simulate deletions without touching files
}

func newModel(claudeDir, claudeJSONPath, projectsDir string) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(clrPurple)

	return model{
		state:            stateLoading,
		claudeDir:        claudeDir,
		claudeJSONPath:   claudeJSONPath,
		projectsDir:      projectsDir,
		selected:         make(map[int]bool),
		categorySelected: make(map[string]bool),
		spinner:          sp,
	}
}

// filteredSessions returns sessions after applying filter, search, and sort.
func (m model) filteredSessions() []Session {
	result := make([]Session, 0, len(m.sessions))
	q := strings.ToLower(m.searchQuery)

	for _, s := range m.sessions {
		switch m.filterMode {
		case filterHasData:
			if !s.HasData {
				continue
			}
		case filterOrphaned:
			if s.HasData {
				continue
			}
		}
		if q != "" {
			name := strings.ToLower(s.Name)
			path := strings.ToLower(s.ProjectPath)
			if !strings.Contains(name, q) && !strings.Contains(path, q) {
				continue
			}
		}
		if m.expiryDays > 0 && s.HasData {
			cutoff := time.Now().AddDate(0, 0, -m.expiryDays)
			if s.Modified.After(cutoff) {
				continue // too recent
			}
		}
		result = append(result, s)
	}

	switch m.sortMode {
	case sortSize:
		sort.Slice(result, func(i, j int) bool { return result[i].Size > result[j].Size })
	case sortTokens:
		sort.Slice(result, func(i, j int) bool { return result[i].TotalTokens > result[j].TotalTokens })
	case sortName:
		sort.Slice(result, func(i, j int) bool {
			ni := result[i].ProjectPath
			if ni == "" {
				ni = result[i].Name
			}
			nj := result[j].ProjectPath
			if nj == "" {
				nj = result[j].Name
			}
			return strings.ToLower(ni) < strings.ToLower(nj)
		})
	}

	return result
}

// ── Init ─────────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	claudeDir := m.claudeDir
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
		func() tea.Msg {
			return categoriesLoadedMsg{scanCategories(claudeDir)}
		},
	)
}

// ── Update ───────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
			if m.searching {
				// q inside search = add to query
				break
			}
			m.persistPrefs()
			return m, tea.Quit
		}
		if m.state == stateUpdatePrompt {
			return m.handleUpdatePromptKey(msg)
		}
		if m.state == stateConfirm {
			return m.handleConfirmKey(msg)
		}
		if m.state == stateCategories {
			return m.handleCategoryKey(msg)
		}
		if m.state == stateCategoryConfirm {
			return m.handleCategoryConfirmKey(msg)
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
		if m.skipUpdateCheck {
			return m, nil
		}
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
		if msg.err == nil {
			m.restartAfterUpdate = true
		}
		return m, tea.Quit

	case categoriesLoadedMsg:
		m.categories = msg.categories
		return m, nil

	case categoryCleanDoneMsg:
		m.deleted = msg.cleaned
		m.failed = msg.failed
		m.categoryMode = true
		m.state = stateDone
		m.categorySelected = make(map[string]bool)
		return m, nil

	case deleteDoneMsg:
		m.deleted = msg.deleted
		m.failed = msg.failed
		m.categoryMode = false
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

// ── Key handlers ─────────────────────────────────────────────────────────────

func (m model) handleUpdatePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h", "tab":
		m.updatePromptIdx = 0
	case "right", "l":
		m.updatePromptIdx = 1
	case "y", "Y":
		m.updatePromptIdx = 0
		return m.doUpdate()
	case "n", "N", "esc":
		m.pendingUpdatePrompt = false
		m.state = stateList
		return m, nil
	case "enter":
		if m.updatePromptIdx == 0 {
			return m.doUpdate()
		}
		m.pendingUpdatePrompt = false
		m.state = stateList
		return m, nil
	}
	return m, nil
}

func (m model) doUpdate() (tea.Model, tea.Cmd) {
	m.state = stateUpdating
	isMock := m.skipUpdateCheck
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			if isMock {
				time.Sleep(2 * time.Second)
				return updateDoneMsg{nil}
			}
			prepareWindowsUpdate()
			cmd := exec.Command("npm", "install", "-g", "claude-cleaner@latest")
			return updateDoneMsg{cmd.Run()}
		},
	)
}

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Search mode: intercept all printable input
	if m.searching {
		switch msg.String() {
		case "esc":
			m.searching = false
			m.searchQuery = ""
			m.cursor = 0
		case "enter":
			m.searching = false
		case "backspace", "ctrl+h":
			if len([]rune(m.searchQuery)) > 0 {
				runes := []rune(m.searchQuery)
				m.searchQuery = string(runes[:len(runes)-1])
				m.cursor = 0
			}
		default:
			if len(msg.Runes) > 0 {
				m.searchQuery += string(msg.Runes)
				m.cursor = 0
			}
		}
		return m, nil
	}

	sessions := m.filteredSessions()
	n := len(sessions)

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		} else if n > 0 {
			m.cursor = n - 1 // wrap to bottom
		}

	case "down", "j":
		if m.cursor < n-1 {
			m.cursor++
		} else {
			m.cursor = 0 // wrap to top
		}

	case "g":
		m.cursor = 0

	case "G":
		if n > 0 {
			m.cursor = n - 1
		}

	case "s":
		m.sortMode = (m.sortMode + 1) % 4
		m.cursor = 0
		m.persistPrefs()

	case "f":
		m.filterMode = (m.filterMode + 1) % 3
		m.cursor = 0
		m.persistPrefs()

	case "e":
		expiryOptions := []int{0, 7, 14, 30, 60, 90}
		m.expiryIdx = (m.expiryIdx + 1) % len(expiryOptions)
		m.expiryDays = expiryOptions[m.expiryIdx]
		m.cursor = 0
		m.persistPrefs()

	case "c":
		// Open category cleanup screen; rescan categories on entry
		claudeDir := m.claudeDir
		m.state = stateCategories
		m.categoryCursor = 0
		return m, tea.Batch(
			m.spinner.Tick,
			func() tea.Msg {
				return categoriesLoadedMsg{scanCategories(claudeDir)}
			},
		)

	case "/":
		m.searching = true
		m.searchQuery = ""
		m.cursor = 0

	case "?":
		m.showHelp = !m.showHelp

	case "esc":
		if m.searchQuery != "" || m.filterMode != filterAll || m.sortMode != sortRecent {
			m.searchQuery = ""
			m.filterMode = filterAll
			m.sortMode = sortRecent
			m.cursor = 0
		} else {
			m.showHelp = false
		}

	case "a":
		allOn := n > 0
		for _, s := range sessions {
			if !m.selected[s.Index] {
				allOn = false
				break
			}
		}
		for _, s := range sessions {
			m.selected[s.Index] = !allOn
		}

	case "n":
		// Unselect all
		m.selected = make(map[int]bool)

	case "o":
		// Select orphaned projects only (○ = no local data)
		m.selected = make(map[int]bool)
		for _, s := range m.sessions {
			if !s.HasData {
				m.selected[s.Index] = true
			}
		}

	case "d":
		// Reset everything to defaults
		m.sortMode = sortRecent
		m.filterMode = filterAll
		m.searchQuery = ""
		m.searching = false
		m.selected = make(map[int]bool)
		m.cursor = 0

	case " ":
		if n > 0 && m.cursor < n {
			idx := sessions[m.cursor].Index
			m.selected[idx] = !m.selected[idx]
		}
		return m, nil

	case "enter":
		if m.state == stateDone {
			m.deleted = nil
			m.failed = nil
			m.categoryMode = false
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
		}

	case "r", "R":
		return m.doRescan()

	case "x", "X":
		if n > 0 && m.cursor < n {
			return m.doPurgeDirect(sessions[m.cursor])
		}

	case "u", "U":
		if m.hasUpdate {
			prepareWindowsUpdate()
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

// ── Category key handlers ─────────────────────────────────────────────────────

func (m model) handleCategoryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.categories)
	switch msg.String() {
	case "up", "k":
		if m.categoryCursor > 0 {
			m.categoryCursor--
		} else if n > 0 {
			m.categoryCursor = n - 1
		}
	case "down", "j":
		if m.categoryCursor < n-1 {
			m.categoryCursor++
		} else {
			m.categoryCursor = 0
		}
	case "g":
		m.categoryCursor = 0
	case "G":
		if n > 0 {
			m.categoryCursor = n - 1
		}
	case " ":
		if n > 0 && m.categoryCursor < n {
			cat := m.categories[m.categoryCursor]
			if cat.Exists {
				m.categorySelected[cat.Key] = !m.categorySelected[cat.Key]
			}
		}
	case "a":
		allOn := true
		for _, cat := range m.categories {
			if cat.Exists && !m.categorySelected[cat.Key] {
				allOn = false
				break
			}
		}
		for _, cat := range m.categories {
			if cat.Exists {
				m.categorySelected[cat.Key] = !allOn
			}
		}
	case "n":
		m.categorySelected = make(map[string]bool)
	case "enter":
		count := 0
		for _, cat := range m.categories {
			if m.categorySelected[cat.Key] {
				count++
			}
		}
		if count > 0 {
			m.state = stateCategoryConfirm
			m.confirmIdx = 0
		}
	case "esc":
		m.state = stateList
		m.categorySelected = make(map[string]bool)
	}
	return m, nil
}

func (m model) handleCategoryConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.state = stateCategories
		m.confirmIdx = 0
		return m, nil
	case "left", "h", "tab":
		m.confirmIdx = 0
	case "right", "l":
		m.confirmIdx = 1
	case "y":
		m.confirmIdx = 1
		return m.doCategoryClean()
	case "enter":
		if m.confirmIdx == 1 {
			return m.doCategoryClean()
		}
		m.state = stateCategories
		m.confirmIdx = 0
		return m, nil
	}
	return m, nil
}

// ── Category clean action ─────────────────────────────────────────────────────

func (m model) doCategoryClean() (tea.Model, tea.Cmd) {
	if m.dryRun {
		var cleaned []string
		for _, cat := range m.categories {
			if m.categorySelected[cat.Key] {
				cleaned = append(cleaned, cat.Label)
			}
		}
		m.categoryMode = true
		m.state = stateDone
		m.deleted = cleaned
		m.failed = nil
		m.categorySelected = make(map[string]bool)
		return m, nil
	}

	m.state = stateDeleting
	selected := make(map[string]bool, len(m.categorySelected))
	for k, v := range m.categorySelected {
		selected[k] = v
	}
	cats := make([]Category, len(m.categories))
	copy(cats, m.categories)
	claudeDir := m.claudeDir

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			var cleaned, failed []string
			for _, cat := range cats {
				if !selected[cat.Key] {
					continue
				}
				if err := cleanCategory(cat, claudeDir); err != nil {
					failed = append(failed, cat.Label)
				} else {
					cleaned = append(cleaned, cat.Label)
				}
			}
			return categoryCleanDoneMsg{cleaned, failed}
		},
	)
}

// ── Preferences ───────────────────────────────────────────────────────────────

func (m model) persistPrefs() {
	writePrefs(m.claudeDir, Preferences{
		SortMode:   int(m.sortMode),
		FilterMode: int(m.filterMode),
		ExpiryDays: m.expiryDays,
	})
}

// ── Actions ───────────────────────────────────────────────────────────────────

func (m model) doRescan() (tea.Model, tea.Cmd) {
	m.sessionsReady = false
	m.pendingUpdatePrompt = false
	claudeJSONPath := m.claudeJSONPath
	projectsDir := m.projectsDir

	if m.state == stateList {
		m.rescanning = true
		return m, tea.Batch(
			m.spinner.Tick,
			func() tea.Msg {
				sessions, err := scanSessions(claudeJSONPath, projectsDir)
				cliVersion := DetectClaudeCLI()
				return rescanDoneMsg{sessions, err, cliVersion}
			},
		)
	}

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
	// Dry run: simulate without touching any files
	if m.dryRun {
		var deleted []string
		for _, s := range m.sessions {
			if m.selected[s.Index] {
				name := s.Name
				if s.ProjectPath != "" {
					name = s.ProjectPath
				}
				deleted = append(deleted, name)
			}
		}
		m.state = stateDone
		m.deleted = deleted
		m.failed = nil
		m.selected = make(map[int]bool)
		m.deleteTotal = 0
		m.deleteProgress = 0
		m.purgeMode = false
		return m, nil
	}

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

	return m, tea.Batch(
		m.spinner.Tick,
		nextDeleteCmd(m.sessions, snap, 0, 0, total, nil, nil, m.projectsDir),
	)
}

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
	if m.dryRun {
		name := s.Name
		if s.ProjectPath != "" {
			name = s.ProjectPath
		}
		m.state = stateDone
		m.deleted = []string{name}
		m.failed = nil
		return m, nil
	}

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

// ── Views ─────────────────────────────────────────────────────────────────────

func (m model) View() string {
	header := m.renderHeader()

	if m.showHelp {
		return header + m.viewHelp()
	}

	var body string
	switch m.state {
	case stateLoading:
		body = "\n  " + m.spinner.View() + " Scanning sessions…\n"
	case stateUpdating:
		body = "\n  " + m.spinner.View() + " Installing " +
			lipgloss.NewStyle().Foreground(clrCyan).Render("claude-cleaner@latest") +
			" via npm…\n\n  " + dimStyle.Render("Please wait, this may take a moment.")
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
	case stateCategories:
		body = m.viewCategories()
	case stateCategoryConfirm:
		body = m.viewCategoryConfirm()
	}

	return header + body
}

func (m model) viewHelp() string {
	cyan := func(s string) string { return lipgloss.NewStyle().Foreground(clrCyan).Bold(true).Render(s) }
	dim := func(s string) string { return dimStyle.Render(s) }

	row := func(keys, desc string) string {
		return fmt.Sprintf("  %-24s %s\n", cyan(keys), dim(desc))
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(clrPurple).Bold(true).Render("Key bindings") + "\n")
	sb.WriteString("  " + dimStyle.Render(strings.Repeat("─", 44)) + "\n\n")

	sb.WriteString(row("↑/↓  j/k", "Navigate list"))
	sb.WriteString(row("g / G", "Jump to top / bottom"))
	sb.WriteString(row("space", "Toggle selection"))
	sb.WriteString(row("a", "Select / deselect all (visible)"))
	sb.WriteString(row("n", "Unselect all"))
	sb.WriteString(row("o", "Select all orphaned projects (○)"))
	sb.WriteString(row("d", "Reset sort / filter / search / selection"))
	sb.WriteString(row("enter", "Confirm delete (when items selected)"))
	sb.WriteString(row("p", "Purge mode (full claude project purge)"))
	sb.WriteString(row("x", "Force-purge at cursor — no confirm"))
	sb.WriteString("\n")
	sb.WriteString(row("s", "Cycle sort: recent → size → tokens → name"))
	sb.WriteString(row("f", "Cycle filter: all → has data → orphaned"))
	sb.WriteString(row("e", "Cycle expiry: off → 7d → 14d → 30d → 60d → 90d"))
	sb.WriteString(row("/", "Search by project name / path"))
	sb.WriteString(row("c", "Open category cleanup (debug, telemetry, history…)"))
	sb.WriteString(row("esc", "Clear search / filter / sort  (or close help)"))
	sb.WriteString("\n")
	sb.WriteString(row("r", "Rescan / refresh project list"))
	sb.WriteString(row("u", "Update claude-cleaner in-place"))
	sb.WriteString(row("?", "Toggle this help"))
	sb.WriteString(row("q / ctrl+c", "Quit"))

	sb.WriteString("\n  " + dimStyle.Render("Press ? or esc to close"))
	return sb.String()
}

func (m model) viewUpdatePrompt() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(clrGreen).Bold(true).Render("⬆  New version available: v"+m.latestVersion) + "\n\n")
	sb.WriteString("  " + dimStyle.Render("Current: v"+version) + "\n\n")
	sb.WriteString("  Update now via " + lipgloss.NewStyle().Foreground(clrCyan).Render("npm install -g claude-cleaner@latest") + "?\n\n")

	yes := dimStyle.Render("[ Y ]  Yes, update now")
	no := dimStyle.Render("[ N ]  No, skip")
	if m.updatePromptIdx == 0 {
		yes = lipgloss.NewStyle().Foreground(clrGreen).Bold(true).Render("[ Y ]  Yes, update now")
	} else {
		no = lipgloss.NewStyle().Foreground(clrFg).Bold(true).Render("[ N ]  No, skip")
	}
	sb.WriteString("  " + yes + "      " + no + "\n\n")
	sb.WriteString("  " + dimStyle.Render("←/→ select  enter confirm  y yes  n/esc skip"))
	return sb.String()
}

func (m model) viewList() string {
	sessions := m.filteredSessions()

	if len(sessions) == 0 {
		if m.rescanning {
			return "\n  " + m.spinner.View() + " Rescanning…\n"
		}
		msg := "No Claude project sessions found."
		if m.searchQuery != "" || m.filterMode != filterAll {
			msg = "No sessions match current filter / search."
		}
		return "\n  " + dimStyle.Render(msg) + "\n" +
			"\n  " + dimStyle.Render("esc clear filter  r rescan  q quit")
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

	// Sort / filter / search / expiry status bar
	var statusParts []string
	if m.sortMode != sortRecent {
		statusParts = append(statusParts, "sort: "+m.sortMode.label())
	}
	if m.filterMode != filterAll {
		statusParts = append(statusParts, "filter: "+m.filterMode.label())
	}
	if m.searching {
		statusParts = append(statusParts, "search: "+m.searchQuery+"▌")
	} else if m.searchQuery != "" {
		statusParts = append(statusParts, "search: "+m.searchQuery)
	}
	if m.expiryDays > 0 {
		statusParts = append(statusParts, fmt.Sprintf("expiry: >%dd", m.expiryDays))
	}
	if len(statusParts) > 0 {
		sb.WriteString("  " + lipgloss.NewStyle().Foreground(clrCyan).Render(strings.Join(statusParts, "   ")) + "\n\n")
	}

	// Column header
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

	for i, s := range sessions {
		isCursor := m.cursor == i
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

	// Selection summary with size + tokens
	selected := 0
	var selSize int64
	var selTokens int64
	var anySelTokens bool
	for _, s := range m.sessions {
		if m.selected[s.Index] {
			selected++
			selSize += s.Size
			if s.HasTokenData {
				selTokens += s.TotalTokens
				anySelTokens = true
			}
		}
	}

	selInfo := countStyle.Render(fmt.Sprintf("%d", selected)) + " selected"
	if selected > 0 {
		if selSize > 0 {
			selInfo += "  •  " + lipgloss.NewStyle().Foreground(clrCyan).Render(formatSize(selSize))
		}
		if anySelTokens {
			selInfo += "  •  " + lipgloss.NewStyle().Foreground(clrPurple).Render(formatTokens(selTokens)+" tok")
		}
	}

	expiryLabel := "off"
	if m.expiryDays > 0 {
		expiryLabel = fmt.Sprintf("%dd", m.expiryDays)
	}
	footer := fmt.Sprintf(
		"↑/↓ navigate  space select  a all  enter delete  s sort  f filter  e expiry:%s  c categories  / search  ? help  q quit    %s",
		expiryLabel, selInfo,
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
		var label string
		if m.categoryMode {
			label = fmt.Sprintf("%d category/categories cleaned", len(m.deleted))
			if m.dryRun {
				label = fmt.Sprintf("%d category/categories would be cleaned  (dry run — nothing was modified)", len(m.deleted))
			}
		} else {
			label = fmt.Sprintf("%d session(s) deleted", len(m.deleted))
			if m.dryRun {
				label = fmt.Sprintf("%d session(s) would be deleted  (dry run — nothing was modified)", len(m.deleted))
			}
		}
		sb.WriteString(fmt.Sprintf("\n  %s\n", successStyle.Render(label)))
	}
	sb.WriteString("\n  " + dimStyle.Render("enter back to list  q quit"))

	return sb.String()
}

func (m model) viewCategories() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(clrPurple).Bold(true).Render("Category Cleanup") + "\n")
	sb.WriteString("  " + dimStyle.Render(strings.Repeat("─", 60)) + "\n\n")

	if len(m.categories) == 0 {
		sb.WriteString("  " + m.spinner.View() + " Scanning…\n")
		return sb.String()
	}

	rowW := m.width
	if rowW < 82 {
		rowW = 82
	}

	for i, cat := range m.categories {
		isCursor := m.categoryCursor == i
		isSelected := m.categorySelected[cat.Key]

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

		status := lipgloss.NewStyle().Foreground(clrComment).Background(bg).Render("○")
		if cat.Exists {
			status = lipgloss.NewStyle().Foreground(clrGreen).Background(bg).Render("●")
		}

		label := lipgloss.NewStyle().Foreground(clrFg).Background(bg).Width(42).Render(truncate(cat.Label, 42))

		var info string
		switch cat.Key {
		case "json-orphans":
			if cat.FileCount > 0 {
				info = lipgloss.NewStyle().Foreground(clrCyan).Background(bg).Render(
					fmt.Sprintf("%d orphan entries", cat.FileCount))
			} else {
				info = dimStyle.Render("clean")
			}
		case "history-trim":
			if cat.Exists {
				info = lipgloss.NewStyle().Foreground(clrCyan).Background(bg).Render(formatSize(cat.Size))
			} else {
				info = dimStyle.Render("not found")
			}
		default:
			if cat.Exists && (cat.Size > 0 || cat.FileCount > 0) {
				sizeStr := formatSize(cat.Size)
				if cat.FileCount > 0 {
					info = lipgloss.NewStyle().Foreground(clrCyan).Background(bg).Render(
						fmt.Sprintf("%-10s (%d files)", sizeStr, cat.FileCount))
				} else {
					info = lipgloss.NewStyle().Foreground(clrCyan).Background(bg).Render(sizeStr)
				}
			} else {
				info = dimStyle.Render("empty")
			}
		}

		content := cur + check + " " + status + " " + label + "  " + info
		sb.WriteString(rowStyle.Width(rowW).Render(content) + "\n")
	}

	// Total selected size
	var selSize int64
	selCount := 0
	for _, cat := range m.categories {
		if m.categorySelected[cat.Key] {
			selCount++
			selSize += cat.Size
		}
	}
	selInfo := countStyle.Render(fmt.Sprintf("%d", selCount)) + " selected"
	if selCount > 0 && selSize > 0 {
		selInfo += "  •  " + lipgloss.NewStyle().Foreground(clrCyan).Render(formatSize(selSize))
	}

	footer := fmt.Sprintf("↑/↓ navigate  space select  a all  n unselect  enter clean selected  esc back to projects    %s", selInfo)
	sb.WriteString(helpStyle.Render(footer))
	return sb.String()
}

func (m model) viewCategoryConfirm() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  " + dangerStyle.Render("⚠  Will clean the following categories:") + "\n\n")

	var totalSize int64
	for _, cat := range m.categories {
		if !m.categorySelected[cat.Key] {
			continue
		}
		totalSize += cat.Size
		var detail string
		switch cat.Key {
		case "json-orphans":
			detail = fmt.Sprintf("%d orphan entries", cat.FileCount)
		case "history-trim":
			detail = fmt.Sprintf("trim to 500 lines  (%s)", formatSize(cat.Size))
		default:
			detail = formatSize(cat.Size)
		}
		sb.WriteString(fmt.Sprintf("    %s  %s  %s\n",
			checkOnStyle.Render("✓"),
			nameStyle.Render(truncate(cat.Label, 44)),
			sizeStyle.Render(detail),
		))
	}

	if totalSize > 0 {
		sb.WriteString(fmt.Sprintf("\n  Total: %s\n", sizeStyle.Render(formatSize(totalSize))))
	}
	sb.WriteString("\n  " + dimStyle.Render("Original source code and settings.json are NOT affected.") + "\n\n")

	no := dimStyle.Render("[ N ]  No, cancel")
	yes := dimStyle.Render("[ Y ]  Yes, clean")
	if m.confirmIdx == 0 {
		no = lipgloss.NewStyle().Foreground(clrFg).Bold(true).Render("[ N ]  No, cancel")
	} else {
		yes = lipgloss.NewStyle().Foreground(clrRed).Bold(true).Render("[ Y ]  Yes, clean")
	}
	sb.WriteString("  " + no + "      " + yes + "\n\n")
	sb.WriteString("  " + dimStyle.Render("←/→ or y/n select  enter confirm  esc back"))
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

	// Total stats across all projects
	var totalSize int64
	var totalTokens int64
	var anyTok bool
	for _, s := range m.sessions {
		totalSize += s.Size
		if s.HasTokenData {
			totalTokens += s.TotalTokens
			anyTok = true
		}
	}
	totalTokStr := "—"
	if anyTok {
		totalTokStr = formatTokens(totalTokens)
	}
	statsLine := label("Projects") + lipgloss.NewStyle().Foreground(clrFg).Render(
		fmt.Sprintf("%d  •  %s  •  %s tokens", len(m.sessions), formatSize(totalSize), totalTokStr),
	)

	lines := []string{title, divider, dirLine, webLine, claudeLine, verLine, scanLine, statsLine}
	if m.dryRun {
		dryBadge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFB86C")).
			Padding(0, 1).
			Render("DRY RUN — no files will be modified")
		lines = append(lines, dryBadge)
	}
	info := strings.Join(lines, "\n")
	infoPanel := lipgloss.NewStyle().Padding(0, 2).Render(info)

	if m.width > 0 && m.width < 90 {
		return lipgloss.JoinVertical(lipgloss.Left, logoPanel, infoPanel) + "\n"
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, logoPanel, infoPanel) + "\n"
}
