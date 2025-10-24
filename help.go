package main

import "fmt"

func printHelp() {
	printBold("Jig - Jira Git Integration Tool")
	fmt.Println()
	printInfo("Usage:")
	fmt.Println("  jig [flags]")
	fmt.Println("  jig h")
	fmt.Println()
	printInfo("Commands:")
	fmt.Println("  h                     Show this help message")
	fmt.Println()
	printInfo("Flags:")
	fmt.Println("  -h                    Show this help message")
	fmt.Println("  -e                    Fetch epics from project")
	fmt.Println("  -o                    Run once and exit (oneshot mode)")
	fmt.Println()
	printInfo("Interactive Mode:")
	fmt.Println("  When run without -o flag, jig will:")
	fmt.Println("  1. Fetch the latest sprint and active issues")
	fmt.Println("  2. Allow you to select an issue to create a subtask")
	fmt.Println("  3. Loop back to step 1 after each action (continuous mode)")
	fmt.Println("  With -o flag, jig exits after completing one action")
	fmt.Println()
	printInfo("Examples:")
	fmt.Println("  jig                   Run in continuous interactive mode (loops after each action)")
	fmt.Println("  jig -o                Run once and exit (oneshot mode)")
	fmt.Println("  jig -e                Fetch epics")
	fmt.Println("  jig h                 Show help")
	fmt.Println("  jig -h                Show help")
}

func printInteractiveHelp() {
	fmt.Println()
	printInfo("Interactive Selection Help:")
	fmt.Println("  - Enter a number (1-N) to select an issue")
	fmt.Println("  - Add -p after the number to assign to yourself (e.g., '3 -p')")
	fmt.Println("  - Add -s after the number to change status (e.g., '3 -s')")
	fmt.Println("  - Add -g after the number to create git branch for issue (e.g., '3 -g')")
	fmt.Println("  - Add -su after the number to create subtask + branch (e.g., '3 -su')")
	fmt.Println("  - Enter 0 to exit without selecting")
	fmt.Println("  - Enter h to show this help message")
	fmt.Println()
}
