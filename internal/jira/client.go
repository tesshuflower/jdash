// Package jira provides a Jira API client wrapping jira-cli for jdash.
package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

// Client wraps the jira-cli client with our interface
type Client struct {
	client        *jira.Client
	installation  string // "Cloud" or "Local"
	sprintField   string // Custom field ID for sprint (e.g., "customfield_10020")
	severityField string // Severity field key/ID (e.g., "severity" or "customfield_12345")
}

// EnrichedIssue wraps jira.Issue with additional parsed data
type EnrichedIssue struct {
	*jira.Issue
	SprintName  string
	SprintState string
	BoardID     int    // Board ID from the issue's sprint (0 if no sprint)
	Severity    string // Parsed severity value (from configured severity field)
}

// SearchResult holds the issues returned by a search and pagination metadata from Jira
type SearchResult struct {
	Issues []*EnrichedIssue
	Total  int  // Total number of matching issues (always 0 on Jira Cloud v3)
	IsLast bool // True when no further pages exist (reliable on all API versions)
}

// NewClient creates a new Jira client
func NewClient(cfg *jira.Config, installation, sprintField, severityField string) (*Client, error) {
	client := jira.NewClient(*cfg, jira.WithTimeout(15*time.Second))
	return &Client{
		client:        client,
		installation:  installation,
		sprintField:   sprintField,
		severityField: severityField,
	}, nil
}

// cloudPageSize is the maximum number of issues Jira Cloud returns per request,
// regardless of the maxResults parameter. Confirmed by probe testing.
const cloudPageSize = 100

// SearchIssues searches for issues using JQL up to limit results, paginating
// automatically when the server returns fewer issues than requested.
//
// Jira Cloud v3 hard-caps each page at 100 results regardless of maxResults.
// When limit > 100, this function makes multiple sequential requests using
// nextPageToken until limit is satisfied or isLast=true.
//
// Jira Server v2 uses startAt-based pagination with no known hard cap.
func (c *Client) SearchIssues(jql string, limit uint) (SearchResult, error) {
	if c.installation == "Cloud" {
		return c.searchIssuesCloud(jql, limit)
	}
	return c.searchIssuesServer(jql, limit)
}

// searchIssuesCloud fetches issues from Jira Cloud (v3 API) with nextPageToken pagination.
func (c *Client) searchIssuesCloud(jql string, limit uint) (SearchResult, error) {
	var allEnriched []*EnrichedIssue
	var lastTotal int
	nextPageToken := ""

	for {
		remaining := limit - uint(len(allEnriched))
		pageSize := uint(cloudPageSize)
		if remaining < pageSize {
			pageSize = remaining
		}

		path := fmt.Sprintf("/search/jql?jql=%s&maxResults=%d&fields=*all", url.QueryEscape(jql), pageSize)
		if nextPageToken != "" {
			path += "&nextPageToken=" + url.QueryEscape(nextPageToken)
		}

		enriched, total, isLast, token, err := c.fetchPage(path, true)
		if err != nil {
			return SearchResult{}, err
		}

		allEnriched = append(allEnriched, enriched...)
		lastTotal = total
		nextPageToken = token

		if isLast || uint(len(allEnriched)) >= limit {
			return SearchResult{Issues: allEnriched, Total: lastTotal, IsLast: isLast}, nil
		}
	}
}

// searchIssuesServer fetches issues from Jira Server (v2 API) with startAt pagination.
func (c *Client) searchIssuesServer(jql string, limit uint) (SearchResult, error) {
	var allEnriched []*EnrichedIssue
	var lastTotal int
	startAt := 0

	for {
		remaining := limit - uint(len(allEnriched))

		path := fmt.Sprintf("/search?jql=%s&startAt=%d&maxResults=%d", url.QueryEscape(jql), startAt, remaining)

		enriched, total, isLast, _, err := c.fetchPage(path, false)
		if err != nil {
			return SearchResult{}, err
		}

		allEnriched = append(allEnriched, enriched...)
		lastTotal = total
		startAt += len(enriched)

		if isLast || len(enriched) == 0 || uint(len(allEnriched)) >= limit {
			return SearchResult{Issues: allEnriched, Total: lastTotal, IsLast: isLast}, nil
		}
	}
}

// fetchPage makes a single HTTP search request and returns the parsed page of issues
// along with pagination metadata. cloud=true uses the v3 API; false uses v2.
func (c *Client) fetchPage(path string, cloud bool) (enriched []*EnrichedIssue, total int, isLast bool, nextPageToken string, err error) {
	var res *http.Response
	if cloud {
		res, err = c.client.Get(context.Background(), path, nil)
	} else {
		res, err = c.client.GetV2(context.Background(), path, nil)
	}
	if err != nil {
		return nil, 0, false, "", fmt.Errorf("failed to search issues: %w", err)
	}
	if res == nil {
		return nil, 0, true, "", nil
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, 0, false, "", fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, false, "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Pass 1: standard fields via jira.SearchResult
	var standardResult jira.SearchResult
	if err := json.Unmarshal(body, &standardResult); err != nil {
		return nil, 0, false, "", fmt.Errorf("failed to unmarshal standard fields: %w", err)
	}

	// Pass 2: pagination metadata + sprint custom fields
	var rawResult struct {
		Total         int    `json:"total"`
		IsLast        bool   `json:"isLast"`
		NextPageToken string `json:"nextPageToken"`
		Issues        []struct {
			Key    string                     `json:"key"`
			Fields map[string]json.RawMessage `json:"fields"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(body, &rawResult); err != nil {
		return nil, 0, false, "", fmt.Errorf("failed to unmarshal raw fields: %w", err)
	}

	enriched = make([]*EnrichedIssue, len(standardResult.Issues))
	for i, issue := range standardResult.Issues {
		enrichedIssue := &EnrichedIssue{Issue: issue}

		if i < len(rawResult.Issues) {
			if c.sprintField != "" {
				if sprintRaw, ok := rawResult.Issues[i].Fields[c.sprintField]; ok {
					var rawSprints []struct {
						ID      int    `json:"id"`
						Name    string `json:"name"`
						State   string `json:"state"`
						BoardID int    `json:"boardId"`
					}
					if err := json.Unmarshal(sprintRaw, &rawSprints); err == nil && len(rawSprints) > 0 {
						lastSprint := rawSprints[len(rawSprints)-1]
						enrichedIssue.SprintName = lastSprint.Name
						enrichedIssue.SprintState = lastSprint.State
						enrichedIssue.BoardID = lastSprint.BoardID
					}
				}
			}

			if c.severityField != "" {
				if severityRaw, ok := rawResult.Issues[i].Fields[c.severityField]; ok {
					enrichedIssue.Severity = parseDisplayFieldValue(severityRaw)
				}
			}
		}

		enriched[i] = enrichedIssue
	}

	return enriched, rawResult.Total, rawResult.IsLast, rawResult.NextPageToken, nil
}

// parseDisplayFieldValue extracts a user-friendly string from common Jira field shapes.
func parseDisplayFieldValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return plain
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err == nil {
		for _, key := range []string{"name", "value", "displayName"} {
			if v, ok := obj[key].(string); ok {
				return v
			}
		}
	}

	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		parts := make([]string, 0, len(arr))
		for _, item := range arr {
			value := parseDisplayFieldValue(item)
			if value != "" {
				parts = append(parts, value)
			}
		}
		return strings.Join(parts, ",")
	}

	return ""
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
