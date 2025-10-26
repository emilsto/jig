package main

import (
	"fmt"
	"strings"
	"os"
	"bufio"
	"strconv"
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


