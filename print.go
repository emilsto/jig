package main

import (
	"fmt"
	"strings"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// printSuccess prints a success message with a checkmark
func printSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s✓ %s%s\n", colorGreen, msg, colorReset)
}

// printError prints an error message
func printError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s✗ %s%s\n", colorRed, msg, colorReset)
}

// printWarning prints a warning message
func printWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s⚠ %s%s\n", colorYellow, msg, colorReset)
}

// printInfo prints an informational message
func printInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", colorBlue, msg, colorReset)
}

// printDim prints dimmed text for secondary information
func printDim(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", colorDim, msg, colorReset)
}

// printPrompt prints a prompt for user input
func printPrompt(text string) {
	fmt.Printf("%s%s:%s ", colorYellow, text, colorReset)
}

// printHighlight prints text in cyan (for keys, IDs, etc.)
func printHighlight(text string) string {
	return fmt.Sprintf("%s%s%s", colorCyan, text, colorReset)
}

// printStatus prints text in yellow (for status values)
func printStatus(text string) string {
	return fmt.Sprintf("%s%s%s", colorYellow, text, colorReset)
}

// printBold prints bold text
func printBold(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", colorBold, msg, colorReset)
}

// printTableRow prints a formatted table row
func printTableRow(num int, key, summary, status, assignee string, maxSummaryLen int) {
	// Truncate summary if too long
	if len(summary) > maxSummaryLen {
		summary = summary[:maxSummaryLen-3] + "..."
	}

	fmt.Printf("%s%3d%s │ %s%-20s%s │ %-*s │ %s%-15s%s │ %s\n",
		colorDim, num, colorReset,
		colorCyan, key, colorReset,
		maxSummaryLen, summary,
		colorYellow, status, colorReset,
		assignee)
}

// printTableHeader prints the table header
func printTableHeader(maxSummaryLen int) {
	fmt.Printf("%s%3s%s │ %s%-20s%s │ %-*s │ %s%-15s%s │ %s\n",
		colorBold, "#", colorReset,
		colorBold, "KEY", colorReset,
		maxSummaryLen, "SUMMARY",
		colorBold, "STATUS", colorReset,
		"ASSIGNEE")

	// Print separator line
	fmt.Printf("────┼──────────────────────┼─%s─┼─────────────────┼─%s\n",
		strings.Repeat("─", maxSummaryLen),
		strings.Repeat("─", 30))
}

// printIssueDetails prints detailed information about an issue
func printIssueDetails(issue *DetailedIssue, extractDesc func(interface{}) string) {
	fmt.Println()
	printBold("Issue Details:")
	fmt.Printf("  %sKey:%s          %s\n", colorDim, colorReset, printHighlight(issue.Key))
	fmt.Printf("  %sSummary:%s      %s\n", colorDim, colorReset, issue.Fields.Summary)
	fmt.Printf("  %sType:%s         %s\n", colorDim, colorReset, issue.Fields.IssueType.Name)
	fmt.Printf("  %sStatus:%s       %s\n", colorDim, colorReset, printStatus(issue.Fields.Status.Name))

	assignee := issue.Fields.Assignee.DisplayName
	if assignee == "" {
		assignee = "Unassigned"
	}
	fmt.Printf("  %sAssignee:%s     %s\n", colorDim, colorReset, assignee)

	priority := issue.Fields.Priority.Name
	if priority == "" {
		priority = "None"
	}
	fmt.Printf("  %sPriority:%s     %s\n", colorDim, colorReset, priority)

	if len(issue.Fields.Labels) > 0 {
		fmt.Printf("  %sLabels:%s       %s\n", colorDim, colorReset, strings.Join(issue.Fields.Labels, ", "))
	}

	description := extractDesc(issue.Fields.Description)
	if description != "" {
		fmt.Printf("\n  %sDescription:%s\n", colorDim, colorReset)
		// Wrap description text
		descLines := strings.Split(description, "\n")
		for _, line := range descLines {
			if line != "" {
				fmt.Printf("    %s\n", line)
			}
		}
	}
	fmt.Println()
}
