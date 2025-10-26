package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	BaseURL  string
	AgileURL string
	Email    string
	APIKey   string
}

type Client struct {
	baseURL    *url.URL
	agileURL   *url.URL
	email      string
	apiKey     string
	httpClient *http.Client
}

// creates a new Jira API client
func NewClient(cfg Config) (*Client, error) {
	base, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	agile, err := url.Parse(cfg.AgileURL)
	if err != nil {
		return nil, fmt.Errorf("invalid agile URL: %w", err)
	}

	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	if !strings.HasSuffix(agile.Path, "/") {
		agile.Path += "/"
	}

	return &Client{
		baseURL:    base,
		agileURL:   agile,
		email:      cfg.Email,
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}, nil
}

func (c *Client) makeRequest(ctx context.Context, method, urlStr string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.email, c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *Client) GetSprints(ctx context.Context, boardID int) ([]Sprint, error) {
	u, err := c.agileURL.Parse(fmt.Sprintf("board/%d/sprint?state=active,future", boardID))
	if err != nil {
		return nil, err
	}

	body, err := c.makeRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	var sprintsResp SprintsResponse
	if err := json.Unmarshal(body, &sprintsResp); err != nil {
		return nil, err
	}

	return sprintsResp.Values, nil
}

func (c *Client) GetSprintIssues(ctx context.Context, sprintID int) ([]Issue, error) {
	jql := url.QueryEscape("type in (Story, Task, Bug)")
	u, err := c.agileURL.Parse(fmt.Sprintf("sprint/%d/issue?jql=%s", sprintID, jql))
	if err != nil {
		return nil, err
	}

	body, err := c.makeRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	var issuesResp IssuesResponse
	if err := json.Unmarshal(body, &issuesResp); err != nil {
		return nil, err
	}

	return issuesResp.Issues, nil
}

func (c *Client) getSubtaskIssueTypeID(ctx context.Context, parentKey string) (string, error) {
	projectKey := parentKey[:strings.Index(parentKey, "-")]

	u, err := c.baseURL.Parse(fmt.Sprintf("project/%s", projectKey))
	if err != nil {
		return "", err
	}

	body, err := c.makeRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", err
	}

	var project map[string]any
	if err := json.Unmarshal(body, &project); err != nil {
		return "", err
	}

	issueTypesRaw, ok := project["issueTypes"].([]any)
	if !ok {
		return "", fmt.Errorf("no issueTypes in project")
	}

	for _, itRaw := range issueTypesRaw {
		it, ok := itRaw.(map[string]any)
		if !ok {
			continue
		}

		subtask, _ := it["subtask"].(bool)
		if subtask {
			id, ok := it["id"].(string)
			if ok {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("no subtask issue type found in project")
}

func (c *Client) CreateSubtask(ctx context.Context, parentKey, summary string) (string, error) {
	projectKey := parentKey[:strings.Index(parentKey, "-")]

	subtaskTypeID, err := c.getSubtaskIssueTypeID(ctx, parentKey)
	if err != nil {
		return "", fmt.Errorf("failed to get subtask issue type: %v", err)
	}

	payload := map[string]any{
		"fields": map[string]any{
			"project": map[string]any{
				"key": projectKey,
			},
			"parent": map[string]any{
				"key": parentKey,
			},
			"summary": summary,
			"issuetype": map[string]any{
				"id": subtaskTypeID,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	u, err := c.baseURL.Parse("issue")
	if err != nil {
		return "", err
	}

	body, err := c.makeRequest(ctx, "POST", u.String(), bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	key, ok := result["key"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get issue key from response")
	}

	return key, nil
}

func (c *Client) getCurrentUser(ctx context.Context) (string, error) {
	u, err := c.baseURL.Parse("myself")
	if err != nil {
		return "", err
	}

	body, err := c.makeRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", err
	}

	var user map[string]any
	if err := json.Unmarshal(body, &user); err != nil {
		return "", err
	}

	accountId, ok := user["accountId"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get accountId from user response")
	}

	return accountId, nil
}

func (c *Client) AssignToSelf(ctx context.Context, issueKey string) error {
	accountId, err := c.getCurrentUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	payload := map[string]any{
		"accountId": accountId,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	u, err := c.baseURL.Parse(fmt.Sprintf("issue/%s/assignee", issueKey))
	if err != nil {
		return err
	}

	_, err = c.makeRequest(ctx, "PUT", u.String(), bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetTransitions(ctx context.Context, issueKey string) ([]Transition, error) {
	u, err := c.baseURL.Parse(fmt.Sprintf("issue/%s/transitions", issueKey))
	if err != nil {
		return nil, err
	}

	body, err := c.makeRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	var transitionsResp TransitionsResponse
	if err := json.Unmarshal(body, &transitionsResp); err != nil {
		return nil, err
	}

	return transitionsResp.Transitions, nil
}

func (c *Client) TransitionIssue(ctx context.Context, issueKey, transitionID string) error {
	payload := map[string]any{
		"transition": map[string]any{
			"id": transitionID,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	u, err := c.baseURL.Parse(fmt.Sprintf("issue/%s/transitions", issueKey))
	if err != nil {
		return err
	}

	_, err = c.makeRequest(ctx, "POST", u.String(), bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetIssueDetails(ctx context.Context, issueKey string) (*DetailedIssue, error) {
	u, err := c.baseURL.Parse(fmt.Sprintf("issue/%s", issueKey))
	if err != nil {
		return nil, err
	}

	body, err := c.makeRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	var issue DetailedIssue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func (c *Client) GetAllProjects(ctx context.Context) ([]JiraProject, error) {
	u, err := c.baseURL.Parse("project")
	if err != nil {
		return nil, err
	}

	body, err := c.makeRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	var projects []JiraProject
	if err := json.Unmarshal(body, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func (c *Client) GetProjectBoards(ctx context.Context, projectKeyOrID string) ([]JiraBoard, error) {
	u, err := c.agileURL.Parse(fmt.Sprintf("board?projectKeyOrId=%s", projectKeyOrID))
	if err != nil {
		return nil, err
	}

	body, err := c.makeRequest(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	var boardsResp BoardsResponse
	if err := json.Unmarshal(body, &boardsResp); err != nil {
		return nil, err
	}

	return boardsResp.Values, nil
}
