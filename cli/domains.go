package cli

import (
	"fmt"
	"os"
	"swift/domain"
	"swift/operation"
	"swift/utils"

	"github.com/gosimple/slug"
	"github.com/spf13/cobra"

	"libvirt.org/go/libvirt"
)

func ListDomains() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "Show all vms on this system",
		Run: func(cmd *cobra.Command, args []string) {
			l, err := utils.InitLib()
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			defer l.Disconnect()
			domain.AllDomains(l)
		},
		Aliases: []string{"vms"},
	}
	return cmd
}

func UndefineDomain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete a vm from the hypervisor",
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
		Use:   "start",
		Short: "Start up a vm from the hypervisor",
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
		},
	}
	return cmd
}

func ShutdownDomain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poweroff",
		Short: "Shutdown a vm from the hypervisor",
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
		Short: "Get vm status",
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

func GetVmXML() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "xml",
		Short: "Display vm instance xml details",
		Run: func(cmd *cobra.Command, args []string) {
			conn, err := libvirt.NewConnect("qemu:///system")
			if err != nil {
				fmt.Printf("Oops! %v\n", err)
				return
			}
			if len(args) == 0 {
				cmd.Usage()
				return
			}
			domainDetails, er := vmInstance(args[0])
			if er != nil {
				fmt.Printf("Oops! %v\n", er)
				return
			}
			dom, e := conn.LookupDomainByName(domainDetails.Name)
			if e != nil {
				fmt.Printf("Oops! %v\n", e)
				return
			}
			xml, erro := dom.GetXMLDesc(0)
			if erro != nil {
				fmt.Printf("Oops! %v\n", erro)
				return
			}
			fmt.Printf("%s", xml)
		},
	}
	return cmd
}
