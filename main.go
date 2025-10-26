package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/emilsto/jig/jira"
)

func main() {
	if handleCommandLine() {
		return
	}

	// TOOO: Handle epics flag
	helpFlag, _, oneshotFlag := parseFlags()

	mainConfig, err := getOrCreateConfig("config.toml")
	if err != nil {
		log.Fatal(err)
	}

	if helpFlag {
		printHelp()
		return
	}
	project, board, err := selectProjectAndBoard(mainConfig)
	if err != nil {
		log.Fatalf("Failed to select project/board: %v", err)
	}

	jiraCfg := jira.Config{
		BaseURL:  mainConfig.Api.Baseurl,
		AgileURL: mainConfig.Api.Agileurl,
		Email:    mainConfig.Api.Email,
		APIKey:   mainConfig.Api.Apikey,
	}

	jiraClient, err := jira.NewClient(jiraCfg)
	if err != nil {
		log.Fatalf("Failed to create Jira client: %v", err)
	}

	printDim("Using Project: %s (ID: %s), Board: %s (ID: %d)", project.Name, project.ID, board.Name, board.ID)
	fmt.Println()

	sprints, err := jiraClient.GetSprints(context.Background(), board.ID)
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
		config:     mainConfig,
		sprint:     sprint,
		reader:     bufio.NewReader(os.Stdin),
		oneshot:    oneshotFlag,
		jiraClient: jiraClient,
		board:      board,
	}

	runInteractiveLoop(ctx)
}
