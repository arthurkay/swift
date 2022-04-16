package main

import (
	"rs/cli"

	"github.com/spf13/cobra"
)

func main() {
	// Define vm domains instance
	var createDomain = cli.CreateDomain()

	rootCmd := &cobra.Command{Use: "rs", Version: "0.0.0"}
	rootCmd.AddCommand(createDomain)
	rootCmd.Execute()
}
