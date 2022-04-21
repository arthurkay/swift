package main

import (
	"swift/cli"

	"github.com/spf13/cobra"
)

func main() {
	// Define vm domains instance
	var createDomain = cli.CreateDomain()
	// Show everything that has to do with domains
	var domains = cli.ListDomains()
	// undefine the domain
	var undefine = cli.UndefineDomain()
	// run a VM instance
	var startVM = cli.StartDomain()
	// Shutdown a VM instance
	var shutdown = cli.ShutdownDomain()
	// Get domain status
	var status = cli.DomainState()

	rootCmd := &cobra.Command{Use: "swift", Version: "0.0.0"}
	rootCmd.AddCommand(createDomain)
	rootCmd.AddCommand(domains)
	rootCmd.AddCommand(undefine)
	rootCmd.AddCommand(startVM)
	rootCmd.AddCommand(shutdown)
	rootCmd.AddCommand(status)
	rootCmd.Execute()
}
