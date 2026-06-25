package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// JdashConfig represents jdash's configuration
type JdashConfig struct {
	Sections []SectionConfig `yaml:"sections"`
	Layout   []string        `yaml:"layout,omitempty"`
}

// SectionConfig represents a single section configuration
type SectionConfig struct {
	Title   string   `yaml:"title"`
	Filters string   `yaml:"filters"`
	Layout  []string `yaml:"layout,omitempty"` // Optional per-section layout override
}

// LoadJdashConfig loads jdash config from ~/.config/jdash/config.yaml
// If the file doesn't exist, returns default config with login email substituted
func LoadJdashConfig(login string) (JdashConfig, error) {
	configPath := getJdashConfigPath()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
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
			Title:   "No Sprint Assigned",
			Filters: fmt.Sprintf(`assignee = "%s" AND sprint is EMPTY AND resolution = Unresolved`, login),
		},
	}

	return JdashConfig{
		Sections: sections,
		Layout:   getDefaultLayout(),
	}
}

// getDefaultLayout returns the default column layout
func getDefaultLayout() []string {
	return []string{"key", "type", "summary", "status", "assignee", "component", "sprint", "updated"}
}

// SaveExampleConfig writes an example config file to ~/.config/jdash/config.yaml.example
// This is a helper for users to get started
func SaveExampleConfig() error {
	configPath := getJdashConfigPath()
	examplePath := strings.Replace(configPath, "config.yaml", "config.yaml.example", 1)

	example := `# jdash configuration file
# Location: ~/.config/jdash/config.yaml

# Sections define the filtered views of issues
sections:
  - title: In Sprint
    filters: assignee = currentUser() AND sprint in openSprints()
    # Optional: per-section layout override
    layout: [key, summary, status, assignee, sprint]

  - title: No Sprint Assigned
    filters: assignee = currentUser() AND sprint is EMPTY AND resolution = Unresolved

  # Example: Bugs with priority-focused columns
  - title: Open Bugs
    filters: project = ACM AND type = Bug AND resolution = Unresolved
    layout: [key, summary, status, priority, reporter, updated]

  # Example: Team view scoped to component
  # - title: Team Sprint (ACM)
  #   filters: sprint in openSprints() AND component = "ACM"

# Global column layout (used when section doesn't specify its own)
# Available fields: key, type, summary, status, assignee, component, sprint, updated, priority, reporter
layout: [key, type, summary, status, assignee, component, updated]
`

	// Ensure directory exists
	dir := filepath.Dir(examplePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(examplePath, []byte(example), 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}

	return nil
}
