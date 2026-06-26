package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ankitpokhrel/jira-cli/pkg/browser"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/tesshuflower/jdash/internal/config"
	jiraClient "github.com/tesshuflower/jdash/internal/jira"
)

var (
	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	detailStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("250"))

	activeTabStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57")).
				Bold(true)

	searchBarStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 1)

	searchBarEditingStyle = lipgloss.NewStyle().
					BorderStyle(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("170")).
					Padding(0, 1)
)

// transitionItem wraps a jira.Transition for the list component
type transitionItem struct {
	transition *jira.Transition
}

func (i transitionItem) Title() string       { return i.transition.Name }
func (i transitionItem) Description() string { return "" }
func (i transitionItem) FilterValue() string { return i.transition.Name }

// sprintItem wraps a jira.Sprint for the list component
type sprintItem struct {
	sprint *jira.Sprint
}

func (i sprintItem) Title() string {
	if i.sprint.BoardID > 0 {
		return fmt.Sprintf("%s (%s) [board:%d]", i.sprint.Name, i.sprint.Status, i.sprint.BoardID)
	}
	return fmt.Sprintf("%s (%s)", i.sprint.Name, i.sprint.Status)
}
func (i sprintItem) Description() string { return "" }
func (i sprintItem) FilterValue() string { return i.sprint.Name }

// keyMap defines all key bindings for jdash
type keyMap struct {
	// Navigation
	SwitchSectionNext key.Binding
	SwitchSectionPrev key.Binding
	Navigate          key.Binding
	FirstItem         key.Binding
	LastItem          key.Binding

	// Actions
	Comment       key.Binding
	ChangeStatus  key.Binding
	MoveToSprint  key.Binding
	OpenBrowser   key.Binding
	CreateIssue   key.Binding

	// Other
	EditQuery   key.Binding
	Refresh     key.Binding
	RefreshAll  key.Binding
	Help        key.Binding
	Quit        key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.SwitchSectionNext, k.SwitchSectionPrev, k.Navigate, k.FirstItem, k.LastItem},
		{k.Comment, k.ChangeStatus, k.MoveToSprint, k.OpenBrowser, k.CreateIssue},
		{k.EditQuery, k.Refresh, k.RefreshAll, k.Help, k.Quit},
	}
}

func newKeyMap() keyMap {
	return keyMap{
		SwitchSectionNext: key.NewBinding(
			key.WithKeys("l", "right", "tab"),
			key.WithHelp("l/→/tab", "next section"),
		),
		SwitchSectionPrev: key.NewBinding(
			key.WithKeys("h", "left", "shift+tab"),
			key.WithHelp("h/←/⇧tab", "prev section"),
		),
		Navigate: key.NewBinding(
			key.WithKeys("j", "k", "up", "down"),
			key.WithHelp("j/k/↑/↓", "navigate"),
		),
		FirstItem: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "first"),
		),
		LastItem: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "last"),
		),
		Comment: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "comment"),
		),
		ChangeStatus: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "change status"),
		),
		MoveToSprint: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "move to sprint"),
		),
		OpenBrowser: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open in browser"),
		),
		CreateIssue: key.NewBinding(
			key.WithKeys("O"),
			key.WithHelp("O", "create issue"),
		),
		EditQuery: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "edit query"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		RefreshAll: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh all"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

type Model struct {
	sections            []sectionModel
	activeSectionIdx    int
	client              *jiraClient.Client
	serverURL           string
	projectKey          string
	width               int
	height              int
	editing             bool
	queryInput          textinput.Model
	commenting          bool
	commentInput        textarea.Model
	transitioning       bool
	transitionList      list.Model
	transitionIssueKey  string
	movingSprint        bool
	sprintList          list.Model
	sprintIssueKey      string
	help                help.Model
	keys                keyMap
}

type sectionModel struct {
	config  config.SectionConfig
	table   table.Model
	issues  []*jiraClient.EnrichedIssue
	loading bool
	loaded  bool     // Has this section been loaded at least once?
	err     error
	layout  []string // Column layout for this section
}

type sectionIssuesLoadedMsg struct {
	sectionIdx int
	issues     []*jiraClient.EnrichedIssue
	err        error
}

type commentAddedMsg struct {
	issueKey string
	err      error
}

type transitionsLoadedMsg struct {
	transitions []*jira.Transition
	issueKey    string
	err         error
}

