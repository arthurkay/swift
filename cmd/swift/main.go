package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"swift/internal/cloudinit"
	"swift/internal/config"
	"swift/internal/hv"
	"swift/internal/image"
	"swift/internal/ui"

	"github.com/gosimple/slug"
	"github.com/spf13/cobra"
)

var version = "0.4.0"

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
	rootCmd.AddCommand(networkCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// resolveName returns the VM name from either a positional arg or --name flag.
func resolveName(cmd *cobra.Command, args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	name, _ := cmd.Flags().GetString("name")
	return name
}

// resolveNetworkName returns the network name from either a positional arg or --name flag.
func resolveNetworkName(cmd *cobra.Command, args []string) string {
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

// parseNetworkFlag parses the --network flag value "name:MODEL,name2:MODEL2".
func parseNetworkFlag(val string) []hv.NetworkConfig {
	if val == "" {
		return nil
	}
	var nets []hv.NetworkConfig
	for _, part := range strings.Split(val, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name := part
		model := "e1000"
		if idx := strings.Index(part, ":"); idx != -1 {
			name = part[:idx]
			model = part[idx+1:]
		}
		nets = append(nets, hv.NetworkConfig{Name: name, Model: model})
	}
	return nets
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
  swift create myvm -d ~/images/fedora.qcow2 --ssh-key "ssh-rsa AAAA..."
  swift create myvm -d ~/images/ubuntu.qcow2 --network default:e1000,mynet:virtio
  swift create myvm -d ~/images/ubuntu.qcow2 --graphics vnc --websocket --serial
  swift create myvm -f vm.yaml
  swift create -f <(swift show myvm -o yaml)   # recreate from YAML round-trip`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		}
		disk, _ := cmd.Flags().GetString("disk")
		memory, _ := cmd.Flags().GetUint("memory")
		cpu, _ := cmd.Flags().GetUint("cpu")
		storageStr, _ := cmd.Flags().GetString("storage")
		password, _ := cmd.Flags().GetString("password")
		sshKey, _ := cmd.Flags().GetString("ssh-key")
		user, _ := cmd.Flags().GetString("user")
		networkFlag, _ := cmd.Flags().GetString("network")
		filePath, _ := cmd.Flags().GetString("file")
		graphics, _ := cmd.Flags().GetString("graphics")
		websocket, _ := cmd.Flags().GetBool("websocket")
		serial, _ := cmd.Flags().GetBool("serial")

		// Load YAML config if --file provided, flags override YAML values
		var yamlVM *config.VMConfig
		if filePath != "" {
			content, readErr := os.ReadFile(filePath)
			if readErr != nil {
				return fmt.Errorf("read file: %w", readErr)
			}

			// Check if this is XML-mirroring YAML (has "domain" top-level key with map value)
			if hv.IsXMLMirroringYAML(content) {
				// Warn if any flags were provided — they are ignored with XML-mirroring YAML
				ignoredFlags := []string{}
				for _, f := range []string{"disk", "memory", "cpu", "storage", "password", "ssh-key", "user", "network", "graphics", "websocket", "serial"} {
					if cmd.Flags().Changed(f) {
						ignoredFlags = append(ignoredFlags, "--"+f)
					}
				}
				if name != "" || len(args) > 0 {
					ignoredFlags = append(ignoredFlags, "name")
				}
				if len(ignoredFlags) > 0 {
					fmt.Fprintf(os.Stderr, "Warning: flags ignored with XML-mirroring YAML: %s\n", strings.Join(ignoredFlags, ", "))
				}

				xmlStr, convErr := hv.YAMLToXML(string(content))
				if convErr != nil {
					return fmt.Errorf("convert YAML to XML: %w", convErr)
				}

				// Warn about referenced files that don't exist
				hv.WarnMissingFiles(xmlStr)

				if err := hypervisor.DefineDomain(xmlStr); err != nil {
					return fmt.Errorf("define domain: %w", err)
				}
				_ = hypervisor.SetDomainAutostart(name, true)
				fmt.Println("VM defined from YAML definition")
				fmt.Println("Note: disk images and cloud-init ISOs are not created by this path.")
				fmt.Println("Ensure referenced files exist before starting the VM.")
				return nil
			}

			cfg, err := config.LoadVMConfig(filePath)
			if err != nil {
				return err
			}
			yamlVM = cfg
		}

		// Apply YAML values as defaults, then let flags override
		if yamlVM != nil {
			if name == "" {
				name = yamlVM.Name
			}
			if disk == "" {
				disk = yamlVM.Disk
			}
			if memory == 0 && yamlVM.Memory > 0 {
				memory = yamlVM.Memory
			}
			if cpu == 0 && yamlVM.CPU > 0 {
				cpu = yamlVM.CPU
			}
			if storageStr == "" {
				storageStr = yamlVM.Storage
			}
			if password == "" {
				password = yamlVM.Password
			}
			if sshKey == "" {
				sshKey = yamlVM.SSHKey
			}
			if user == "" {
				user = yamlVM.Username
			}
			if graphics == "" {
				graphics = yamlVM.Graphics
			}
			if !websocket {
				websocket = yamlVM.WebSocket
			}
			if !serial {
				serial = yamlVM.Serial
			}
		}

		if name == "" {
			return fmt.Errorf("VM name is required (positional arg or --file with name field)")
		}
		if disk == "" {
			return fmt.Errorf("--disk is required (flag or disk field in YAML)")
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
		if user != "" {
			ud.User = user
		}
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

		// Build network configs from --network flag or YAML
		networks := parseNetworkFlag(networkFlag)
		if len(networks) == 0 && yamlVM != nil && len(yamlVM.Networks) > 0 {
			for _, n := range yamlVM.Networks {
				model := n.Model
				if model == "" {
					model = "e1000"
				}
				networks = append(networks, hv.NetworkConfig{Name: n.Name, Model: model})
			}
		}

		resources := hv.DomainResources{
			Name:         name,
			Memory:       memory,
			Unit:         "MiB",
			CpuCount:     cpu,
			BootOS:       diskPath,
			CDRom:        isoPath,
			MAC:          mac,
			Networks:     networks,
			GraphicsType: graphics,
			WebSocket:    websocket,
			Serial:       serial,
		}

		domainXML := resources.BuildDomainXML()
		xmlBytes, err := xml.MarshalIndent(domainXML, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal domain XML: %w", err)
		}

		if err := hypervisor.DefineDomain(string(xmlBytes)); err != nil {
			return fmt.Errorf("define domain: %w", err)
		}
		_ = hypervisor.SetDomainAutostart(name, true)

		fmt.Printf("  Specs: %d MiB RAM, %d CPU(s)\n", memory, cpu)
		if len(networks) > 0 {
			for _, n := range networks {
				fmt.Printf("  NIC:   %s (%s)\n", n.Name, n.Model)
			}
		}
		if graphics != "none" {
			gfxLabel := strings.ToUpper(graphics)
			if graphics == "" {
				gfxLabel = "SPICE"
			}
			ws := ""
			if websocket {
				ws = " + WebSocket"
			}
			fmt.Printf("  GFX:   %s%s\n", gfxLabel, ws)
		}
		if serial {
			fmt.Printf("  Serial: PTY console enabled\n")
		}
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
	createCmd.Flags().StringP("user", "u", "", "Cloud-init username (default: swift)")
	createCmd.Flags().StringP("network", "n", "", "Networks as name:MODEL,... (e.g. default:e1000,mynet:virtio)")
	createCmd.Flags().StringP("file", "f", "", "YAML config file (simplified or XML-mirroring from 'swift show')")
	createCmd.Flags().StringP("graphics", "g", "", "Graphics type: spice (default), vnc, none")
	createCmd.Flags().Bool("websocket", false, "Enable WebSocket listener for browser-based console access")
	createCmd.Flags().Bool("serial", false, "Add PTY serial console for headless/CLI access")
	showCmd.Flags().StringP("format", "o", "yaml", "Output format: yaml (default), xml")
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

// --- stop ---

var stopCmd = &cobra.Command{
	Use:     "stop [name]",
	Aliases: []string{"poweroff"},
	Short:   "Stop a virtual machine",
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
			fmt.Printf("VM '%s' shutting down (ACPI signal sent)...\n", name)

			// Poll for up to 30 seconds to see if the VM shuts down gracefully
			fmt.Print("Waiting for VM to shut down")
			for i := 0; i < 30; i++ {
				time.Sleep(1 * time.Second)
				state, err := hypervisor.DomainState(name)
				if err != nil || state == "Shutoff" || state == "Shutdown" || state == "Crashed" {
					fmt.Println()
					if state == "Shutoff" {
						fmt.Printf("VM '%s' shut down successfully\n", name)
					} else {
						fmt.Printf("VM '%s' is in state: %s\n", name, state)
					}
					return nil
				}
				fmt.Print(".")
			}
			fmt.Println()

			// VM didn't shut down in time, force stop it
			fmt.Printf("VM '%s' did not shut down gracefully. Force stopping...\n", name)
			if err := hypervisor.StopDomain(name); err != nil {
				return fmt.Errorf("force stop domain after timeout: %w", err)
			}
			fmt.Printf("VM '%s' force stopped\n", name)
		}
		return nil
	},
}

func init() {
	stopCmd.Flags().BoolP("force", "f", false, "Force stop (pull the plug) instead of graceful shutdown")
}

// --- delete ---

var deleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a virtual machine",
	Long: `Delete (undefine) a virtual machine from libvirt and remove its files.

Prompts for confirmation before removing the VM definition.
Removes the VM's disk images, cloud-init ISO, and all files from /var/swift/<name>/.`,
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

		// Remove the VM project directory (disk images, ISOs, cloud-init files)
		projectDir, pathErr := config.VMPath(slug.Make(name))
		if pathErr == nil {
			os.RemoveAll(projectDir)
		}

		fmt.Printf("VM '%s' deleted successfully\n", name)
		return nil
	},
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

// --- show ---

var showCmd = &cobra.Command{
	Use:     "show [name]",
	Aliases: []string{"xml"},
	Short:   "Show the definition of a virtual machine",
	Long: `Display the definition of a virtual machine in YAML (default) or XML format.

The YAML output mirrors the libvirt XML structure exactly.
Edit and recreate with: swift create -f <file>`,
	Example: `  swift show myvm              # YAML (default)
  swift show myvm -o xml      # raw libvirt XML
  swift show myvm -o yaml     # explicit YAML
  swift show                  # interactive selection`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveName(cmd, args)
		format, _ := cmd.Flags().GetString("format")

		if name == "" {
			domains, err := vmList()
			if err != nil {
				return fmt.Errorf("list domains: %w", err)
			}
			if len(domains) == 0 {
				fmt.Println("No VMs defined.")
				return nil
			}
			name, err = ui.SelectVM("Select VM to view", vmItems(domains))
			if err != nil {
				return fmt.Errorf("selection cancelled")
			}
		}

		domainXML, err := hypervisor.DomainXML(name)
		if err != nil {
			return fmt.Errorf("get domain XML: %w", err)
		}

		switch format {
		case "xml":
			fmt.Println(domainXML)
		case "yaml":
			yamlStr, err := hv.XMLToYAML(domainXML)
			if err != nil {
				return fmt.Errorf("convert to YAML: %w", err)
			}
			fmt.Print(yamlStr)
		default:
			return fmt.Errorf("unknown format %q: supported values are yaml, xml", format)
		}
		return nil
	},
}

