package main

import (
	"context"
	"fmt"

	"swift/internal/cluster"
	"swift/internal/ui"

	agentpb "swift/internal/proto"

	"github.com/spf13/cobra"
)

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage VMs across multiple hosts",
	Long: `Manage virtual machines across a cluster of hosts.

Requires a cluster config at ~/.swift/config.yaml.
Single-host commands (swift create, swift list, etc.) continue to work
against the local host without an agent.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// --- cluster list ---

var clusterListCmd = &cobra.Command{
	Use:   "hosts",
	Short: "List all cluster hosts",
	Long:  "List all configured cluster hosts with their health status and resources.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cluster.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		orch := cluster.NewOrchestrator(cfg)
		defer orch.Close()

		results := orch.ListHosts()

		fmt.Printf("%-20s %-30s %-8s %-10s\n", "NAME", "ADDRESS", "HEALTHY", "VMs")
		fmt.Printf("%-20s %-30s %-8s %-10s\n", "----", "-------", "-------", "---")
		for _, r := range results {
			if r.Err != nil {
				fmt.Printf("%-20s %-30s %-8s %v\n", r.Host, "error", "?", r.Err)
				continue
			}
			health := "no"
			if r.Info.Healthy {
				health = "yes"
			}
			fmt.Printf("%-20s %-30s %-8s %d\n", r.Info.Name, r.Info.Address, health, len(r.Info.Vms))
		}
		return nil
	},
}

// --- cluster vm ---

var clusterVMCmd = &cobra.Command{
	Use:   "vm",
	Short: "List VMs across all hosts",
	Long:  "List all virtual machines across all cluster hosts.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cluster.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		orch := cluster.NewOrchestrator(cfg)
		defer orch.Close()

		results := orch.ListAllVMs()

		fmt.Printf("%-20s %-12s %-38s %-15s %s\n", "NAME", "HOST", "UUID", "ID", "STATE")
		fmt.Printf("%-20s %-12s %-38s %-15s %s\n", "----", "----", "----", "--", "-----")
		for _, r := range results {
			if r.Err != nil {
				fmt.Printf("%-20s %-12s error: %v\n", r.Host, "", r.Err)
				continue
			}
			for _, vm := range r.VMs {
				fmt.Printf("%-20s %-12s %-38s %-15d %s\n", vm.Name, r.Host, vm.Uuid, vm.Id, vm.State)
			}
		}
		return nil
	},
}

// --- cluster define ---

var clusterDefineCmd = &cobra.Command{
	Use:   "define [name]",
	Short: "Define a VM on a cluster host (auto-places or targets --host)",
	Long: `Define a virtual machine on a cluster host.

