package main

import (
	"fmt"
	"log"
	"petricoh/config"
	"petricoh/operation"
	"petricoh/utils"

	"github.com/digitalocean/go-libvirt"
)

func main() {

	l := libvirt.NewWithDialer(&operation.DomainInstance{
		Socket: "/var/run/libvirt/libvirt-sock",
	})

	if err := l.Connect(); err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	v, err := l.ConnectGetLibVersion()
	if err != nil {
		log.Fatalf("failed to retrieve libvirt version: %v", err)
	}
	fmt.Println("Version:", v)

	/* params := &libvirt.ConnectListAllDomainsArgs{
		NeedResults: int32(libvirt.ConnectListDomainsPersistent),
		Flags:       libvirt.ConnectListAllDomainsFlags(libvirt.ConnectListDomainsPersistent),
	} */
	/* domains, _, err := l.ConnectListAllDomains(params.NeedResults, params.Flags)
	if err != nil {
		log.Fatalf("failed to retrieve domains: %v", err)
	} */

	/* tabWriter := tabwriter.NewWriter(os.Stdout, 2, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(tabWriter, "ID\tName\t\tUUID")
	for _, d := range domains {
		fmt.Fprintf(tabWriter, "%d\t%s\t%x\n", d.ID, d.Name, d.UUID)
	} */
	//operation.Undefine(domains[0], l)

	inst := config.ComputeResources{
		Name:    "Test Instance",
		Memory:  1024,
		Vcpu:    2,
		BootIso: "/home/arthur/Downloads/ubuntu-20.10-live-server-amd64.iso",
		//BootIso: "/home/arthur/Documents/Dev/misc/libvirt-go/Images/centos/CentOS-Stream-ec2-8-20200113.0.x86_64.qcow2",
		MacAddr: utils.NewMacAddress(),
	}

	inst.VirtInstall(l)
	//tabWriter.Flush()

	if err := l.Disconnect(); err != nil {
		log.Fatalf("failed to disconnect: %v", err)
	}
	/* routes := web.Routes()
	fmt.Printf("Application up and running")
	err := http.ListenAndServe(fmt.Sprintf(":%s", "8000"), routes)
	if err != nil {
		panic(err)
	} */
}