// ====================== network ======================

var networkCmd = &cobra.Command{
	Use:     "network",
	Aliases: []string{"net"},
	Short:   "Manage virtual networks",
	Long:    "Create, list, start, stop, and manage libvirt virtual networks.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// --- network create ---

var networkCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new virtual network",
	Long: `Create and define a new libvirt virtual network.

Use --file to load settings from a YAML config, or specify flags directly.
Supported forward modes: nat, route, bridge, macvtap, passthrough, isolated.`,
	Example: `  swift network create mynet --cidr 192.168.100.0/24
  swift network create mynet --mode isolated --cidr 10.10.10.0/24
  swift network create mynet --file network.yaml
  swift network create mynet --mode bridge --bridge br0
  swift network create -f <(swift network show mynet -o yaml)  # recreate from YAML`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		}
		cidr, _ := cmd.Flags().GetString("cidr")
		mode, _ := cmd.Flags().GetString("mode")
		bridge, _ := cmd.Flags().GetString("bridge")
		device, _ := cmd.Flags().GetString("device")
		dhcp, _ := cmd.Flags().GetBool("dhcp")
		dhcpStart, _ := cmd.Flags().GetString("dhcp-start")
		dhcpEnd, _ := cmd.Flags().GetString("dhcp-end")
		filePath, _ := cmd.Flags().GetString("file")

		// Load YAML config if --file provided, flags override YAML values
		if filePath != "" {
			content, readErr := os.ReadFile(filePath)
			if readErr != nil {
				return fmt.Errorf("read file: %w", readErr)
			}

			// Check if this is XML-mirroring YAML (has "network" top-level key with map value)
			if hv.IsXMLMirroringYAML(content) {
				// Warn if any flags were provided — they are ignored with XML-mirroring YAML
				ignoredFlags := []string{}
				for _, f := range []string{"cidr", "mode", "bridge", "device", "dhcp", "dhcp-start", "dhcp-end"} {
					if cmd.Flags().Changed(f) {
						ignoredFlags = append(ignoredFlags, "--"+f)
					}
				}
				if name != "" || len(args) > 0 {
					ignoredFlags = append(ignoredFlags, "name")
				}
				if len(ignoredFlags) > 0 {
					fmt.Fprintf(os.Stderr, "Warning: flags ignored with XML-mirroring YAML: %s\n", strings.Join(ignoredFlags, ", "))
				}

				xmlStr, convErr := hv.YAMLToXML(string(content))
				if convErr != nil {
					return fmt.Errorf("convert YAML to XML: %w", convErr)
				}
				if err := hypervisor.DefineNetwork(xmlStr); err != nil {
					return fmt.Errorf("define network: %w", err)
				}
				_ = hypervisor.SetNetworkAutostart(args[0], true)
				fmt.Println("Network defined from YAML definition")
				return nil
			}

			cfg, err := config.LoadNetworkConfig(filePath)
			if err != nil {
				return err
			}
			if name == "" {
				name = cfg.Name
			}
			if cidr == "" {
				cidr = cfg.CIDR
			}
			if mode == "" {
				mode = cfg.Mode
			}
			if bridge == "" {
				bridge = cfg.Bridge
			}
			if device == "" {
				device = cfg.Device
			}
			if !dhcp {
				dhcp = cfg.DHCP
			}
			if dhcpStart == "" {
				dhcpStart = cfg.DHCPStart
			}
			if dhcpEnd == "" {
				dhcpEnd = cfg.DHCPEnd
			}
		}

		if name == "" {
			return fmt.Errorf("network name is required")
		}
		if cidr == "" && mode != "bridge" {
			return fmt.Errorf("--cidr is required for non-bridge networks")
		}

		res := hv.NetworkResources{
			Name:       name,
			CIDR:       cidr,
			Mode:       mode,
			BridgeName: bridge,
			Device:     device,
			DHCP:       dhcp,
			DHCPStart:  dhcpStart,
			DHCPEnd:    dhcpEnd,
		}
		res.NetworkResourcesDefaults()

		xml, err := res.BuildNetworkXML()
		if err != nil {
			return fmt.Errorf("build network XML: %w", err)
		}

		if err := hypervisor.DefineNetwork(xml); err != nil {
			return fmt.Errorf("define network: %w", err)
		}
		_ = hypervisor.SetNetworkAutostart(name, true)

		fmt.Printf("Network '%s' created successfully\n", name)
		return nil
	},
}

