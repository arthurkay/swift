package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"

	"swift/internal/cloudinit"
	"swift/internal/config"
	"swift/internal/hv"
	"swift/internal/image"
	"swift/internal/ui"

	"github.com/spf13/cobra"
)

var version = "0.3.0"

var hypervisor hv.Hypervisor

var rootCmd = &cobra.Command{
	Use:   "swift",
	Short: "A swift way of provisioning infrastructure in seconds",
	Long: `Swift provisions and manages local virtual machines using libvirt/KVM.

It creates QCOW2 disk images from cloud images, generates cloud-init
seed ISOs for automated OS configuration, and defines VMs in libvirt.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip hypervisor connection for commands that don't need it
		switch cmd.Name() {
		case "version", "swift":
			return
		}
		var err error
		hypervisor, err = hv.Connect()
		if err != nil {
			if strings.Contains(err.Error(), "Permission denied") {
				fmt.Fprintln(os.Stderr, "Error: permission denied connecting to libvirt.")
				fmt.Fprintln(os.Stderr)
				fmt.Fprintln(os.Stderr, "Add your user to the libvirt group:")
				fmt.Fprintln(os.Stderr, "  sudo usermod -aG libvirt $USER")
				fmt.Fprintln(os.Stderr)
				fmt.Fprintln(os.Stderr, "Then log out and back in, or run: newgrp libvirt")
			} else {
				fmt.Fprintf(os.Stderr, "Error: connect to hypervisor: %v\n", err)
			}
			os.Exit(1)
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if h, ok := hypervisor.(*hv.LibvirtHypervisor); ok && h != nil {
			h.Disconnect()
		}
	},
	SilenceUsage: true,
}

func main() {
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// resolveName returns the VM name from either a positional arg or --name flag.
// If neither is provided, it returns empty (caller should trigger interactive selection).
func resolveName(cmd *cobra.Command, args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	name, _ := cmd.Flags().GetString("name")
	return name
}

// vmNames fetches domain names from the hypervisor.
func vmNames() ([]string, error) {
	return hypervisor.DomainNames()
}

// vmList fetches all domains as DomainInfo for interactive selection.
func vmList() ([]hv.DomainInfo, error) {
	return hypervisor.ListDomains()
}

// vmItems converts DomainInfo slices to UI items for interactive prompts.
func vmItems(domains []hv.DomainInfo) []ui.VMItem {
	items := make([]ui.VMItem, len(domains))
	for i, d := range domains {
		items[i] = ui.VMItem{Name: d.Name, State: d.State}
	}
	return items
}

// --- version ---

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the swift version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("swift %s\n", version)
	},
}

// --- create ---

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new virtual machine",
	Long: `Create and provision a new virtual machine.

Creates a QCOW2 disk image from a backing cloud image, generates
a cloud-init seed ISO for automated OS configuration, and defines
the VM in libvirt with the specified resources.`,
	Example: `  swift create myvm --disk ~/images/ubuntu-22.04.qcow2
  swift create myvm -d ~/images/ubuntu.qcow2 -m 2048 -c 4 -s 20G
  swift create myvm -d ~/images/fedora.qcow2 --ssh-key "ssh-rsa AAAA..."`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		disk, _ := cmd.Flags().GetString("disk")
		memory, _ := cmd.Flags().GetUint("memory")
		cpu, _ := cmd.Flags().GetUint("cpu")
		storageStr, _ := cmd.Flags().GetString("storage")
		password, _ := cmd.Flags().GetString("password")
		sshKey, _ := cmd.Flags().GetString("ssh-key")

		if disk == "" {
			return fmt.Errorf("--disk is required")
		}

		if memory == 0 {
			memory = 512
		}
		if cpu == 0 {
			cpu = 1
		}

		storage, err := parseSize(storageStr)
		if err != nil {
			return fmt.Errorf("invalid storage size %q: %w", storageStr, err)
		}
		if storage == 0 {
			storage = 8 * image.GiB
		}

		if password == "" {
			password = config.DefaultPassword
		}

		fmt.Printf("Creating VM: %s\n", name)

		// Create cloud-init user-data and meta-data
		ud := cloudinit.NewUserData(name, password, sshKey)
		if err := ud.CreateProjectFiles(); err != nil {
			return fmt.Errorf("create project files: %w", err)
		}
		if err := ud.CloudConfig(); err != nil {
			return fmt.Errorf("generate cloud config: %w", err)
		}

		projectDir, err := config.VMPath(ud.Slug)
		if err != nil {
			return fmt.Errorf("get project path: %w", err)
		}

		// Create QCOW2 disk image
		diskPath := projectDir + "/" + ud.Slug + ".qcow2"
		img := image.NewImage(diskPath, image.FormatQCOW2, storage)
		if err := img.SetBackingFile(disk); err != nil {
			return fmt.Errorf("set backing file: %w", err)
		}
		if err := img.Create(); err != nil {
			return fmt.Errorf("create disk image: %w", err)
		}
		fmt.Printf("  Disk:  %s (%d GiB)\n", diskPath, storage/image.GiB)

		// Create cloud-init seed ISO
		isoPath := projectDir + "/cidata.iso"
		seed := cloudinit.NewSeed(isoPath, projectDir+"/user-data", projectDir+"/meta-data")
		if err := seed.Create(); err != nil {
			return fmt.Errorf("create seed ISO: %w", err)
		}
		fmt.Printf("  ISO:   %s\n", isoPath)

		// Generate MAC and build domain XML
		mac, err := config.GenerateMAC()
		if err != nil {
			return fmt.Errorf("generate MAC: %w", err)
		}

		resources := hv.DomainResources{
			Name:     name,
			Memory:   memory,
			Unit:     "MiB",
			CpuCount: cpu,
			BootOS:   diskPath,
			CDRom:    isoPath,
			MAC:      mac,
		}

		domainXML := resources.BuildDomainXML()
		xmlBytes, err := xml.MarshalIndent(domainXML, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal domain XML: %w", err)
		}

		if err := hypervisor.DefineDomain(string(xmlBytes)); err != nil {
			return fmt.Errorf("define domain: %w", err)
		}

		fmt.Printf("  Specs: %d MiB RAM, %d CPU(s)\n", memory, cpu)
		fmt.Printf("\nVM '%s' created successfully\n", name)
		return nil
	},
}

func init() {
	createCmd.Flags().StringP("disk", "d", "", "Path to cloud image backing file (required)")
	createCmd.Flags().UintP("memory", "m", 0, "Memory in MiB (default: 512)")
	createCmd.Flags().UintP("cpu", "c", 0, "Number of CPUs (default: 1)")
	createCmd.Flags().StringP("storage", "s", "", "Disk size, e.g. 10G, 512M, 1T (default: 8G)")
	createCmd.Flags().StringP("password", "p", "", "VM password (default: swift1234)")
	createCmd.Flags().String("ssh-key", "", "SSH public key for the default user")
}

// --- list ---

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"vm", "vms"},
	Short:   "List all virtual machines",
	Long: `List all defined virtual machines with their state.

Shows the name, UUID, ID, and current state (Running, Shutoff, etc.)
of every VM managed by swift.`,
	Example: `  swift list
  swift vm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		domains, err := vmList()
		if err != nil {
			return fmt.Errorf("list domains: %w", err)
		}

		if len(domains) == 0 {
			fmt.Println("No VMs defined.")
			fmt.Println("\nCreate one with: swift create <name> --disk <image>")
			return nil
		}

		fmt.Printf("%-20s %-38s %-12s %s\n", "NAME", "UUID", "ID", "STATE")
		fmt.Printf("%-20s %-38s %-12s %s\n", "----", "----", "--", "-----")
		for _, d := range domains {
			fmt.Printf("%-20s %-38s %-12d %s\n", d.Name, d.UUID, d.ID, d.State)
		}
		return nil
	},
}

