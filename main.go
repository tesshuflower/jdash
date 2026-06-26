// Package main is the entry point for jdash, a terminal UI for Jira.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/tesshuflower/jdash/internal/config"
	"github.com/tesshuflower/jdash/internal/jira"
	"github.com/tesshuflower/jdash/internal/ui"
)

func main() {
	// Load all config (jira-cli + jdash)
	appCfg, err := config.LoadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create Jira client
	client, err := jira.NewClient(appCfg.JiraCfg, appCfg.Installation, appCfg.Jdash.SprintField)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Jira client: %v\n", err)
		os.Exit(1)
	}

	// Create and run TUI
	model := ui.NewModel(client, appCfg)

	// Run the program (Init will fetch issues)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
