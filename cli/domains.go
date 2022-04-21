package cli

import (
	"fmt"
	"swift/domain"
	"swift/operation"
	"swift/utils"

	"github.com/digitalocean/go-libvirt"
	"github.com/spf13/cobra"
)

func ListDomains() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "Everything that has to do with managing VM domain instances",
		Run: func(cmd *cobra.Command, args []string) {
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
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
			var domain libvirt.Domain
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			if len(args) == 0 {
				cmd.Usage()
				return
			}
			var name = args[0]
			domain, err = l.DomainLookupByName(name)
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			operation.Undefine(domain, l)
		},
	}
	return cmd
}

func StartDomain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start up a domain from the hypervisor",
		Run: func(cmd *cobra.Command, args []string) {
			var domain libvirt.Domain
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			if len(args) == 0 {
				cmd.Usage()
				return
			}
			var name = args[0]
			domain, err = l.DomainLookupByName(name)
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			operation.StartUp(domain.Name, l)
		},
	}
	return cmd
}

func ShutdownDomain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poweroff",
		Short: "Shutdown a domain from the hypervisor",
		Run: func(cmd *cobra.Command, args []string) {
			var domain libvirt.Domain
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			if len(args) == 0 {
				cmd.Usage()
				return
			}
			var name = args[0]
			domain, err = l.DomainLookupByName(name)
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
			var domain libvirt.Domain
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			if len(args) == 0 {
				cmd.Usage()
				return
			}
			var name = args[0]
			domain, err = l.DomainLookupByName(name)
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			operation.DomainState(domain.Name, l)
		},
	}
	return cmd
}
