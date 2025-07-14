package utils

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GetResolvedTag determines the final image tag based on user input or auto-generation.
// If userProvidedTag is not empty, it's used directly.
// Otherwise, a tag in YYYY.MM.DD.commit_hash format is generated.
func GetResolvedTag(userProvidedTag string) (string, error) {
	resolvedTag := userProvidedTag
	if resolvedTag == "" {
		// No tag is provided via flag, generate one
		commitHash, err := getGitCommitHash()
		if err != nil {
			return "", fmt.Errorf("failed to get git commit hash for auto-tagging: %v", err)
		}
		dateTag := getFormattedDate()
		resolvedTag = fmt.Sprintf("%s.%s", dateTag, commitHash[:7]) // Use short commit hash
		fmt.Printf("Info: No tag provided via --tag flag. Auto-generated tag: %s\n", resolvedTag)
	} else {
		fmt.Printf("Info: Using tag provided via --tag flag: %s\n", resolvedTag)
	}
	return resolvedTag, nil
}

// getGitCommitHash executes 'git rev-parse HEAD' to get the current commit hash.
func getGitCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			return "", fmt.Errorf("git command failed with: %s (stderr: %s)", exitErr.Error(), string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to execute git command: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// getFormattedDate returns the current date in YYYY.MM.DD format.
func getFormattedDate() string {
	return time.Now().Format("2006.01.02")
}
