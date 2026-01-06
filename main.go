package main

import (
	"os"
	commands "ratelimit-challenge/cmd"
)

func main() {
	rootCmd := commands.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
