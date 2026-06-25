package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"gopkg.in/yaml.v3"
)

// JiraCLIConfig represents the structure of jira-cli's config file
type JiraCLIConfig struct {
	Installation string `yaml:"installation"` // "Cloud" or "Local"
	Server       string `yaml:"server"`
	Login        string `yaml:"login"`
	AuthType     string `yaml:"auth_type"` // "basic", "bearer", or "mtls"
	Insecure     bool   `yaml:"insecure"`
	Project      struct {
		Key string `yaml:"key"`
	} `yaml:"project"`
	Board struct {
		ID int `yaml:"id"`
	} `yaml:"board"`
	MTLS struct {
		CACert     string `yaml:"ca_cert"`
		ClientCert string `yaml:"client_cert"`
		ClientKey  string `yaml:"client_key"`
	} `yaml:"mtls"`
}

// Load reads the jira-cli config and returns a jira.Config ready for NewClient
// Returns: (*jira.Config, installation, login, projectKey, error)
func Load() (*jira.Config, string, string, string, error) {
	// Locate config file
	configPath, err := locateConfigFile()
	if err != nil {
		return nil, "", "", "", fmt.Errorf("jira-cli config not found: %w\n\nPlease install and configure jira-cli first:\n  brew install jira-cli\n  jira init", err)
	}

	// Parse YAML
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("failed to read jira-cli config at %s: %w", configPath, err)
	}

	var cfg JiraCLIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, "", "", "", fmt.Errorf("failed to parse jira-cli config: %w", err)
	}

	// Validate required fields
	if cfg.Server == "" {
		return nil, "", "", "", fmt.Errorf("jira-cli config missing 'server' field")
	}
	if cfg.Login == "" {
		return nil, "", "", "", fmt.Errorf("jira-cli config missing 'login' field")
	}

	// Resolve API token
	apiToken := resolveAPIToken()
	if apiToken == "" {
		return nil, "", "", "", fmt.Errorf("JIRA API token not found.\n\nSet the JIRA_API_TOKEN environment variable:\n  export JIRA_API_TOKEN=your-token-here")
	}

	// Build jira.Config
	authType := jira.AuthType(cfg.AuthType)
	if authType == "" {
		authType = jira.AuthTypeBasic
	}

	jiraCfg := &jira.Config{
		Server:   cfg.Server,
		Login:    cfg.Login,
		APIToken: apiToken,
		AuthType: &authType,
		Insecure: &cfg.Insecure,
	}

	// Add MTLS config if auth type is mtls
	if authType == "mtls" {
		jiraCfg.MTLSConfig = jira.MTLSConfig{
			CaCert:     cfg.MTLS.CACert,
			ClientCert: cfg.MTLS.ClientCert,
			ClientKey:  cfg.MTLS.ClientKey,
		}
	}

	return jiraCfg, cfg.Installation, cfg.Login, cfg.Project.Key, nil
}

// locateConfigFile finds the jira-cli config file
func locateConfigFile() (string, error) {
	// 1. Check JIRA_CONFIG_FILE env var
	if path := os.Getenv("JIRA_CONFIG_FILE"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// 2. Check XDG_CONFIG_HOME
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		path := filepath.Join(xdgConfig, ".jira", ".config.yml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// 3. Check ~/.config/.jira/.config.yml (default)
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".config", ".jira", ".config.yml")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("config not found at any standard location")
}

// resolveAPIToken resolves the API token from environment
func resolveAPIToken() string {
	// For now, only check JIRA_API_TOKEN env var
	// Future: could check .netrc or OS keyring
	return os.Getenv("JIRA_API_TOKEN")
}