If --host is specified, the VM is placed on that host.
Otherwise, the placement engine selects the best host based on
available resources and placement strategy.`,
	Example: `  swift cluster define web-1 --disk https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img
  swift cluster define web-1 --host host-a --disk /path/to/image.qcow2
  swift cluster define web-1 --disk image.qcow2 --memory 2048 --cpu 2`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		}
		diskURL, _ := cmd.Flags().GetString("disk")
		hostName, _ := cmd.Flags().GetString("host")
		memory, _ := cmd.Flags().GetUint("memory")
		cpu, _ := cmd.Flags().GetUint("cpu")
		storageStr, _ := cmd.Flags().GetString("storage")

		if name == "" {
			return fmt.Errorf("VM name is required")
		}
		if diskURL == "" {
			return fmt.Errorf("--disk is required (URL or local path)")
		}

		cfg, err := cluster.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		orch := cluster.NewOrchestrator(cfg)
		defer orch.Close()

		var targetHost *cluster.HostConfig
		if hostName != "" {
			targetHost, err = cfg.GetHostByName(hostName)
			if err != nil {
				return fmt.Errorf("host: %w", err)
			}
		} else {
			// Auto-place
			storage, _ := parseSize(storageStr)
			if storage == 0 {
				storage = 8 * 1073741824 // 8 GiB
			}
			result, placeErr := orch.Place(&cluster.PlacementRequest{
				Memory:   uint32(memory),
				CpuCount: uint32(cpu),
				DiskSize: storage,
			})
			if placeErr != nil {
				return fmt.Errorf("placement: %w", placeErr)
			}
			fmt.Printf("Placed on host: %s (%s)\n", result.Host, result.Reason)
			targetHost, _ = cfg.GetHostByName(result.Host)
		}

		// Connect to target agent and define
		client, err := orch.GetPool().Get(targetHost.HostAddress())
		if err != nil {
			return fmt.Errorf("connect to agent: %w", err)
		}

		resp, err := (*client).DefineVM(context.Background(), &agentpb.DefineVMRequest{
			Name:            name,
			Memory:          uint32(memory),
			MemoryUnit:      "MiB",
			CpuCount:        uint32(cpu),
			BackingImageUrl: diskURL,
			DiskSize:        uint64(memory) * 1024 * 1024, // simplified
		})
		if err != nil {
			return fmt.Errorf("define VM: %w", err)
		}
		if !resp.Success {
			return fmt.Errorf("define VM: %s", resp.Error)
		}

		fmt.Printf("VM '%s' defined on host '%s' (UUID: %s)\n", name, targetHost.Name, resp.VmUuid)
		return nil
	},
}

func init() {
	clusterDefineCmd.Flags().StringP("disk", "d", "", "Backing image URL or path (required)")
	clusterDefineCmd.Flags().String("host", "", "Target host name (default: auto-place)")
	clusterDefineCmd.Flags().UintP("memory", "m", 512, "Memory in MiB")
	clusterDefineCmd.Flags().UintP("cpu", "c", 1, "Number of CPUs")
	clusterDefineCmd.Flags().StringP("storage", "s", "8G", "Disk size (e.g. 10G)")
}

// --- cluster start ---

var clusterStartCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a VM across the cluster",
	Long:  "Find and start a VM by name across all cluster hosts.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cluster.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		orch := cluster.NewOrchestrator(cfg)
		defer orch.Close()

		if err := orch.StartVM(args[0]); err != nil {
			return fmt.Errorf("start VM: %w", err)
		}
		fmt.Printf("VM '%s' started\n", args[0])
		return nil
	},
}

// --- cluster stop ---

var clusterStopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop a VM across the cluster",
	Long:  "Find and stop a VM by name across all cluster hosts.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cluster.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		orch := cluster.NewOrchestrator(cfg)
		defer orch.Close()

		if err := orch.StopVM(args[0]); err != nil {
			return fmt.Errorf("stop VM: %w", err)
		}
		fmt.Printf("VM '%s' stopped\n", args[0])
		return nil
	},
}

// --- cluster delete ---

var clusterDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a VM across the cluster",
	Long:  "Find and delete a VM by name across all cluster hosts.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cluster.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		orch := cluster.NewOrchestrator(cfg)
		defer orch.Close()

		confirm, _ := ui.ConfirmOperation()
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Delete cancelled.")
			return nil
		}

		if err := orch.DeleteVM(args[0]); err != nil {
			return fmt.Errorf("delete VM: %w", err)
		}
		fmt.Printf("VM '%s' deleted\n", args[0])
		return nil
	},
}

// --- cluster status ---

var clusterStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show VM status across the cluster",
	Long:  "Find and show the status of a VM across all cluster hosts.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cluster.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		orch := cluster.NewOrchestrator(cfg)
		defer orch.Close()

		if len(args) > 0 {
			vm, _, err := orch.FindVM(args[0])
			if err != nil {
				return fmt.Errorf("find VM: %w", err)
			}
			fmt.Printf("VM '%s' on host '%s': %s\n", vm.Name, vm.Host, vm.State)
			return nil
		}

		// Show all host statuses
		results := orch.ListHosts()
		for _, r := range results {
			if r.Err != nil {
				fmt.Printf("Host %s: error - %v\n", r.Host, r.Err)
				continue
			}
			health := "unhealthy"
			if r.Info.Healthy {
				health = "healthy"
			}
			fmt.Printf("Host %s [%s]: %d VMs\n", r.Info.Name, health, len(r.Info.Vms))
		}
		return nil
	},
}

// --- cluster xml ---

var clusterShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show VM XML definition across the cluster",
	Long:  "Find and show the XML definition of a VM across all cluster hosts.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cluster.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		orch := cluster.NewOrchestrator(cfg)
		defer orch.Close()

		xml, err := orch.GetVMXML(args[0])
		if err != nil {
			return fmt.Errorf("get XML: %w", err)
		}
		fmt.Println(xml)
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(clusterListCmd)
	clusterCmd.AddCommand(clusterVMCmd)
	clusterCmd.AddCommand(clusterDefineCmd)
	clusterCmd.AddCommand(clusterStartCmd)
	clusterCmd.AddCommand(clusterStopCmd)
	clusterCmd.AddCommand(clusterDeleteCmd)
	clusterCmd.AddCommand(clusterStatusCmd)
	clusterCmd.AddCommand(clusterShowCmd)
}
