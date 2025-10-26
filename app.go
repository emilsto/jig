package main

import (
	"log"
	"fmt"
	"strings"
	"bufio"

	"github.com/emilsto/jig/jira"
)

type actionContext struct {
	config     *Config
	sprint     jira.Sprint
	reader     *bufio.Reader
	oneshot    bool
	jiraClient *jira.Client
	board      *Board
}

// Parsed command from user
type userAction struct {
	selection     int
	assignToSelf  bool
	changeStatus  bool
	createBranch  bool
	createSubtask bool
	listIssues    bool
	getParents    bool
}

func runInteractiveLoop(ctx *actionContext) {

	activeIssues, err := getActiveIssues(ctx)
	if err != nil {
		log.Fatalf("Failed to get sprint issues: %v", err)
	}

	if len(activeIssues) == 0 {
		fmt.Println("\nNo active items in this sprint")
		return
	}

	displayIssues(activeIssues)

	for {
		fmt.Println()
		printPrompt("Select action (number) + command suffix or 'h' for help")
		input, err := ctx.reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read input: %v", err)
		}

		input = strings.TrimSpace(input)

		if input == "h" {
			printInteractiveHelp()
			if ctx.oneshot {
				return
			}
			continue
		}

		action, err := parseUserInput(input, len(activeIssues))
		if err != nil {
			fmt.Println("Invalid selection")
			if ctx.oneshot {
				return
			}
			continue
		}

		// Handle list command
		if action.listIssues {
			activeIssues, err = getActiveIssues(ctx)
			if err != nil {
				log.Fatalf("Failed to get sprint issues: %v", err)
			}
			if len(activeIssues) == 0 {
				fmt.Println("\nNo active items in this sprint")
				return
			}
			displayIssues(activeIssues)
			if ctx.oneshot {
				return
			}
			continue
		}

		if action.selection == 0 {
			fmt.Println("Exiting")
			return
		}

		selectedIssue := activeIssues[action.selection-1]

		var actionErr error
		switch {
		case action.assignToSelf:
			actionErr = handleAssignToSelf(ctx, selectedIssue)
		case action.createBranch:
			actionErr = handleCreateBranch(ctx, selectedIssue)
		case action.changeStatus:
			actionErr = handleChangeStatus(ctx, selectedIssue)
		case action.createSubtask:
			actionErr = handleCreateSubtask(ctx, selectedIssue)
		default:
			actionErr = handleShowDetails(ctx, selectedIssue)
		}

		if actionErr != nil {
			log.Fatalf("Action failed: %v", actionErr)
		}

		if ctx.oneshot {
			return
		}
	}
}