// --- start ---

var startCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a virtual machine",
	Long: `Start a defined virtual machine.

If no name is provided, an interactive prompt lists available VMs
with their current state for selection.`,
	Example: `  swift start myvm
  swift start          # interactive selection`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveName(cmd, args)

		if name == "" {
			domains, err := vmList()
			if err != nil {
				return fmt.Errorf("list domains: %w", err)
			}
			if len(domains) == 0 {
				fmt.Println("No VMs defined.")
				return nil
			}
			name, err = ui.SelectVM("Select VM to start", vmItems(domains))
			if err != nil {
				return fmt.Errorf("selection cancelled")
			}
		}

		if err := hypervisor.StartDomain(name); err != nil {
			return fmt.Errorf("start domain: %w", err)
		}

		fmt.Printf("VM '%s' started\n", name)
		return nil
	},
}

func init() {
	startCmd.Flags().StringP("name", "n", "", "VM name (or use positional arg)")
}

// --- stop ---

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop a virtual machine",
	Long: `Stop a running virtual machine.

By default, sends an ACPI shutdown signal for a graceful stop.
Use --force to immediately pull the plug (equivalent to pulling the power cord).`,
	Example: `  swift stop myvm                # graceful ACPI shutdown
  swift stop myvm --force        # immediate force stop
  swift poweroff myvm            # same as 'stop' (alias)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveName(cmd, args)
		force, _ := cmd.Flags().GetBool("force")

		if name == "" {
			domains, err := vmList()
			if err != nil {
				return fmt.Errorf("list domains: %w", err)
			}
			if len(domains) == 0 {
				fmt.Println("No VMs defined.")
				return nil
			}
			name, err = ui.SelectVM("Select VM to stop", vmItems(domains))
			if err != nil {
				return fmt.Errorf("selection cancelled")
			}
		}

		if force {
			if err := hypervisor.StopDomain(name); err != nil {
				return fmt.Errorf("force stop domain: %w", err)
			}
			fmt.Printf("VM '%s' force stopped\n", name)
		} else {
			if err := hypervisor.ShutdownDomain(name); err != nil {
				return fmt.Errorf("shutdown domain: %w", err)
			}
			fmt.Printf("VM '%s' shutting down\n", name)
		}
		return nil
	},
}

func init() {
	stopCmd.Flags().StringP("name", "n", "", "VM name (or use positional arg)")
	stopCmd.Flags().BoolP("force", "f", false, "Force stop (pull the plug) instead of graceful shutdown")
}

// --- delete ---

var deleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a virtual machine",
	Long: `Delete (undefine) a virtual machine from libvirt.