type transitionDoneMsg struct {
	issueKey string
	err      error
}

type sprintsLoadedMsg struct {
	sprints    []*jira.Sprint
	issueKey   string
	err        error
}

type sprintMoveMsg struct {
	issueKey string
	err      error
}

func NewModel(client *jiraClient.Client, appCfg *config.AppConfig) Model {
	// Create a section for each config section
	sections := make([]sectionModel, len(appCfg.Jdash.Sections))
	for i, secCfg := range appCfg.Jdash.Sections {
		// Use section-specific layout if provided, otherwise global default
		layout := secCfg.Layout
		if len(layout) == 0 {
			layout = appCfg.Jdash.Layout
		}
		sections[i] = newSectionModel(secCfg, layout)
	}

	keys := newKeyMap()
	m := Model{
		sections:         sections,
		activeSectionIdx: 0,
		client:           client,
		serverURL:        appCfg.JiraCfg.Server,
		projectKey:       appCfg.ProjectKey,
		help:             help.New(),
		keys:             keys,
	}

	return m
}

func newSectionModel(secCfg config.SectionConfig, layout []string) sectionModel {
	// Build columns from layout
	columns := buildColumnsFromLayout(layout)

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return sectionModel{
		config:  secCfg,
		table:   t,
		layout:  layout,
		loading: true,
	}
}

// buildColumnsFromLayout creates table columns from layout config
func buildColumnsFromLayout(layout []string) []table.Column {
	// Column widths - sensible defaults for each field
	columnWidths := map[string]int{
		"key":        12,
		"type":       10,
		"summary":    50,
		"status":     15,
		"assignee":   20,
		"component":  15,
		"sprint":     20,
		"updated":    12,
		"created":    12,
		"priority":   10,
		"reporter":   20,
		"labels":     20,
		"resolution": 15,
		"fixversion": 15,
		"parent":     12,
	}

	// Column titles - how to display each field
	columnTitles := map[string]string{
		"key":        "Key",
		"type":       "Type",
		"summary":    "Summary",
		"status":     "Status",
		"assignee":   "Assignee",
		"component":  "Component",
		"sprint":     "Sprint",
		"updated":    "Updated",
		"created":    "Created",
		"priority":   "Priority",
		"reporter":   "Reporter",
		"labels":     "Labels",
		"resolution": "Resolution",
		"fixversion": "Fix Version",
		"parent":     "Parent",
	}

	columns := make([]table.Column, len(layout))
	for i, field := range layout {
		title := columnTitles[field]
		if title == "" {
			title = strings.Title(field) // Fallback for unknown fields
		}
		width := columnWidths[field]
		if width == 0 {
			width = 15 // Default width for unknown fields
		}
		columns[i] = table.Column{
			Title: title,
			Width: width,
		}
	}

	return columns
}

func (m Model) Init() tea.Cmd {
	// Fetch all sections in parallel, except lazy ones
	cmds := make([]tea.Cmd, 0, len(m.sections))
	for i := range m.sections {
		if !m.sections[i].config.Lazy {
			cmds = append(cmds, m.fetchSectionIssues(i))
		}
	}
	return tea.Batch(cmds...)
}

func (m Model) fetchSectionIssues(sectionIdx int) tea.Cmd {
	return func() tea.Msg {
		section := m.sections[sectionIdx]
		issues, err := m.client.SearchIssues(section.config.Filters, 100)
		return sectionIssuesLoadedMsg{
			sectionIdx: sectionIdx,
			issues:     issues,
			err:        err,
		}
	}
}

func (m Model) addComment(comment string) tea.Cmd {
	return func() tea.Msg {
		// Get the selected issue
		if m.activeSectionIdx < 0 || m.activeSectionIdx >= len(m.sections) {
			return commentAddedMsg{err: fmt.Errorf("no active section")}
		}

		activeSection := m.sections[m.activeSectionIdx]
		if len(activeSection.issues) == 0 {
			return commentAddedMsg{err: fmt.Errorf("no issues")}
		}

		cursor := activeSection.table.Cursor()
		if cursor < 0 || cursor >= len(activeSection.issues) {
			return commentAddedMsg{err: fmt.Errorf("invalid cursor")}
		}

		issue := activeSection.issues[cursor]
		err := m.client.AddComment(issue.Key, comment)
		return commentAddedMsg{
			issueKey: issue.Key,
			err:      err,
		}
	}
}

