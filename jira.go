package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Sprint struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type SprintsResponse struct {
	Values []Sprint `json:"values"`
}

type Issue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
		Assignee struct {
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
	} `json:"fields"`
}

type DetailedIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string      `json:"summary"`
		Description interface{} `json:"description"`
		Status      struct {
			Name string `json:"name"`
		} `json:"status"`
		Assignee struct {
			DisplayName  string `json:"displayName"`
			EmailAddress string `json:"emailAddress"`
		} `json:"assignee"`
		IssueType struct {
			Name string `json:"name"`
		} `json:"issuetype"`
		Priority struct {
			Name string `json:"name"`
		} `json:"priority"`
		Labels  []string `json:"labels"`
		Created string   `json:"created"`
		Updated string   `json:"updated"`
	} `json:"fields"`
}

// extractDescription extracts text from Jira description field (handles both string and ADF format)
func extractDescription(desc interface{}) string {
	if desc == nil {
		return ""
	}

	// If it's already a string, return it
	if str, ok := desc.(string); ok {
		return str
	}

	// If it's an ADF object, extract text from content
	if adf, ok := desc.(map[string]interface{}); ok {
		content, ok := adf["content"].([]interface{})
		if !ok {
			return ""
		}

		var text strings.Builder
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				extractTextFromADF(itemMap, &text)
			}
		}
		return strings.TrimSpace(text.String())
	}

	return ""
}

// extractTextFromADF recursively extracts text from ADF nodes
func extractTextFromADF(node map[string]interface{}, builder *strings.Builder) {
	if text, ok := node["text"].(string); ok {
		builder.WriteString(text)
	}

	if content, ok := node["content"].([]interface{}); ok {
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				extractTextFromADF(itemMap, builder)
			}
		}
		// Add newline after paragraph-like nodes
		if nodeType, ok := node["type"].(string); ok {
			if nodeType == "paragraph" || nodeType == "heading" {
				builder.WriteString("\n")
			}
		}
	}
}

type IssuesResponse struct {
	Issues []Issue `json:"issues"`
}

type IssueType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subtask bool   `json:"subtask"`
}

func makeJiraRequest(method, url, email, apiKey string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(email, apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
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

func getSprints(agileURL string, boardID int, email, apiKey string) ([]Sprint, error) {
	url := fmt.Sprintf("%s/board/%d/sprint?state=active,future", agileURL, boardID)
	body, err := makeJiraRequest("GET", url, email, apiKey, nil)
	if err != nil {
		return nil, err
	}

	var sprintsResp SprintsResponse
	if err := json.Unmarshal(body, &sprintsResp); err != nil {
		return nil, err
	}

	return sprintsResp.Values, nil
}

func getSprintIssues(agileURL string, sprintID int, email, apiKey string) ([]Issue, error) {
	jql := url.QueryEscape("type in (Story, Task)")
	url := fmt.Sprintf("%s/sprint/%d/issue?jql=%s", agileURL, sprintID, jql)
	body, err := makeJiraRequest("GET", url, email, apiKey, nil)
	if err != nil {
		return nil, err
	}

	var issuesResp IssuesResponse
	if err := json.Unmarshal(body, &issuesResp); err != nil {
		return nil, err
	}

	return issuesResp.Issues, nil
}

func getSubtaskIssueTypeID(baseURL, parentKey, email, apiKey string) (string, error) {
	projectKey := parentKey[:strings.Index(parentKey, "-")]

	// Get project details including issue types
	url := fmt.Sprintf("%s/project/%s", baseURL, projectKey)
	body, err := makeJiraRequest("GET", url, email, apiKey, nil)
	if err != nil {
		return "", err
	}

	var project map[string]interface{}
	if err := json.Unmarshal(body, &project); err != nil {
		return "", err
	}

	issueTypesRaw, ok := project["issueTypes"].([]interface{})
	if !ok {
		return "", fmt.Errorf("no issueTypes in project")
	}

	for _, itRaw := range issueTypesRaw {
		it, ok := itRaw.(map[string]interface{})
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

func createSubtask(baseURL, parentKey, summary, email, apiKey string) (string, error) {
	projectKey := parentKey[:strings.Index(parentKey, "-")]

	subtaskTypeID, err := getSubtaskIssueTypeID(baseURL, parentKey, email, apiKey)
	if err != nil {
		return "", fmt.Errorf("failed to get subtask issue type: %v", err)
	}

	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]interface{}{
				"key": projectKey,
			},
			"parent": map[string]interface{}{
				"key": parentKey,
			},
			"summary": summary,
			"issuetype": map[string]interface{}{
				"id": subtaskTypeID,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := baseURL + "/issue"
	body, err := makeJiraRequest("POST", url, email, apiKey, bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	key, ok := result["key"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get issue key from response")
	}

	return key, nil
}

func getCurrentUser(baseURL, email, apiKey string) (string, error) {
	url := fmt.Sprintf("%s/myself", baseURL)
	body, err := makeJiraRequest("GET", url, email, apiKey, nil)
	if err != nil {
		return "", err
	}

	var user map[string]interface{}
	if err := json.Unmarshal(body, &user); err != nil {
		return "", err
	}

	accountId, ok := user["accountId"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get accountId from user response")
	}

	return accountId, nil
}

func assignToSelf(baseURL, issueKey, email, apiKey string) error {
	// Get current user's account ID
	accountId, err := getCurrentUser(baseURL, email, apiKey)
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	payload := map[string]interface{}{
		"accountId": accountId,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/issue/%s/assignee", baseURL, issueKey)
	_, err = makeJiraRequest("PUT", url, email, apiKey, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	return nil
}

type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		Name string `json:"name"`
	} `json:"to"`
}

type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

func getTransitions(baseURL, issueKey, email, apiKey string) ([]Transition, error) {
	url := fmt.Sprintf("%s/issue/%s/transitions", baseURL, issueKey)
	body, err := makeJiraRequest("GET", url, email, apiKey, nil)
	if err != nil {
		return nil, err
	}

	var transitionsResp TransitionsResponse
	if err := json.Unmarshal(body, &transitionsResp); err != nil {
		return nil, err
	}

	return transitionsResp.Transitions, nil
}

func transitionIssue(baseURL, issueKey, transitionID, email, apiKey string) error {
	payload := map[string]interface{}{
		"transition": map[string]interface{}{
			"id": transitionID,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/issue/%s/transitions", baseURL, issueKey)
	_, err = makeJiraRequest("POST", url, email, apiKey, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	return nil
}

func getIssueDetails(baseURL, issueKey, email, apiKey string) (*DetailedIssue, error) {
	url := fmt.Sprintf("%s/issue/%s", baseURL, issueKey)
	body, err := makeJiraRequest("GET", url, email, apiKey, nil)
	if err != nil {
		return nil, err
	}

	var issue DetailedIssue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}
