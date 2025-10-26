package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emilsto/jig/jira"
)

// getActiveIssues fetches and filters issues for the current sprint
func getActiveIssues(ctx *actionContext) ([]jira.Issue, error) {
	issues, err := ctx.jiraClient.GetSprintIssues(context.Background(), ctx.sprint.ID)
	if err != nil {
		return nil, err
	}

	activeIssues := []jira.Issue{}
	for _, issue := range issues {
		if issue.Fields.Status.Name != "Done" {
			activeIssues = append(activeIssues, issue)
		}
	}
	return activeIssues, nil
}

// displayIssues prints the list of issues in a table
func displayIssues(issues []jira.Issue) {
	fmt.Println()
	printBold("Active Items (%d):", len(issues))

	maxSummaryLen := 60
	printTableHeader(maxSummaryLen)
	for i, issue := range issues {
		assignee := issue.Fields.Assignee.DisplayName
		if assignee == "" {
			assignee = "Unassigned"
		}
		printTableRow(i+1, issue.Key, issue.Fields.Summary, issue.Fields.Status.Name, assignee, maxSummaryLen)
	}
}

// parseUserInput parses the raw string from the user into a structured action
func parseUserInput(input string, maxSelection int) (*userAction, error) {
	input = strings.TrimSpace(input)

	action := &userAction{}

	if input == "-l" || input == "l" || input == "list" {
		action.listIssues = true
		return action, nil
	}

	fields := strings.Fields(input)
	if len(fields) == 0 {
		return action, nil
	}

	lastField := fields[len(fields)-1]

	switch lastField {
	case "-p":
		action.assignToSelf = true
		fields = fields[:len(fields)-1]
	case "-s":
		action.changeStatus = true
		fields = fields[:len(fields)-1]
	case "-g":
		action.createBranch = true
		fields = fields[:len(fields)-1]
	case "-su":
		action.createSubtask = true
		fields = fields[:len(fields)-1]
	case "-pa":
		action.getParents = true
		fields = fields[:len(fields)-1]
	}

	input = strings.Join(fields, " ")

	selection, err := strconv.Atoi(input)
	if err != nil || selection < 0 || selection > maxSelection {
		return nil, fmt.Errorf("invalid selection")
	}

	action.selection = selection
	return action, nil
}

// --- Action Handlers ---

func handleAssignToSelf(ctx *actionContext, issue jira.Issue) error {
	printInfo("Assigning %s to self", issue.Key)
	if err := ctx.jiraClient.AssignToSelf(context.Background(), issue.Key); err != nil {
		return err
	}
	printSuccess("Assigned %s to self", printHighlight(issue.Key))
	return nil
}

func handleCreateBranch(ctx *actionContext, issue jira.Issue) error {
	fmt.Println()
	printPrompt("Enter meaningful description for git branch name")
	branchDesc, err := ctx.reader.ReadString('\n')
	if err != nil {
		return err
	}
	branchDesc = strings.TrimSpace(branchDesc)

	if branchDesc == "" {
		return fmt.Errorf("branch description cannot be empty")
	}

	if err := createGitBranch(ctx.config.Git.Branchbase, issue.Key, branchDesc); err != nil {
		return err
	}

	fmt.Println()
	printSuccess("Complete")
	return nil
}

func handleChangeStatus(ctx *actionContext, issue jira.Issue) error {
	printInfo("Getting available transitions for %s...", issue.Key)
	transitions, err := ctx.jiraClient.GetTransitions(context.Background(), issue.Key)
	if err != nil {
		return err
	}

	if len(transitions) == 0 {
		fmt.Println("No available transitions for this issue")
		return nil
	}

	fmt.Println()
	printBold("Available Transitions:")
	for i, transition := range transitions {
		fmt.Printf("  %d. %s → %s\n", i+1, printStatus(issue.Fields.Status.Name), printStatus(transition.To.Name))
	}

	fmt.Println()
	printPrompt("Select transition (number) or 0 to cancel")
	transitionInput, err := ctx.reader.ReadString('\n')
	if err != nil {
		return err
	}

	transitionInput = strings.TrimSpace(transitionInput)
	transitionSelection, err := strconv.Atoi(transitionInput)
	if err != nil || transitionSelection < 0 || transitionSelection > len(transitions) {
		return fmt.Errorf("invalid selection")
	}

	if transitionSelection == 0 {
		fmt.Println("Cancelled")
		return nil
	}

	selectedTransition := transitions[transitionSelection-1]
	printInfo("Transitioning %s to %s...", issue.Key, selectedTransition.To.Name)
	if err := ctx.jiraClient.TransitionIssue(context.Background(), issue.Key, selectedTransition.ID); err != nil {
		return err
	}

	printSuccess("Status changed: %s → %s", printStatus(issue.Fields.Status.Name), printStatus(selectedTransition.To.Name))
	return nil
}

func handleCreateSubtask(ctx *actionContext, issue jira.Issue) error {
	printPrompt("Enter subtask summary")
	summary, err := ctx.reader.ReadString('\n')
	if err != nil {
		return err
	}
	summary = strings.TrimSpace(summary)

	if summary == "" {
		return fmt.Errorf("subtask summary cannot be empty")
	}

	fmt.Println()
	printInfo("Creating subtask for %s...", issue.Key)

	subtaskKey, err := ctx.jiraClient.CreateSubtask(context.Background(), issue.Key, summary)
	if err != nil {
		return err
	}

	printSuccess("Subtask created successfully: %s", printHighlight(subtaskKey))

	fmt.Println()
	printPrompt("Enter ticket description for branch name")
	branchDesc, err := ctx.reader.ReadString('\n')
	if err != nil {
		return err
	}
	branchDesc = strings.TrimSpace(branchDesc)

	if branchDesc == "" {
		return fmt.Errorf("branch description cannot be empty")
	}

	if err := createGitBranch(ctx.config.Git.Branchbase, subtaskKey, branchDesc); err != nil {
		return err
	}

	fmt.Println()
	printSuccess("Complete")
	return nil
}

func handleShowDetails(ctx *actionContext, issue jira.Issue) error {
	printInfo("Fetching issue details for %s...", issue.Key)
	issueDetails, err := ctx.jiraClient.GetIssueDetails(context.Background(), issue.Key)
	if err != nil {
		return err
	}
	printIssueDetails(issueDetails, jira.ExtractDescription)
	return nil
}