func (m Model) fetchTransitions(issueKey string) tea.Cmd {
	return func() tea.Msg {
		transitions, err := m.client.GetTransitions(issueKey)
		return transitionsLoadedMsg{
			transitions: transitions,
			issueKey:    issueKey,
			err:         err,
		}
	}
}

func (m Model) transitionIssue(transition *jira.Transition) tea.Cmd {
	return func() tea.Msg {
		err := m.client.TransitionIssue(m.transitionIssueKey, transition.ID.String(), transition.Name)
		return transitionDoneMsg{
			issueKey: m.transitionIssueKey,
			err:      err,
		}
	}
}

func (m Model) fetchSprints(issueKey string, boardID int) tea.Cmd {
	return func() tea.Msg {
		var sprints []*jira.Sprint
		var err error

		// If no board ID (issue has no sprint), fetch sprints from ALL boards
		if boardID == 0 {
			sprints, err = m.client.GetAllProjectSprints(m.projectKey)
		} else {
			sprints, err = m.client.GetBoardSprints(boardID)
		}

		return sprintsLoadedMsg{
			sprints:  sprints,
			issueKey: issueKey,
			err:      err,
		}
	}
}

func (m Model) moveToSprint(sprint *jira.Sprint) tea.Cmd {
	return func() tea.Msg {
		err := m.client.MoveIssueToSprint(m.sprintIssueKey, fmt.Sprintf("%d", sprint.ID))
		return sprintMoveMsg{
			issueKey: m.sprintIssueKey,
			err:      err,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)

		// Table gets ~60% of height, detail pane gets rest
		// Account for section tabs (2 lines) + borders
		tableHeight := (m.height * 6) / 10
		for i := range m.sections {
			m.sections[i].table.SetHeight(tableHeight - 6)
			m.sections[i].table.SetWidth(m.width - 4)
		}

	case tea.KeyPressMsg:
		// Handle editing mode keys first
		if m.editing {
			switch msg.String() {
			case "enter":
				// Apply the edited query
				newQuery := m.queryInput.Value()
				m.sections[m.activeSectionIdx].config.Filters = newQuery
				m.sections[m.activeSectionIdx].loading = true
				m.editing = false
				return m, m.fetchSectionIssues(m.activeSectionIdx)

			case "esc":
				// Cancel editing
				m.editing = false
				return m, nil

			default:
				// Forward all other keys to the text input
				m.queryInput, cmd = m.queryInput.Update(msg)
				return m, cmd
			}
		}

		// Handle commenting mode keys
		if m.commenting {
			switch msg.String() {
			case "ctrl+d":
				// Submit comment
				comment := m.commentInput.Value()
				m.commenting = false
				return m, m.addComment(comment)

			case "esc":
				// Cancel commenting
				m.commenting = false
				return m, nil

			default:
				// Forward all other keys to the textarea
				m.commentInput, cmd = m.commentInput.Update(msg)
				return m, cmd
			}
		}

		// Handle transitioning mode keys
		if m.transitioning {
			switch msg.String() {
			case "enter":
				// Only select if NOT filtering - let list handle enter while filtering
				if m.transitionList.FilterState() != list.Filtering {
					if item, ok := m.transitionList.SelectedItem().(transitionItem); ok {
						m.transitioning = false
						return m, m.transitionIssue(item.transition)
					}
				} else {
					// Filtering - forward enter to list to exit filter mode
					var cmd tea.Cmd
					m.transitionList, cmd = m.transitionList.Update(msg)
					return m, cmd
				}
				return m, nil

			case "esc":
				// If filtering, let list handle esc (clears filter), otherwise cancel
				if m.transitionList.FilterState() == list.Filtering {
					var cmd tea.Cmd
					m.transitionList, cmd = m.transitionList.Update(msg)
					return m, cmd
				}
				m.transitioning = false
				return m, nil

			default:
				// Forward all other keys to list (handles j/k, filtering, etc)
				var cmd tea.Cmd
				m.transitionList, cmd = m.transitionList.Update(msg)
				return m, cmd
			}
		}

		// Handle movingSprint mode keys
		if m.movingSprint {
			switch msg.String() {
			case "enter":
				// Only select if NOT filtering - let list handle enter while filtering
				if m.sprintList.FilterState() != list.Filtering {
					if item, ok := m.sprintList.SelectedItem().(sprintItem); ok {
						m.movingSprint = false
						return m, m.moveToSprint(item.sprint)
					}
				} else {
					// Filtering - forward enter to list to exit filter mode
					var cmd tea.Cmd
					m.sprintList, cmd = m.sprintList.Update(msg)
					return m, cmd
				}
				return m, nil

			case "esc":
				// If filtering, let list handle esc (clears filter), otherwise cancel
				if m.sprintList.FilterState() == list.Filtering {
					var cmd tea.Cmd
					m.sprintList, cmd = m.sprintList.Update(msg)
					return m, cmd
				}
				m.movingSprint = false
				return m, nil

			default:
				// Forward all other keys to list (handles j/k, filtering, etc)
				var cmd tea.Cmd
				m.sprintList, cmd = m.sprintList.Update(msg)
				return m, cmd
			}
		}

		// Normal mode keys
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "?":
			// Toggle help
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case "/":
			// Enter query editing mode
			if m.activeSectionIdx >= 0 && m.activeSectionIdx < len(m.sections) {
				m.editing = true
				m.queryInput = textinput.New()
				m.queryInput.SetValue(m.sections[m.activeSectionIdx].config.Filters)
				m.queryInput.Focus()
				m.queryInput.SetWidth(m.width - 4)
			}
			return m, nil

		case "c":
			// Enter comment mode
			if m.activeSectionIdx >= 0 && m.activeSectionIdx < len(m.sections) {
				activeSection := m.sections[m.activeSectionIdx]
				if len(activeSection.issues) > 0 {
					m.commenting = true
					m.commentInput = textarea.New()
					m.commentInput.Focus()
					m.commentInput.SetWidth(m.width - 8)
					m.commentInput.SetHeight(10)
				}
			}
			return m, nil

		case "s":
			// Enter status change mode
			if m.activeSectionIdx >= 0 && m.activeSectionIdx < len(m.sections) {
				activeSection := m.sections[m.activeSectionIdx]
				if len(activeSection.issues) > 0 {
					cursor := activeSection.table.Cursor()
					if cursor >= 0 && cursor < len(activeSection.issues) {
						issue := activeSection.issues[cursor]
						return m, m.fetchTransitions(issue.Key)
					}
				}
			}
			return m, nil

		case "m":
			// Enter sprint move mode
			if m.activeSectionIdx >= 0 && m.activeSectionIdx < len(m.sections) {
				activeSection := m.sections[m.activeSectionIdx]
				if len(activeSection.issues) > 0 {
					cursor := activeSection.table.Cursor()
					if cursor >= 0 && cursor < len(activeSection.issues) {
						issue := activeSection.issues[cursor]

						// Create empty list with loading message immediately
						delegate := list.NewDefaultDelegate()
						delegate.ShowDescription = false
						delegate.SetSpacing(0)
						// Calculate available height: detail pane gets ~40% of screen minus borders/padding
						detailHeight := (m.height * 4) / 10
						listHeight := detailHeight - 8 // Account for borders and padding
						if listHeight < 5 {
							listHeight = 5 // Minimum height
						}
						m.sprintList = list.New([]list.Item{}, delegate, m.width-8, listHeight)
						m.sprintList.Title = fmt.Sprintf("Move to Sprint - %s (Loading...)", issue.Key)
						m.sprintList.SetShowHelp(false)
						m.sprintIssueKey = issue.Key
						m.movingSprint = true

						// Fetch sprints in background
						return m, m.fetchSprints(issue.Key, issue.BoardID)
					}
				}
			}
			return m, nil

		case "h", "left", "shift+tab":
			// Previous section
			if m.activeSectionIdx > 0 {
				m.activeSectionIdx--
				// Lazy load if this is the first visit
				newSection := m.sections[m.activeSectionIdx]
				if newSection.config.Lazy && !newSection.loaded {
					m.sections[m.activeSectionIdx].loading = true
					return m, m.fetchSectionIssues(m.activeSectionIdx)
				}
			}
			return m, nil

		case "l", "right", "tab":
			// Next section
			if m.activeSectionIdx < len(m.sections)-1 {
				m.activeSectionIdx++
				// Lazy load if this is the first visit
				newSection := m.sections[m.activeSectionIdx]
				if newSection.config.Lazy && !newSection.loaded {
					m.sections[m.activeSectionIdx].loading = true
					return m, m.fetchSectionIssues(m.activeSectionIdx)
				}
			}
			return m, nil

		case "o":
			// Open selected issue in browser
			return m, m.openSelectedIssueInBrowser()

		case "O":
			// Open browser to create new issue
			return m, m.openCreateIssueInBrowser()

		case "r":
			// Refresh current section
			if m.activeSectionIdx >= 0 && m.activeSectionIdx < len(m.sections) {
				m.sections[m.activeSectionIdx].loading = true
				return m, m.fetchSectionIssues(m.activeSectionIdx)
			}
			return m, nil

		case "R":
			// Refresh all sections
			cmds := make([]tea.Cmd, 0, len(m.sections))
			for i := range m.sections {
				m.sections[i].loading = true
				cmds = append(cmds, m.fetchSectionIssues(i))
			}
			return m, tea.Batch(cmds...)
		}

	case sectionIssuesLoadedMsg:
		if msg.sectionIdx >= 0 && msg.sectionIdx < len(m.sections) {
			m.sections[msg.sectionIdx].loading = false
			m.sections[msg.sectionIdx].loaded = true
			if msg.err != nil {
				m.sections[msg.sectionIdx].err = msg.err
			} else {
				m.sections[msg.sectionIdx].issues = msg.issues
				// Build rows using section's layout
				m.sections[msg.sectionIdx].table.SetRows(issuesToRows(msg.issues, m.sections[msg.sectionIdx].layout))
			}
		}
		return m, nil

	case commentAddedMsg:
		// TODO: Show feedback to user (success/error)
		// For now, just ignore - comment was added in background
		return m, nil

	case transitionsLoadedMsg:
		if msg.err != nil {
			// TODO: Show error to user
			return m, nil
		}
		// Create list items
		items := make([]list.Item, len(msg.transitions))
		for i, t := range msg.transitions {
			items[i] = transitionItem{transition: t}
		}
		// Create compact list delegate
		delegate := list.NewDefaultDelegate()
		delegate.ShowDescription = false
		delegate.SetSpacing(0)
		// Calculate available height: detail pane gets ~40% of screen minus borders/padding
		detailHeight := (m.height * 4) / 10
		listHeight := detailHeight - 8 // Account for borders and padding
		if listHeight < 5 {
			listHeight = 5 // Minimum height
		}
		// Create list
		m.transitionList = list.New(items, delegate, m.width-8, listHeight)
		m.transitionList.Title = fmt.Sprintf("Change Status - %s", msg.issueKey)
		m.transitionList.SetShowHelp(false)
		m.transitionIssueKey = msg.issueKey
		m.transitioning = true
		return m, nil

	case transitionDoneMsg:
		if msg.err == nil {
			// Re-fetch the active section to refresh the status
			return m, m.fetchSectionIssues(m.activeSectionIdx)
		}
		// TODO: Show error to user
		return m, nil

	case sprintsLoadedMsg:
		if msg.err != nil {
			// Update title to show error
			m.sprintList.Title = fmt.Sprintf("Move to Sprint - %s (Error: %v)", msg.issueKey, msg.err)
			return m, nil
		}
		// Update existing list with loaded items
		items := make([]list.Item, len(msg.sprints))
		for i, s := range msg.sprints {
			items[i] = sprintItem{sprint: s}
		}
		m.sprintList.SetItems(items)
		m.sprintList.Title = fmt.Sprintf("Move to Sprint - %s", msg.issueKey)
		return m, nil

	case sprintMoveMsg:
		if msg.err == nil {
			// Re-fetch the active section to refresh the sprint
			return m, m.fetchSectionIssues(m.activeSectionIdx)
		}
		// TODO: Show error to user
		return m, nil
	}

	// Forward unhandled messages to active components
	if m.transitioning {
		m.transitionList, cmd = m.transitionList.Update(msg)
		return m, cmd
	}

	if m.movingSprint {
		m.sprintList, cmd = m.sprintList.Update(msg)
		return m, cmd
	}

	// Update the active section's table
	if m.activeSectionIdx >= 0 && m.activeSectionIdx < len(m.sections) {
		m.sections[m.activeSectionIdx].table, cmd = m.sections[m.activeSectionIdx].table.Update(msg)
	}

	return m, cmd
}

