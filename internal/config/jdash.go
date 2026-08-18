package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultLimit is the number of issues fetched per section when no limit is configured.
const DefaultLimit uint = 100

// JdashConfig represents jdash's configuration
type JdashConfig struct {
	Sections    []SectionConfig `yaml:"sections"`
	Layout      []string        `yaml:"layout,omitempty"`
	SprintField string          `yaml:"sprint_field,omitempty"`
	Limit       uint            `yaml:"limit,omitempty"` // Global default max issues per section (default 100)
}

// SectionConfig represents a single section configuration
type SectionConfig struct {
	Title   string   `yaml:"title"`
	Filters string   `yaml:"filters"`
	Layout  []string `yaml:"layout,omitempty"` // Optional per-section layout override
	Lazy    bool     `yaml:"lazy,omitempty"`   // If true, don't load this section until user navigates to it
	Limit   uint     `yaml:"limit,omitempty"`  // Max issues for this section; overrides global limit
}

// EffectiveLimit returns the limit to use for this section, resolving the
// precedence: section limit > global limit > DefaultLimit.
func (s SectionConfig) EffectiveLimit(globalLimit uint) uint {
	if s.Limit > 0 {
		return s.Limit
	}
	if globalLimit > 0 {
		return globalLimit
	}
	return DefaultLimit
}

// LoadJdashConfig loads jdash config from ~/.config/jdash/config.yaml
// If the file doesn't exist, creates it with defaults and returns default config
func LoadJdashConfig(login string) (JdashConfig, error) {
	configPath := getJdashConfigPath()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Write default config so user has a template to customize
		if writeErr := writeDefaultConfig(configPath, login); writeErr != nil {
			// Non-fatal: still return defaults even if write fails
			fmt.Fprintf(os.Stderr, "Warning: could not write default config: %v\n", writeErr)
		}
		// Return default config with login email substituted
		return getDefaultConfig(login), nil
	}

	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return JdashConfig{}, fmt.Errorf("failed to read jdash config at %s: %w", configPath, err)
	}

	var cfg JdashConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return JdashConfig{}, fmt.Errorf("failed to parse jdash config: %w", err)
	}

	// Validate config
	if len(cfg.Sections) == 0 {
		return JdashConfig{}, fmt.Errorf("jdash config must have at least one section")
	}

	for i, section := range cfg.Sections {
		if section.Title == "" {
			return JdashConfig{}, fmt.Errorf("section %d missing title", i)
		}
		if section.Filters == "" {
			return JdashConfig{}, fmt.Errorf("section %q missing filters", section.Title)
		}
	}

	// If no layout specified, use default
	if len(cfg.Layout) == 0 {
		cfg.Layout = getDefaultLayout()
	}

	return cfg, nil
}

// getJdashConfigPath returns the path to the jdash config file
func getJdashConfigPath() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "jdash", "config.yaml")
	}

	// Fall back to ~/.config/jdash/config.yaml
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "jdash", "config.yaml")
}

// getDefaultConfig returns the default configuration with login email substituted
func getDefaultConfig(login string) JdashConfig {
	// Build default sections with login email instead of currentUser()
	// which doesn't work in some Jira instances
	sections := []SectionConfig{
		{
			Title:   "In Sprint",
			Filters: fmt.Sprintf(`assignee = "%s" AND sprint in openSprints()`, login),
		},
		{
			Title:   "No Sprint / Future Sprint",
			Filters: fmt.Sprintf(`assignee = "%s" AND (sprint is EMPTY OR sprint in futureSprints()) AND resolution = Unresolved`, login),
		},
	}

	return JdashConfig{
		Sections:    sections,
		Layout:      getDefaultLayout(),
		SprintField: "customfield_10020", // Common default, but may vary by Jira instance
	}
}

// getDefaultLayout returns the default column layout
func getDefaultLayout() []string {
	return []string{"key", "type", "summary", "status", "assignee", "component", "sprint", "updated"}
}

// writeDefaultConfig writes a default config file with the user's login substituted
func writeDefaultConfig(configPath, login string) error {
	// Config template with login substitution
	template := `# jdash configuration file
# Auto-generated on first run. Edit this file to customize your views.
# Location: ~/.config/jdash/config.yaml

# Sections define the filtered views of issues
sections:
  - title: In Sprint
    filters: assignee = "%s" AND sprint in openSprints()
    # Optional: per-section layout override
    # layout: [key, summary, status, assignee, sprint]
    # Optional: per-section issue limit (overrides global limit)
    # limit: 200

  - title: No Sprint / Future Sprint
    filters: assignee = "%s" AND (sprint is EMPTY OR sprint in futureSprints()) AND resolution = Unresolved

  # Example: Bugs with priority-focused columns
  # - title: Open Bugs
  #   filters: project = ACM AND type = Bug AND resolution = Unresolved
  #   layout: [key, summary, status, priority, reporter, updated]

  # Example: Team view scoped to component with lazy loading and higher limit
  # - title: Team Sprint (ACM)
  #   filters: sprint in openSprints() AND component = "ACM"
  #   lazy: true  # Don't load until user switches to this section
  #   limit: 250  # Fetch more issues for busy team sections

# Global column layout (used when section doesn't specify its own)
# Available fields: key, type, summary, status, assignee, component, sprint, updated, created, priority, reporter, labels, resolution, fixversion, parent
layout: [key, type, summary, status, assignee, component, sprint, updated]

# Global max issues to fetch per section (default: 100, can be overridden per section)
# Press 'L' in the TUI to temporarily change the limit for the current session.
# limit: 100

# Sprint custom field ID (defaults to customfield_10020 if not specified)
# This is the most common value, but may vary by Jira instance.
# To find your sprint field ID, run: curl -u "user:token" "https://your-jira/rest/api/3/field" | grep -i sprint
sprint_field: customfield_10020
`

	// Substitute login email
	config := fmt.Sprintf(template, login, login)

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	return nil
}