Prompts for confirmation before removing the VM definition.
This does not delete the VM's disk images from disk.`,
	Example: `  swift delete myvm
  swift delete          # interactive selection`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveName(cmd, args)

		if name == "" {
			domains, err := vmList()
			if err != nil {
				return fmt.Errorf("list domains: %w", err)
			}
			if len(domains) == 0 {
				fmt.Println("No VMs to delete.")
				return nil
			}
			name, err = ui.SelectVM("Select VM to delete", vmItems(domains))
			if err != nil {
				return fmt.Errorf("selection cancelled")
			}
		}

		fmt.Printf("Are you sure you want to delete VM '%s'? ", name)
		confirm, err := ui.ConfirmOperation()
		if err != nil {
			return fmt.Errorf("confirmation cancelled")
		}
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Delete cancelled.")
			return nil
		}

		if err := hypervisor.UndefineDomain(name); err != nil {
			return fmt.Errorf("undefine domain: %w", err)
		}

		fmt.Printf("VM '%s' deleted successfully\n", name)
		return nil
	},
}

func init() {
	deleteCmd.Flags().StringP("name", "n", "", "VM name (or use positional arg)")
}

// --- status ---

var statusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show the state of a virtual machine",
	Long: `Display the current state of a virtual machine.

States include: Running, Shutoff, Paused, Shutdown, Crashed, etc.`,
	Example: `  swift status myvm
  swift status          # interactive selection`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveName(cmd, args)

		if name == "" {
			domains, err := vmList()
			if err != nil {
				return fmt.Errorf("list domains: %w", err)
			}
			if len(domains) == 0 {
				fmt.Println("No VMs defined.")
				return nil
			}
			name, err = ui.SelectVM("Select VM to check status", vmItems(domains))
			if err != nil {
				return fmt.Errorf("selection cancelled")
			}
		}

		state, err := hypervisor.DomainState(name)
		if err != nil {
			return fmt.Errorf("get domain state: %w", err)
		}

		fmt.Printf("VM '%s': %s\n", name, state)
		return nil
	},
}

func init() {
	statusCmd.Flags().StringP("name", "n", "", "VM name (or use positional arg)")
}

// --- show ---

var showCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show the libvirt XML definition of a virtual machine",
	Long: `Display the full libvirt XML descriptor for a virtual machine.

This is useful for debugging or for manually editing VM configurations.`,
	Example: `  swift show myvm
  swift xml myvm          # same as 'show' (alias)
  swift show              # interactive selection`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveName(cmd, args)

		if name == "" {
			domains, err := vmList()
			if err != nil {
				return fmt.Errorf("list domains: %w", err)
			}
			if len(domains) == 0 {
				fmt.Println("No VMs defined.")
				return nil
			}
			name, err = ui.SelectVM("Select VM to view XML", vmItems(domains))
			if err != nil {
				return fmt.Errorf("selection cancelled")
			}
		}

		domainXML, err := hypervisor.DomainXML(name)
		if err != nil {
			return fmt.Errorf("get domain XML: %w", err)
		}

		fmt.Println(domainXML)
		return nil
	},
}

func init() {
	showCmd.Flags().StringP("name", "n", "", "VM name (or use positional arg)")
}

// --- parseSize ---

// parseSize converts a human-readable size string to bytes.
// Supported formats: 10G, 10GB, 512M, 512MB, 1T, 1TB, or raw number (bytes).
func parseSize(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}

	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)

	multipliers := map[string]uint64{
		"TB": 1099511627776,
		"T":  1099511627776,
		"GB": 1073741824,
		"G":  1073741824,
		"MB": 1048576,
		"M":  1048576,
		"KB": 1024,
		"K":  1024,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(upper, suffix) {
			numStr := strings.TrimSpace(s[:len(s)-len(suffix)])
			num, err := strconv.ParseUint(numStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number: %s", numStr)
			}
			return num * mult, nil
		}
	}

	// Plain number = bytes
	num, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %s", s)
	}
	return num, nil
}
