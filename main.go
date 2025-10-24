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

func main() {

	if len(os.Args) > 1 && os.Args[1] == "h" {
		printHelp()
		return
	}

	config, err := getOrCreateConfig("config.toml")
	if err != nil {
		log.Fatal(err)
	}

	flagHelp := flag.Bool("h", false, "Show help message")
	flagEpics := flag.Bool("e", false, "Fetch epics from project")
	flagOneshot := flag.Bool("o", false, "Run once and exit (oneshot mode)")
	flag.Parse()

	if *flagHelp {
		printHelp()
		return
	}

	if *flagEpics {
		printInfo("Getting epics")
	}

	if config.Api.Email == "" {
		log.Fatal("Email not configured in config.toml. Please add 'email = \"your-email@example.com\"' under [api]")
	}

	printDim("Using Project ID: %s, Board ID: %d", config.Api.Projectid, config.Api.Boardid)
	fmt.Println()

	sprints, err := getSprints(config.Api.Agileurl, config.Api.Boardid, config.Api.Email, config.Api.Apikey)
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

	reader := bufio.NewReader(os.Stdin)

	// Main action loop
	for {
		issues, err := getSprintIssues(config.Api.Agileurl, sprint.ID, config.Api.Email, config.Api.Apikey)
		if err != nil {
			log.Fatalf("Failed to get sprint issues: %v", err)
		}

		activeIssues := []Issue{}
		for _, issue := range issues {
			if issue.Fields.Status.Name != "Done" {
				activeIssues = append(activeIssues, issue)
			}
		}

		if len(activeIssues) == 0 {
			fmt.Println("\nNo active items in this sprint")
			return
		}

		fmt.Println()
		printBold("Active Items (%d):", len(activeIssues))

		// Use table format for issues
		maxSummaryLen := 60
		printTableHeader(maxSummaryLen)
		for i, issue := range activeIssues {
			assignee := issue.Fields.Assignee.DisplayName
			if assignee == "" {
				assignee = "Unassigned"
			}
			printTableRow(i+1, issue.Key, issue.Fields.Summary, issue.Fields.Status.Name, assignee, maxSummaryLen)
		}

		// Interactive selection
		fmt.Println()
		printPrompt("Select action (number) + command suffix or 'h' for help")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read input: %v", err)
		}

		input = strings.TrimSpace(input)

		// Check if user wants help
		if input == "h" {
			printInteractiveHelp()
			if *flagOneshot {
				return
			}
			continue
		}

		// Check if user wants to pick/assign to self
		assignToSelfFlag := false
		changeStatusFlag := false
		createBranchFlag := false
		createSubtaskFlag := false
		if strings.HasSuffix(input, "-p") || strings.HasSuffix(input, " -p") {
			assignToSelfFlag = true
			input = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(input, "-p"), " "))
		} else if strings.HasSuffix(input, "-s") || strings.HasSuffix(input, " -s") {
			changeStatusFlag = true
			input = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(input, "-s"), " "))
		} else if strings.HasSuffix(input, "-g") || strings.HasSuffix(input, " -g") {
			createBranchFlag = true
			input = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(input, "-g"), " "))
		} else if strings.HasSuffix(input, "-su") || strings.HasSuffix(input, " -su") {
			createSubtaskFlag = true
			input = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(input, "-su"), " "))
		}

		selection, err := strconv.Atoi(input)
		if err != nil || selection < 0 || selection > len(activeIssues) {
			fmt.Println("Invalid selection")
			if *flagOneshot {
				return
			}
			continue
		}

		if selection == 0 {
			fmt.Println("Exiting")
			return
		}

		selectedIssue := activeIssues[selection-1]

		// If user wants to assign to self, do that and exit
		if assignToSelfFlag {
			printInfo("Assigning %s to self", selectedIssue.Key)
			err := assignToSelf(config.Api.Baseurl, selectedIssue.Key, config.Api.Email, config.Api.Apikey)
			if err != nil {
				log.Fatalf("Failed to assign to self: %v", err)
			}
			printSuccess("Assigned %s to self", printHighlight(selectedIssue.Key))
			if *flagOneshot {
				return
			}
			continue
		}

		// If user wants to create git branch for issue
		if createBranchFlag {
			fmt.Println()
			printPrompt("Enter meaningful description for git branch name")
			branchDesc, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Failed to read input: %v", err)
			}
			branchDesc = strings.TrimSpace(branchDesc)

			if branchDesc == "" {
				fmt.Println("Branch description cannot be empty")
				if *flagOneshot {
					return
				}
				continue
			}

			if err := createGitBranch(config.Git.Branchbase, selectedIssue.Key, branchDesc); err != nil {
				log.Fatalf("Failed to create git branch: %v", err)
			}

			fmt.Println()
			printSuccess("Complete")
			if *flagOneshot {
				return
			}
			continue
		}

		// If user wants to change status, show transitions
		if changeStatusFlag {
			printInfo("Getting available transitions for %s...", selectedIssue.Key)
			transitions, err := getTransitions(config.Api.Baseurl, selectedIssue.Key, config.Api.Email, config.Api.Apikey)
			if err != nil {
				log.Fatalf("Failed to get transitions: %v", err)
			}

			if len(transitions) == 0 {
				fmt.Println("No available transitions for this issue")
				if *flagOneshot {
					return
				}
				continue
			}

			fmt.Println()
			printBold("Available Transitions:")
			for i, transition := range transitions {
				fmt.Printf("  %d. %s → %s\n", i+1, printStatus(selectedIssue.Fields.Status.Name), printStatus(transition.To.Name))
			}

			fmt.Println()
			printPrompt("Select transition (number) or 0 to cancel")
			transitionInput, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Failed to read input: %v", err)
			}

			transitionInput = strings.TrimSpace(transitionInput)
			transitionSelection, err := strconv.Atoi(transitionInput)
			if err != nil || transitionSelection < 0 || transitionSelection > len(transitions) {
				fmt.Println("Invalid selection")
				if *flagOneshot {
					return
				}
				continue
			}

			if transitionSelection == 0 {
				fmt.Println("Cancelled")
				if *flagOneshot {
					return
				}
				continue
			}

			selectedTransition := transitions[transitionSelection-1]
			printInfo("Transitioning %s to %s...", selectedIssue.Key, selectedTransition.To.Name)
			err = transitionIssue(config.Api.Baseurl, selectedIssue.Key, selectedTransition.ID, config.Api.Email, config.Api.Apikey)
			if err != nil {
				log.Fatalf("Failed to transition issue: %v", err)
			}

			printSuccess("Status changed: %s → %s", printStatus(selectedIssue.Fields.Status.Name), printStatus(selectedTransition.To.Name))
			if *flagOneshot {
				return
			}
			continue
		}

		// If user wants to create subtask
		if createSubtaskFlag {
			// Get subtask summary
			printPrompt("Enter subtask summary")
			summary, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Failed to read input: %v", err)
			}
			summary = strings.TrimSpace(summary)

			if summary == "" {
				fmt.Println("Subtask summary cannot be empty")
				if *flagOneshot {
					return
				}
				continue
			}

			// Create subtask
			fmt.Println()
			printInfo("Creating subtask for %s...", selectedIssue.Key)
			subtaskKey, err := createSubtask(config.Api.Baseurl, selectedIssue.Key, summary, config.Api.Email, config.Api.Apikey)
			if err != nil {
				log.Fatalf("Failed to create subtask: %v", err)
			}

			printSuccess("Subtask created successfully: %s", printHighlight(subtaskKey))

			// Create git branch
			fmt.Println()
			printPrompt("Enter ticket description for branch name")
			branchDesc, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Failed to read input: %v", err)
			}
			branchDesc = strings.TrimSpace(branchDesc)

			if branchDesc == "" {
				fmt.Println("Branch description cannot be empty")
				if *flagOneshot {
					return
				}
				continue
			}

			if err := createGitBranch(config.Git.Branchbase, subtaskKey, branchDesc); err != nil {
				log.Fatalf("Failed to create git branch: %v", err)
			}

			fmt.Println()
			printSuccess("Complete")

			if *flagOneshot {
				return
			}
			continue
		}

		// No action specified - show issue details
		printInfo("Fetching issue details for %s...", selectedIssue.Key)
		issueDetails, err := getIssueDetails(config.Api.Baseurl, selectedIssue.Key, config.Api.Email, config.Api.Apikey)
		if err != nil {
			log.Fatalf("Failed to get issue details: %v", err)
		}

		printIssueDetails(issueDetails, extractDescription)

		if *flagOneshot {
			return
		}
	}
}
