package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
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
)

type Model struct {
	table      table.Model
	issues     []*jira.Issue
	client     *jiraClient.Client
	login      string // User's Jira login email
	projectKey string // Default project key from config
	width      int
	height     int
	err        error
	loading    bool
}

type issuesLoadedMsg struct {
	issues []*jira.Issue
	err    error
}

func NewModel(client *jiraClient.Client, login, projectKey string) Model {
	columns := []table.Column{
		{Title: "Key", Width: 12},
		{Title: "Type", Width: 10},
		{Title: "Summary", Width: 50},
		{Title: "Status", Width: 15},
		{Title: "Assignee", Width: 20},
		{Title: "Component", Width: 15},
		{Title: "Updated", Width: 12},
	}

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

	m := Model{
		table:      t,
		client:     client,
		login:      login,
		projectKey: projectKey,
		loading:    true,
	}

	return m
}

func (m Model) Init() tea.Cmd {
	// Return the fetch command to run on startup
	return m.fetchIssues()
}

func (m Model) fetchIssues() tea.Cmd {
	return func() tea.Msg {
		// Note: currentUser() doesn't work in some Jira instances, using login email from config
		// Also scope to project if configured
		jql := fmt.Sprintf(`assignee = "%s" AND resolution = Unresolved`, m.login)
		if m.projectKey != "" {
			jql = fmt.Sprintf(`project = %s AND %s`, m.projectKey, jql)
		}
		issues, err := m.client.SearchIssues(jql, 100)
		return issuesLoadedMsg{issues: issues, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Table gets ~60% of height, detail pane gets rest
		tableHeight := (m.height * 6) / 10
		m.table.SetHeight(tableHeight - 4) // Account for borders
		m.table.SetWidth(m.width - 4)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case issuesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.issues = msg.issues
		m.table.SetRows(issuesToRows(msg.issues))
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	var content string

	if m.err != nil {
		content = errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit", m.err))
	} else if m.loading {
		content = "Loading issues...\n\nPress q to quit"
	} else {
		// Always render the full UI (table + preview), even if empty
		// Render table
		tableView := tableStyle.Render(m.table.View())

		// Render detail pane
		detailView := m.renderDetailPane()

		// Stack vertically
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			tableView,
			detailView,
		)

		// Add hint at bottom if no issues
		if len(m.issues) == 0 {
			hint := "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No issues found. Press q to quit.")
			content = content + hint
		}
	}

	// Create view with alt screen enabled for full terminal takeover
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) renderDetailPane() string {
	if len(m.issues) == 0 || m.table.Cursor() < 0 || m.table.Cursor() >= len(m.issues) {
		placeholder := "No issue selected"
		if len(m.issues) == 0 {
			placeholder = "No issues found"
		}
		return detailStyle.Width(m.width - 4).Render(placeholder)
	}

	issue := m.issues[m.table.Cursor()]

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

func issuesToRows(issues []*jira.Issue) []table.Row {
	rows := make([]table.Row, len(issues))
	for i, issue := range issues {
		assignee := "Unassigned"
		if issue.Fields.Assignee.Name != "" {
			assignee = issue.Fields.Assignee.Name
		}

		component := ""
		if len(issue.Fields.Components) > 0 {
			component = issue.Fields.Components[0].Name
		}

		// Truncate summary if too long
		summary := issue.Fields.Summary
		if len(summary) > 48 {
			summary = summary[:45] + "..."
		}

		// Format updated date (just take first 10 chars for date)
		updated := issue.Fields.Updated
		if len(updated) > 10 {
			updated = updated[:10]
		}

		rows[i] = table.Row{
			issue.Key,
			issue.Fields.IssueType.Name,
			summary,
			issue.Fields.Status.Name,
			assignee,
			component,
			updated,
		}
	}
	return rows
}
