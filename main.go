package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"petricoh/operation"
	"text/tabwriter"
	"time"

	"github.com/digitalocean/go-libvirt"
	libvirtxml "libvirt.org/libvirt-go-xml"
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
	tabWriter.Flush()

	domainTemplate := &libvirtxml.Domain{
		Type: "kvm",
		Name: "Test Instance Node",
		Memory: &libvirtxml.DomainMemory{
			Value: 1024,
			Unit:  "MiB",
		},
		VCPU: &libvirtxml.DomainVCPU{
			Value: 1,
		},
		OS: &libvirtxml.DomainOS{
			Type: &libvirtxml.DomainOSType{
				Arch: "x86_64",
				Type: "hvm",
			},
		},
		Devices: &libvirtxml.DomainDeviceList{
			Graphics: []libvirtxml.DomainGraphic{
				{
					Spice: &libvirtxml.DomainGraphicSpice{
						AutoPort: "yes",
						Image: &libvirtxml.DomainGraphicSpiceImage{
							Compression: "off",
						},
					},
				},
			},
			Disks: []libvirtxml.DomainDisk{
				{
					Device: "disk",
					Driver: &libvirtxml.DomainDiskDriver{
						Name: "qemu",
						Type: "qcow2",
					},
					Target: &libvirtxml.DomainDiskTarget{
						Dev: "hda",
						Bus: "ide",
					},
					Source: &libvirtxml.DomainDiskSource{
						File: &libvirtxml.DomainDiskSourceFile{
							File: "/home/arthur/Downloads/plod.img",
						},
					},
				},
				{
					Device: "cdrom",
					Target: &libvirtxml.DomainDiskTarget{
						Dev: "hdb",
						Bus: "ide",
					},
					Source: &libvirtxml.DomainDiskSource{
						File: &libvirtxml.DomainDiskSourceFile{
							File: "/home/arthur/Downloads/AlmaLinux-8.5-x86_64-boot.iso",
						},
					},
				},
			},
			Interfaces: []libvirtxml.DomainInterface{
				{
					Model: &libvirtxml.DomainInterfaceModel{
						Type: "e1000",
					},
					Source: &libvirtxml.DomainInterfaceSource{
						Network: &libvirtxml.DomainInterfaceSourceNetwork{
							Network: "default",
						},
					},
				},
			},
		},
	}

	xmlDoc, err := domainTemplate.Marshal()
	if err != nil {
		log.Fatalf("Oops! %v", err)
	}
	dom, err := operation.Define(xmlDoc, l)
	if err != nil {
		log.Fatalf("Oopsy! %v", err)
	}
	fmt.Printf("Response: %s\n", dom.Name)
	if err := l.Disconnect(); err != nil {
		log.Fatalf("failed to disconnect: %v", err)
	}
}
