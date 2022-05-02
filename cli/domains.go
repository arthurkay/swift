package cli

import (
	"fmt"
	"os"
	"swift/domain"
	"swift/operation"
	"swift/utils"

	"github.com/digitalocean/go-libvirt"
	"github.com/gosimple/slug"
	"github.com/spf13/cobra"
)

func ListDomains() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "Show all domains on this system",
		Run: func(cmd *cobra.Command, args []string) {
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			defer l.Disconnect()
			domain.AllDomains(l)
		},
		Aliases: []string{"domains"},
	}
	return cmd
}

func UndefineDomain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undefine",
		Short: "Undefine a domain from the hypervisor",
		Run: func(cmd *cobra.Command, args []string) {
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			defer l.Disconnect()
			if len(args) == 0 {
				cmd.Usage()
				return
			}
			domain, err := vmInstance(args[0])
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			if er := operation.Undefine(domain, l); er != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			configDir, erro := utils.SwiftHome()
			if erro != nil {
				fmt.Printf("%v", erro)
				return
			}
			path := configDir + "/" + slug.Make(domain.Name)
			os.RemoveAll(path)
		},
	}
	return cmd
}

func StartDomain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start up a domain from the hypervisor",
		Run: func(cmd *cobra.Command, args []string) {
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			defer l.Disconnect()
			if len(args) == 0 {
				cmd.Usage()
				return
			}
			dom, err := vmInstance(args[0])
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			operation.StartUp(dom.Name, l)
			er := domain.DomainConsole(dom, libvirt.OptString{"console", "spice", "desktop"}, l)
			fmt.Printf("%v", er)
		},
	}
	return cmd
}

func ShutdownDomain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poweroff",
		Short: "Shutdown a domain from the hypervisor",
		Run: func(cmd *cobra.Command, args []string) {
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			defer l.Disconnect()
			if len(args) == 0 {
				cmd.Usage()
				return
			}
			domain, err := vmInstance(args[0])
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			operation.ShutDown(domain.UUID, l)
		},
	}
	return cmd
}

func DomainState() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get domain status",
		Run: func(cmd *cobra.Command, args []string) {
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			defer l.Disconnect()
			if len(args) == 0 {
				cmd.Usage()
				return
			}
			domain, err := vmInstance(args[0])
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			state, err := operation.DomainState(domain, l)
			if err != nil {
				fmt.Printf("%v\n", err)
				return
			}
			fmt.Printf("%s\n", state)
		},
	}
	return cmd
}
