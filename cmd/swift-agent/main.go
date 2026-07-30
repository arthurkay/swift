package main

import (
	"fmt"
	"log"
	"os"

	"swift/internal/agent"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func main() {
	cmd := &cobra.Command{
		Use:   "swift-agent",
		Short: "Swift VM cluster agent daemon",
		Long:  "An agent that runs on each host, managing local VMs and exposing a gRPC API for cluster operations.",
	}

	var configFile string
	var name, address, libvirtURI string
	var port int
	var labels []string

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Start the agent daemon",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := agent.Config{
				Name:    name,
				Address: address,
				Libvirt: libvirtURI,
				APIPort: port,
				Labels:  parseLabels(labels),
			}

			// Load config file if provided
			if configFile != "" {
				data, err := os.ReadFile(configFile)
				if err != nil {
					log.Fatalf("read config: %v", err)
				}
				if err := yaml.Unmarshal(data, &cfg); err != nil {
					log.Fatalf("parse config: %v", err)
				}
			}

			if cfg.Name == "" {
				hostname, _ := os.Hostname()
				cfg.Name = hostname
			}
			if cfg.APIPort == 0 {
				cfg.APIPort = 9800
			}

			s, err := agent.NewServer(cfg)
			if err != nil {
				log.Fatalf("create server: %v", err)
			}
			defer s.Close()

			if err := s.Run(); err != nil {
				log.Fatalf("server error: %v", err)
			}
		},
	}

	runCmd.Flags().StringVar(&configFile, "config", "", "Path to agent config YAML")
	runCmd.Flags().StringVar(&name, "name", "", "Host name (default: hostname)")
	runCmd.Flags().StringVar(&address, "address", ":9800", "Listen address (host:port)")
	runCmd.Flags().StringVar(&libvirtURI, "uri", "qemu:///system", "Libvirt connection URI")
	runCmd.Flags().IntVarP(&port, "port", "p", 9800, "gRPC listen port")
	runCmd.Flags().StringArrayVar(&labels, "label", nil, "Host labels (key=value)")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check agent status",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("swift-agent: to be implemented with client")
		},
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install as systemd service",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("systemd installation: to be implemented")
		},
	}

	cmd.AddCommand(runCmd, statusCmd, installCmd)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func parseLabels(raw []string) map[string]string {
	labels := make(map[string]string)
	for _, l := range raw {
		for i := 0; i < len(l); i++ {
			if l[i] == '=' {
				labels[l[:i]] = l[i+1:]
				break
			}
		}
	}
	return labels
}
