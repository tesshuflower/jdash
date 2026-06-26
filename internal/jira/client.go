// Package jira provides a Jira API client wrapping jira-cli for jdash.
package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// Client wraps the jira-cli client with our interface
type Client struct {
	client       *jira.Client
	installation string // "Cloud" or "Local"
	sprintField  string // Custom field ID for sprint (e.g., "customfield_10020")
}

// EnrichedIssue wraps jira.Issue with additional parsed data
type EnrichedIssue struct {
	*jira.Issue
	SprintName  string
	SprintState string
	BoardID     int // Board ID from the issue's sprint (0 if no sprint)
}

// NewClient creates a new Jira client
func NewClient(cfg *jira.Config, installation, sprintField string) (*Client, error) {
	client := jira.NewClient(*cfg, jira.WithTimeout(15*time.Second))
	return &Client{
		client:       client,
		installation: installation,
		sprintField:  sprintField,
	}, nil
}

// SearchIssues searches for issues using JQL and returns enriched results with sprint data
func (c *Client) SearchIssues(jql string, limit uint) ([]*EnrichedIssue, error) {
	// Build search URL (same as jira-cli's Search() method)
	var path string
	var res *http.Response
	var err error

	if c.installation == "Cloud" {
		// v3 API
		path = fmt.Sprintf("/search/jql?jql=%s&maxResults=%d&fields=*all", url.QueryEscape(jql), limit)
		res, err = c.client.Get(context.Background(), path, nil)
	} else {
		// v2 API
		path = fmt.Sprintf("/search?jql=%s&startAt=%d&maxResults=%d", url.QueryEscape(jql), 0, limit)
		res, err = c.client.GetV2(context.Background(), path, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}
	if res == nil {
		return []*EnrichedIssue{}, nil
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	// Read full response body for two-pass decode
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Pass 1: Decode into standard jira.SearchResult for standard fields
	var standardResult jira.SearchResult
	if err := json.Unmarshal(body, &standardResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal standard fields: %w", err)
	}

	// Pass 2: Extract sprint data from custom field
	var rawResult struct {
		Issues []struct {
			Key    string                            `json:"key"`
			Fields map[string]json.RawMessage `json:"fields"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(body, &rawResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw fields: %w", err)
	}

	// Build enriched results
	enriched := make([]*EnrichedIssue, len(standardResult.Issues))
	for i, issue := range standardResult.Issues {
		enrichedIssue := &EnrichedIssue{Issue: issue}

		// Extract sprint data from custom field if present
		if i < len(rawResult.Issues) && c.sprintField != "" {
			if sprintRaw, ok := rawResult.Issues[i].Fields[c.sprintField]; ok {
				// Parse sprint manually to get boardId (custom field uses "boardId", not "originBoardId")
				var rawSprints []struct {
					ID      int    `json:"id"`
					Name    string `json:"name"`
					State   string `json:"state"`
					BoardID int    `json:"boardId"` // Note: custom field uses "boardId", not "originBoardId"
				}
				if err := json.Unmarshal(sprintRaw, &rawSprints); err == nil && len(rawSprints) > 0 {
					// Use the last sprint (most recent/active)
					lastSprint := rawSprints[len(rawSprints)-1]
					enrichedIssue.SprintName = lastSprint.Name
					enrichedIssue.SprintState = lastSprint.State
					enrichedIssue.BoardID = lastSprint.BoardID
				}
			}
		}

		enriched[i] = enrichedIssue
	}

	return enriched, nil
}

// AddComment adds a comment to an issue
func (c *Client) AddComment(key, comment string) error {
	return c.client.AddIssueComment(key, comment, false)
}

// GetTransitions fetches available workflow transitions for an issue
func (c *Client) GetTransitions(key string) ([]*jira.Transition, error) {
	if c.installation == "Cloud" {
		return c.client.Transitions(key)
	}
	return c.client.TransitionsV2(key)
}

// TransitionIssue executes a workflow transition on an issue
func (c *Client) TransitionIssue(key, transitionID, transitionName string) error {
	req := &jira.TransitionRequest{
		Transition: &jira.TransitionRequestData{
			ID:   transitionID,
			Name: transitionName,
		},
	}
	_, err := c.client.Transition(key, req)
	return err
}

// GetBoards fetches all boards for a project
func (c *Client) GetBoards(projectKey string) ([]*jira.Board, error) {
	result, err := c.client.Boards(projectKey, "")
	if err != nil {
		return nil, err
	}
	return result.Boards, nil
}

// GetBoardSprints fetches active and future sprints for a board
func (c *Client) GetBoardSprints(boardID int) ([]*jira.Sprint, error) {
	// Fetch active and future sprints
	result, err := c.client.Sprints(boardID, "state=active,future", 0, 50)
	if err != nil {
		return nil, err
	}
	return result.Sprints, nil
}

// GetAllProjectSprints fetches active and future sprints from all boards in a project
func (c *Client) GetAllProjectSprints(projectKey string) ([]*jira.Sprint, error) {
	// Get all boards for the project
	boards, err := c.GetBoards(projectKey)
	if err != nil {
		return nil, err
	}

	var allSprints []*jira.Sprint
	for _, board := range boards {
		sprints, err := c.GetBoardSprints(board.ID)
		if err != nil {
			// Skip boards that error (might not have sprints)
			continue
		}
		// Inject board ID into each sprint for reference
		for _, sprint := range sprints {
			sprint.BoardID = board.ID
		}
		allSprints = append(allSprints, sprints...)
	}

	return allSprints, nil
}

// MoveIssueToSprint moves an issue to a sprint
func (c *Client) MoveIssueToSprint(issueKey, sprintID string) error {
	return c.client.SprintIssuesAdd(sprintID, issueKey)
}
