package domain

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/digitalocean/go-libvirt"
)

var (
	cyan  = "\033[4;36m"
	clear = "\033[0m"
)

func AllDomains(l *libvirt.Libvirt) {
	params := &libvirt.ConnectListAllDomainsArgs{
		NeedResults: int32(libvirt.ConnectListDomainsPersistent),
		Flags:       libvirt.ConnectListAllDomainsFlags(libvirt.ConnectListDomainsPersistent),
	}
	domains, _, err := l.ConnectListAllDomains(params.NeedResults, params.Flags)
	if err != nil {
		fmt.Printf("Oops! %v\n", err)
	}
	tabWriter := tabwriter.NewWriter(os.Stdout, 2, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(tabWriter, cyan+"ID\tName\t\tUUID"+clear)
	for _, d := range domains {
		fmt.Fprintf(tabWriter, "%d\t%s\t%x\n", d.ID, d.Name, d.UUID)
	}
	tabWriter.Flush()
}
