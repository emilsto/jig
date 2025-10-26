package main

import (
	"fmt"
	"strings"
	"github.com/emilsto/jig/jira"
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

func printSuccess(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s✓ %s%s\n", colorGreen, msg, colorReset)
}

func printError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s✗ %s%s\n", colorRed, msg, colorReset)
}

func printWarning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s⚠ %s%s\n", colorYellow, msg, colorReset)
}

func printInfo(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", colorBlue, msg, colorReset)
}

func printDim(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", colorDim, msg, colorReset)
}

func printPrompt(text string) {
	fmt.Printf("%s%s:%s ", colorYellow, text, colorReset)
}

func printHighlight(text string) string {
	return fmt.Sprintf("%s%s%s", colorCyan, text, colorReset)
}

func printStatus(text string) string {
	return fmt.Sprintf("%s%s%s", colorYellow, text, colorReset)
}

func printBold(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", colorBold, msg, colorReset)
}

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

func printIssueDetails(issue *jira.DetailedIssue, extractDesc func(any) string) {
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