func (m Model) View() tea.View {
	var content string

	// Render section tabs
	tabs := m.renderTabs()

	// Render active section
	if m.activeSectionIdx >= 0 && m.activeSectionIdx < len(m.sections) {
		activeSection := m.sections[m.activeSectionIdx]

		if activeSection.err != nil {
			sectionContent := errorStyle.Render(fmt.Sprintf("Error loading section:\n%v", activeSection.err))
			content = lipgloss.JoinVertical(lipgloss.Left, tabs, sectionContent)
		} else if activeSection.loading {
			sectionContent := "Loading issues..."
			content = lipgloss.JoinVertical(lipgloss.Left, tabs, sectionContent)
		} else {
			// Render table
			tableView := tableStyle.Render(activeSection.table.View())

			// Render detail pane
			detailView := m.renderDetailPane(activeSection)

			// Stack: tabs + table + detail
			content = lipgloss.JoinVertical(
				lipgloss.Left,
				tabs,
				tableView,
				detailView,
			)

			// Add hint at bottom if no issues
			if len(activeSection.issues) == 0 {
				hint := "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No issues found in this section")
				content = content + hint
			}
		}
	} else {
		content = "No sections configured"
	}

	// Create view with alt screen enabled for full terminal takeover
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) renderTabs() string {
	var tabs []string
	for i, section := range m.sections {
		style := tabStyle
		if i == m.activeSectionIdx {
			style = activeTabStyle
		}
		tabs = append(tabs, style.Render(section.config.Title))
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	var hint string
	var queryLine string

	if m.editing {
		// Editing mode hints
		hint = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("  Enter: apply  Esc: cancel")
		// Show text input in a bordered box
		boxWidth := m.width - 6
		if boxWidth < 20 {
			boxWidth = 20
		}
		queryLine = "\n" + searchBarEditingStyle.Width(boxWidth).Render(m.queryInput.View())
	} else {
		// Normal mode: show help
		hint = "  " + m.help.View(m.keys)

		// Show current section's query in a bordered box (read-only)
		if m.activeSectionIdx >= 0 && m.activeSectionIdx < len(m.sections) {
			activeSection := m.sections[m.activeSectionIdx]
			queryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)

			// Truncate query if too long (account for box padding/borders)
			query := activeSection.config.Filters
			maxWidth := m.width - 12
			if maxWidth > 0 && len(query) > maxWidth {
				query = query[:maxWidth-3] + "..."
			}

			boxWidth := m.width - 6
			if boxWidth < 20 {
				boxWidth = 20
			}
			queryLine = "\n" + searchBarStyle.Width(boxWidth).Render(queryStyle.Render(query))
		}
	}

	return tabBar + hint + queryLine + "\n"
}

