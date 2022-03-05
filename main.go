package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"petricoh/config"
	"petricoh/utils"
	"text/tabwriter"
	"time"

	"github.com/digitalocean/go-libvirt"
)

var (
	green = "\033[4;32m"
	clear = "\033[0m"
)

type domainInstance struct {
	Socket string
}

func (d *domainInstance) Dial() (net.Conn, error) {
	c, err := net.DialTimeout("unix", d.Socket, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	return c, nil
}

func main() {

	l := libvirt.NewWithDialer(&domainInstance{
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

	params := &libvirt.ConnectListAllDomainsArgs{
		NeedResults: int32(libvirt.ConnectListDomainsPersistent),
		Flags:       libvirt.ConnectListAllDomainsFlags(libvirt.ConnectListDomainsPersistent),
	}
	domains, _, err := l.ConnectListAllDomains(params.NeedResults, params.Flags)
	if err != nil {
		log.Fatalf("failed to retrieve domains: %v", err)
	}

	tabWriter := tabwriter.NewWriter(os.Stdout, 2, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(tabWriter, green+"ID\tName\t\tUUID"+clear)
	for _, d := range domains {
		fmt.Fprintf(tabWriter, "%d\t%s\t%x\n", d.ID, d.Name, d.UUID)
	}
	//operation.Undefine(domains[0], l)

	inst := config.ComputeResources{
		Name:     "Test Instance",
		Memory:   1024,
		Vcpu:     2,
		ImageIso: "/home/arthur/Documents/Dev/misc/libvirt-go/Images/cloud.img",
		BootIso:  "/home/arthur/Documents/Dev/misc/libvirt-go/Images/ubuntu-bliss.img",
		MacAddr:  utils.NewMacAddress(),
	}

	inst.VirtInstall(l)
	tabWriter.Flush()

	if err := l.Disconnect(); err != nil {
		log.Fatalf("failed to disconnect: %v", err)
	}
}
