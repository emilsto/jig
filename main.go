package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func selectProjectAndBoard(config *Config) (*Project, *Board, error) {
	if len(config.Projects) == 0 {
		return nil, nil, fmt.Errorf("no projects configured")
	}

	jigrcPath := findJigRC()
	if jigrcPath != "" {
		jigrc, err := loadJigRC(jigrcPath)
		if err == nil {
			// Try to find matching project and board
			for i := range config.Projects {
				project := &config.Projects[i]

				// Match by ID or name
				if (jigrc.ProjectID != "" && project.ID == jigrc.ProjectID) ||
					(jigrc.ProjectName != "" && project.Name == jigrc.ProjectName) {

					// Find matching board
					for j := range project.Boards {
						board := &project.Boards[j]

						// Match by ID or name
						if (jigrc.BoardID != 0 && board.ID == jigrc.BoardID) ||
							(jigrc.BoardName != "" && board.Name == jigrc.BoardName) {

							printDim("Using .jigrc from: %s", jigrcPath)
							return project, board, nil
						}
					}
				}
			}

			printWarning(".jigrc found but no matching project/board in config")
		}
	}

	var selectedProject *Project
	var selectedBoard *Board

	// If only one project, auto-select it
	if len(config.Projects) == 1 {
		selectedProject = &config.Projects[0]
	} else {
		// Show project selection
		fmt.Println()
		printBold("Select Project:")
		for i, project := range config.Projects {
			fmt.Printf("  %d. %s (ID: %s)\n", i+1, printHighlight(project.Name), project.ID)
		}

		fmt.Println()
		printPrompt("Select project number")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, nil, err
		}

		selection, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || selection < 1 || selection > len(config.Projects) {
			return nil, nil, fmt.Errorf("invalid selection")
		}

		selectedProject = &config.Projects[selection-1]
	}

	// Now select board from the project
	if len(selectedProject.Boards) == 0 {
		return nil, nil, fmt.Errorf("no boards configured for project %s", selectedProject.Name)
	}

	// If only one board, auto-select it
	if len(selectedProject.Boards) == 1 {
		selectedBoard = &selectedProject.Boards[0]
	} else {
		// Show board selection
		fmt.Println()
		printBold("Select Board from %s:", selectedProject.Name)
		for i, board := range selectedProject.Boards {
			fmt.Printf("  %d. %s (ID: %d)\n", i+1, printHighlight(board.Name), board.ID)
		}

		fmt.Println()
		printPrompt("Select board number")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, nil, err
		}

		selection, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || selection < 1 || selection > len(selectedProject.Boards) {
			return nil, nil, fmt.Errorf("invalid selection")
		}

		selectedBoard = &selectedProject.Boards[selection-1]
	}

	return selectedProject, selectedBoard, nil
}

func handleInitJigrc(config *Config) error {
	project, board, err := selectProjectAndBoard(config)
	if err != nil {
		return fmt.Errorf("failed to select project/board: %v", err)
	}

	jigrc := &JigRC{
		ProjectName: project.Name,
		ProjectID:   project.ID,
		BoardName:   board.Name,
		BoardID:     board.ID,
	}

	if err := saveJigRC(jigrc); err != nil {
		return fmt.Errorf("failed to save .jigrc: %v", err)
	}

	currentDir, _ := os.Getwd()
	fmt.Println()
	printSuccess(".jigrc created in %s", printHighlight(currentDir))
	fmt.Printf("  Project: %s (ID: %s)\n", printHighlight(project.Name), project.ID)
	fmt.Printf("  Board: %s (ID: %d)\n", printHighlight(board.Name), board.ID)
	fmt.Println()

	return nil
}

type actionContext struct {
	config  *Config
	sprint  Sprint
	reader  *bufio.Reader
	oneshot bool
}

func handleCommandLine() bool {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			config, err := getOrCreateConfig("config.toml")
			if err != nil {
				log.Fatal(err)
			}
			if err := handleInitJigrc(config); err != nil {
				log.Fatal(err)
			}
			return true
		}
	}
	return false
}

func parseFlags() (help, epics, oneshot bool) {
	flagHelp := flag.Bool("h", false, "Show help message")
	flagEpics := flag.Bool("e", false, "Fetch epics from project")
	flagOneshot := flag.Bool("o", false, "Run once and exit (oneshot mode)")
	flag.Parse()
	return *flagHelp, *flagEpics, *flagOneshot
}