// openSelectedIssueInBrowser opens the currently selected issue in the default browser
func (m Model) openSelectedIssueInBrowser() tea.Cmd {
	return func() tea.Msg {
		if m.activeSectionIdx < 0 || m.activeSectionIdx >= len(m.sections) {
			return nil
		}

		activeSection := m.sections[m.activeSectionIdx]
		if len(activeSection.issues) == 0 {
			return nil
		}

		cursor := activeSection.table.Cursor()
		if cursor < 0 || cursor >= len(activeSection.issues) {
			return nil
		}

		issue := activeSection.issues[cursor]
		url := fmt.Sprintf("%s/browse/%s", m.serverURL, issue.Key)

		// Open in browser (cross-platform via jira-cli's browser package)
		_ = browser.Browse(url) // Fire and forget

		return nil
	}
}

// openCreateIssueInBrowser opens the browser to create a new issue
func (m Model) openCreateIssueInBrowser() tea.Cmd {
	return func() tea.Msg {
		url := fmt.Sprintf("%s/secure/CreateIssue!default.jspa", m.serverURL)

		// Open in browser (cross-platform via jira-cli's browser package)
		_ = browser.Browse(url) // Fire and forget

		return nil
	}
}

func (m Model) renderDetailPane(section sectionModel) string {
	if len(section.issues) == 0 || section.table.Cursor() < 0 || section.table.Cursor() >= len(section.issues) {
		placeholder := "No issue selected"
		if len(section.issues) == 0 {
			placeholder = "No issues found"
		}
		return detailStyle.Width(m.width - 4).Render(placeholder)
	}

	issue := section.issues[section.table.Cursor()]

	// If commenting, show the textarea instead of details
	if m.commenting {
		var commentView strings.Builder
		commentView.WriteString(titleStyle.Render(fmt.Sprintf("Add Comment to %s", issue.Key)))
		commentView.WriteString("\n\n")
		commentView.WriteString(m.commentInput.View())
		commentView.WriteString("\n\n")
		commentView.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Ctrl+D: submit  Esc: cancel"))
		return detailStyle.Width(m.width - 4).Render(commentView.String())
	}

	// If transitioning, show the transition list instead of details
	if m.transitioning {
		return detailStyle.Width(m.width - 4).Render(m.transitionList.View())
	}

	// If moving sprint, show the sprint list instead of details
	if m.movingSprint {
		return detailStyle.Width(m.width - 4).Render(m.sprintList.View())
	}

	// Build detail content
	var details strings.Builder
	details.WriteString(titleStyle.Render(fmt.Sprintf("%s: %s", issue.Key, issue.Fields.Summary)))
	details.WriteString("\n\n")

	details.WriteString(fmt.Sprintf("Type:     %s\n", issue.Fields.IssueType.Name))
	details.WriteString(fmt.Sprintf("Status:   %s\n", issue.Fields.Status.Name))

	assignee := "Unassigned"
	if issue.Fields.Assignee.Name != "" {
		assignee = issue.Fields.Assignee.Name
	}
	details.WriteString(fmt.Sprintf("Assignee: %s\n", assignee))

	if len(issue.Fields.Components) > 0 {
		components := make([]string, len(issue.Fields.Components))
		for i, c := range issue.Fields.Components {
			components[i] = c.Name
		}
		details.WriteString(fmt.Sprintf("Components: %s\n", strings.Join(components, ", ")))
	}

	details.WriteString(fmt.Sprintf("Updated:  %s\n", issue.Fields.Updated))

	// Description (if present)
	if issue.Fields.Description != nil {
		details.WriteString("\n")
		desc, ok := issue.Fields.Description.(string)
		if ok && desc != "" {
			// Truncate long descriptions
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			details.WriteString(fmt.Sprintf("Description:\n%s", desc))
		}
	}

	return detailStyle.Width(m.width - 4).Render(details.String())
}