func init() {
	networkCreateCmd.Flags().String("cidr", "", "CIDR for the network (e.g. 192.168.122.0/24)")
	networkCreateCmd.Flags().StringP("mode", "m", "", "Forward mode: nat, route, bridge, macvtap, passthrough, isolated (default: nat)")
	networkCreateCmd.Flags().String("bridge", "", "Bridge device name")
	networkCreateCmd.Flags().StringP("device", "d", "", "Host device for forwarding")
	networkCreateCmd.Flags().Bool("dhcp", false, "Enable DHCP on the network")
	networkCreateCmd.Flags().String("dhcp-start", "", "DHCP range start IP")
	networkCreateCmd.Flags().String("dhcp-end", "", "DHCP range end IP")
	networkCreateCmd.Flags().StringP("file", "f", "", "YAML config file (simplified or XML-mirroring from 'swift network show')")
	networkShowCmd.Flags().StringP("format", "o", "yaml", "Output format: yaml (default), xml")
}

// --- network list ---

var networkListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"net", "nets"},
	Short:   "List all virtual networks",
	Long:    "List all defined virtual networks with their state and bridge.",
	Example: `  swift network list
  swift net list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		networks, err := hypervisor.ListNetworks()
		if err != nil {
			return fmt.Errorf("list networks: %w", err)
		}

		if len(networks) == 0 {
			fmt.Println("No networks defined.")
			return nil
		}

		fmt.Printf("%-20s %-38s %-12s %-20s %s\n", "NAME", "UUID", "STATE", "BRIDGE", "MODE")
		fmt.Printf("%-20s %-38s %-12s %-20s %s\n", "----", "----", "-----", "------", "----")
		for _, n := range networks {
			fmt.Printf("%-20s %-38s %-12s %-20s %s\n", n.Name, n.UUID, n.State, n.BridgeName, n.ForwardMode)
		}
		return nil
	},
}

// --- network start ---

var networkStartCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a virtual network",
	Long:  "Activate a defined virtual network.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveNetworkName(cmd, args)
		if name == "" {
			return fmt.Errorf("network name is required")
		}
		if err := hypervisor.StartNetwork(name); err != nil {
			return fmt.Errorf("start network: %w", err)
		}
		fmt.Printf("Network '%s' started\n", name)
		return nil
	},
}

// --- network stop ---

var networkStopCmd = &cobra.Command{
	Use:     "stop [name]",
	Aliases: []string{"down"},
	Short:   "Stop a virtual network",
	Long:    "Deactivate a running virtual network.",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveNetworkName(cmd, args)
		if name == "" {
			return fmt.Errorf("network name is required")
		}
		if err := hypervisor.StopNetwork(name); err != nil {
			return fmt.Errorf("stop network: %w", err)
		}
		fmt.Printf("Network '%s' stopped\n", name)
		return nil
	},
}

// --- network delete ---

var networkDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a virtual network",
	Long:  "Undefine a virtual network from libvirt.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveNetworkName(cmd, args)
		if name == "" {
			return fmt.Errorf("network name is required")
		}
		if err := hypervisor.UndefineNetwork(name); err != nil {
			return fmt.Errorf("undefine network: %w", err)
		}
		fmt.Printf("Network '%s' deleted successfully\n", name)
		return nil
	},
}

// --- network status ---

var networkStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show the state of a virtual network",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveNetworkName(cmd, args)
		if name == "" {
			return fmt.Errorf("network name is required")
		}
		state, err := hypervisor.NetworkState(name)
		if err != nil {
			return fmt.Errorf("get network state: %w", err)
		}
		fmt.Printf("Network '%s': %s\n", name, state)
		return nil
	},
}

// --- network show ---

var networkShowCmd = &cobra.Command{
	Use:     "show [name]",
	Aliases: []string{"xml"},
	Short:   "Show the definition of a virtual network",
	Long: `Display the definition of a virtual network in YAML (default) or XML format.

