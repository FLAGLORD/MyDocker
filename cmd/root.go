package cmd

import "github.com/spf13/cobra"

// Usage of cobra reference blog: https://www.qikqiak.com/post/create-cli-app-with-cobra/
var rootCmd = &cobra.Command{
	Use: "mydocker",
	Short: "mydocker is a simple container runtime implementation.",
}

// Execute exexcuted the root command
func Execute() error{
	return rootCmd.Execute()
}

