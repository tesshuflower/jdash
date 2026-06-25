package jira

import (
	"fmt"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// Client wraps the jira-cli client with our interface
type Client struct {
	client       *jira.Client
	installation string // "Cloud" or "Local"
}

// NewClient creates a new Jira client
func NewClient(cfg *jira.Config, installation string) (*Client, error) {
	client := jira.NewClient(*cfg, jira.WithTimeout(15*time.Second))
	return &Client{
		client:       client,
		installation: installation,
	}, nil
}

// SearchIssues searches for issues using JQL and returns the results
func (c *Client) SearchIssues(jql string, limit uint) ([]*jira.Issue, error) {
	var result *jira.SearchResult
	var err error

	// Dispatch to Cloud (v3) or Local (v2) API based on installation type
	if c.installation == "Cloud" {
		result, err = c.client.Search(jql, limit)
	} else {
		// Local uses v2 API with startAt parameter
		result, err = c.client.SearchV2(jql, 0, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}

	if result == nil {
		return []*jira.Issue{}, nil
	}

	return result.Issues, nil
}