func getActiveIssues(ctx *actionContext) ([]Issue, error) {
	issues, err := getSprintIssues(ctx.config.Api.Agileurl, ctx.sprint.ID, ctx.config.Api.Email, ctx.config.Api.Apikey)
	if err != nil {
		return nil, err
	}

	activeIssues := []Issue{}
	for _, issue := range issues {
		if issue.Fields.Status.Name != "Done" {
			activeIssues = append(activeIssues, issue)
		}
	}
	return activeIssues, nil
}

func displayIssues(issues []Issue) {
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

type userAction struct {
	selection     int
	assignToSelf  bool
	changeStatus  bool
	createBranch  bool
	createSubtask bool
	listIssues    bool
}

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
	}

	input = strings.Join(fields, " ")

	selection, err := strconv.Atoi(input)
	if err != nil || selection < 0 || selection > maxSelection {
		return nil, fmt.Errorf("invalid selection")
	}

	action.selection = selection
	return action, nil
}

func handleAssignToSelf(ctx *actionContext, issue Issue) error {
	printInfo("Assigning %s to self", issue.Key)
	if err := assignToSelf(ctx.config.Api.Baseurl, issue.Key, ctx.config.Api.Email, ctx.config.Api.Apikey); err != nil {
		return err
	}
	printSuccess("Assigned %s to self", printHighlight(issue.Key))
	return nil
}

func handleCreateBranch(ctx *actionContext, issue Issue) error {
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

func handleChangeStatus(ctx *actionContext, issue Issue) error {
	printInfo("Getting available transitions for %s...", issue.Key)
	transitions, err := getTransitions(ctx.config.Api.Baseurl, issue.Key, ctx.config.Api.Email, ctx.config.Api.Apikey)
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
	if err := transitionIssue(ctx.config.Api.Baseurl, issue.Key, selectedTransition.ID, ctx.config.Api.Email, ctx.config.Api.Apikey); err != nil {
		return err
	}

	printSuccess("Status changed: %s → %s", printStatus(issue.Fields.Status.Name), printStatus(selectedTransition.To.Name))
	return nil
}

func handleCreateSubtask(ctx *actionContext, issue Issue) error {
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
	subtaskKey, err := createSubtask(ctx.config.Api.Baseurl, issue.Key, summary, ctx.config.Api.Email, ctx.config.Api.Apikey)
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

func handleShowDetails(ctx *actionContext, issue Issue) error {
	printInfo("Fetching issue details for %s...", issue.Key)
	issueDetails, err := getIssueDetails(ctx.config.Api.Baseurl, issue.Key, ctx.config.Api.Email, ctx.config.Api.Apikey)
	if err != nil {
		return err
	}
	printIssueDetails(issueDetails, extractDescription)
	return nil
}

func runInteractiveLoop(ctx *actionContext) {
	// Fetch and display issues once at startup
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

func main() {
	if handleCommandLine() {
		return
	}

	helpFlag, epicsFlag, oneshotFlag := parseFlags()

	config, err := getOrCreateConfig("config.toml")
	if err != nil {
		log.Fatal(err)
	}

	if helpFlag {
		printHelp()
		return
	}

	if epicsFlag {
		printInfo("Getting epics")
	}

	if config.Api.Email == "" {
		log.Fatal("Email not configured in config.toml. Please add 'email = \"your-email@example.com\"' under [api]")
	}

	project, board, err := selectProjectAndBoard(config)
	if err != nil {
		log.Fatalf("Failed to select project/board: %v", err)
	}

	printDim("Using Project: %s (ID: %s), Board: %s (ID: %d)", project.Name, project.ID, board.Name, board.ID)
	fmt.Println()

	sprints, err := getSprints(config.Api.Agileurl, board.ID, config.Api.Email, config.Api.Apikey)
	if err != nil {
		log.Fatalf("Failed to get sprints: %v", err)
	}

	if len(sprints) == 0 {
		fmt.Println("No active or future sprints found")
		return
	}

	sprint := sprints[0]
	printBold("Latest Sprint:")
	fmt.Printf("  - ID: %d, Name: %s, State: %s\n\n", sprint.ID, printHighlight(sprint.Name), printStatus(sprint.State))

	ctx := &actionContext{
		config:  config,
		sprint:  sprint,
		reader:  bufio.NewReader(os.Stdin),
		oneshot: oneshotFlag,
	}

	runInteractiveLoop(ctx)
}
