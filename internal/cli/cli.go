package cli

import (
	"github.com/runparcel/runparcel/internal/generate"

	"github.com/spf13/cobra"
)

// Execute initializes and runs the CLI application.
func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "runparcel",
		Short: "A package manager for Cloud Run deployments",
		Long:  `runparcel is a CLI tool to manage Cloud Run deployments across multiple environments.`,
	}

	// Add subcommands
	rootCmd.AddCommand(generate.Cmd())

	return rootCmd.Execute()
}
