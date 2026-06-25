package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
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
)

type Model struct {
	sections         []sectionModel
	activeSectionIdx int
	client           *jiraClient.Client
	serverURL        string
	projectKey       string
	width            int
	height           int
}

type sectionModel struct {
	config  config.SectionConfig
	table   table.Model
	issues  []*jira.Issue
	loading bool
	err     error
	layout  []string // Column layout for this section
}

type sectionIssuesLoadedMsg struct {
	sectionIdx int
	issues     []*jira.Issue
	err        error
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

	m := Model{
		sections:         sections,
		activeSectionIdx: 0,
		client:           client,
		serverURL:        appCfg.JiraCfg.Server,
		projectKey:       appCfg.ProjectKey,
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
	// Fetch all sections in parallel
	cmds := make([]tea.Cmd, len(m.sections))
	for i := range m.sections {
		cmds[i] = m.fetchSectionIssues(i)
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Table gets ~60% of height, detail pane gets rest
		// Account for section tabs (2 lines) + borders
		tableHeight := (m.height * 6) / 10
		for i := range m.sections {
			m.sections[i].table.SetHeight(tableHeight - 6)
			m.sections[i].table.SetWidth(m.width - 4)
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "h", "left", "shift+tab":
			// Previous section
			if m.activeSectionIdx > 0 {
				m.activeSectionIdx--
			}
			return m, nil

		case "l", "right", "tab":
			// Next section
			if m.activeSectionIdx < len(m.sections)-1 {
				m.activeSectionIdx++
			}
			return m, nil

		case "o":
			// Open selected issue in browser
			return m, m.openSelectedIssueInBrowser()

		case "O":
			// Open browser to create new issue
			return m, m.openCreateIssueInBrowser()
		}

	case sectionIssuesLoadedMsg:
		if msg.sectionIdx >= 0 && msg.sectionIdx < len(m.sections) {
			m.sections[msg.sectionIdx].loading = false
			if msg.err != nil {
				m.sections[msg.sectionIdx].err = msg.err
			} else {
				m.sections[msg.sectionIdx].issues = msg.issues
				// Build rows using section's layout
				m.sections[msg.sectionIdx].table.SetRows(issuesToRows(msg.issues, m.sections[msg.sectionIdx].layout))
			}
		}
		return m, nil
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
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("  h/l: switch sections  j/k: navigate  o: open in browser  q: quit")

	// Show current section's query
	var queryLine string
	if m.activeSectionIdx >= 0 && m.activeSectionIdx < len(m.sections) {
		activeSection := m.sections[m.activeSectionIdx]
		queryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)

		// Truncate query if too long
		query := activeSection.config.Filters
		maxWidth := m.width - 10
		if maxWidth > 0 && len(query) > maxWidth {
			query = query[:maxWidth-3] + "..."
		}
		queryLine = "\n" + queryStyle.Render("  "+query)
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

func issuesToRows(issues []*jira.Issue, layout []string) []table.Row {
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
func getIssueFieldValue(issue *jira.Issue, field string) string {
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
