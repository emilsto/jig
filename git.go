package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func createGitBranch(branchBase, subtaskKey, description string) error {
	description = strings.ToLower(description)
	description = strings.ReplaceAll(description, " ", "-")

	branchName := fmt.Sprintf("%s/%s/%s", branchBase, subtaskKey, description)

	fmt.Printf("\nCreating git branch: %s\n", branchName)
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create git branch: %v", err)
	}

	fmt.Printf("✓ Git branch created and checked out: %s\n", branchName)
	return nil
}
