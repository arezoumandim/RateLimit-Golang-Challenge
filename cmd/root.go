package commands

import (
	"demo-saturday/cmd/commands/server"

	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "rate-limiter",
		Short: "High-performance distributed rate limiter",
		Long:  "A high-performance distributed rate limiter for API gateways",
	}

	// Add subcommands
	rootCmd.AddCommand(server.NewCommand())

	return rootCmd
}