The YAML output mirrors the libvirt XML structure exactly.
Edit and recreate with: swift network create -f <file>`,
	Example: `  swift network show mynet      # YAML (default)
  swift network show mynet -o xml  # raw libvirt XML`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := resolveNetworkName(cmd, args)
		format, _ := cmd.Flags().GetString("format")
		if name == "" {
			return fmt.Errorf("network name is required")
		}
		xmlStr, err := hypervisor.NetworkXML(name)
		if err != nil {
			return fmt.Errorf("get network XML: %w", err)
		}
		switch format {
		case "xml":
			fmt.Println(xmlStr)
		case "yaml":
			yamlStr, err := hv.XMLToYAML(xmlStr)
			if err != nil {
				return fmt.Errorf("convert to YAML: %w", err)
			}
			fmt.Print(yamlStr)
		default:
			return fmt.Errorf("unknown format %q: supported values are yaml, xml", format)
		}
		return nil
	},
}

func init() {
	networkCmd.AddCommand(networkCreateCmd)
	networkCmd.AddCommand(networkListCmd)
	networkCmd.AddCommand(networkStartCmd)
	networkCmd.AddCommand(networkStopCmd)
	networkCmd.AddCommand(networkDeleteCmd)
	networkCmd.AddCommand(networkStatusCmd)
	networkCmd.AddCommand(networkShowCmd)
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