func issuesToRows(issues []*jiraClient.EnrichedIssue, layout []string) []table.Row {
	rows := make([]table.Row, len(issues))
	for i, issue := range issues {
		row := make(table.Row, len(layout))
		for j, field := range layout {
			row[j] = getIssueFieldValue(issue, field)
		}
		rows[i] = row
	}
	return rows
}

// getIssueFieldValue extracts the value for a specific field from an issue
func getIssueFieldValue(issue *jiraClient.EnrichedIssue, field string) string {
	switch field {
	case "key":
		return issue.Key

	case "type":
		return issue.Fields.IssueType.Name

	case "summary":
		summary := issue.Fields.Summary
		if len(summary) > 48 {
			summary = summary[:45] + "..."
		}
		return summary

	case "status":
		return issue.Fields.Status.Name

	case "assignee":
		if issue.Fields.Assignee.Name != "" {
			return issue.Fields.Assignee.Name
		}
		return "Unassigned"

	case "component":
		if len(issue.Fields.Components) > 0 {
			return issue.Fields.Components[0].Name
		}
		return ""

	case "sprint":
		return issue.SprintName

	case "updated":
		updated := issue.Fields.Updated
		if len(updated) > 10 {
			return updated[:10]
		}
		return updated

	case "created":
		created := issue.Fields.Created
		if len(created) > 10 {
			return created[:10]
		}
		return created

	case "priority":
		return issue.Fields.Priority.Name

	case "reporter":
		return issue.Fields.Reporter.Name

	case "labels":
		if len(issue.Fields.Labels) > 0 {
			// Show first label, or comma-separated if they fit
			return strings.Join(issue.Fields.Labels, ",")
		}
		return ""

	case "resolution":
		return issue.Fields.Resolution.Name

	case "fixversion":
		if len(issue.Fields.FixVersions) > 0 {
			return issue.Fields.FixVersions[0].Name
		}
		return ""

	case "parent":
		if issue.Fields.Parent != nil {
			return issue.Fields.Parent.Key
		}
		return ""

	default:
		// Unknown field - return empty
		return ""
	}
}
