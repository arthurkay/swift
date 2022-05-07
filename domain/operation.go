package domain

import (
	"fmt"
	"os"
	"strconv"
	"swift/operation"
	"text/tabwriter"

	"github.com/digitalocean/go-libvirt"
)

var (
	cyan  = "\033[4;36m" + "\033[1;36m"
	clear = "\033[0m"
)

func AllDomains(l *libvirt.Libvirt) {
	domains, err := DefinedDomains(l)
	if err != nil {
		fmt.Printf("Oops! %v\n", err)
	}
	tabWriter := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Fprintln(tabWriter, cyan+"ID\tName\tState\tUUID\t"+clear)
	for i, d := range domains {
		status, err := operation.DomainState(d, l)
		if err != nil {
			status = "Unknown"
		}
		id := strconv.Itoa(i + 1)
		uuid := fmt.Sprintf("%x", d.UUID)
		fmt.Fprintln(tabWriter, cyan+id+clear+"\t"+d.Name+"\t"+status+"\t"+string(uuid)+"\t")
	}
	tabWriter.Flush()
}

func DefinedDomains(l *libvirt.Libvirt) ([]libvirt.Domain, error) {
	params := &libvirt.ConnectListAllDomainsArgs{
		NeedResults: int32(libvirt.ConnectListDomainsPersistent),
		Flags:       libvirt.ConnectListAllDomainsFlags(libvirt.ConnectListDomainsPersistent),
	}
	domains, _, err := l.ConnectListAllDomains(params.NeedResults, params.Flags)
	return domains, err
}

func VmNames(l *libvirt.Libvirt) ([]string, error) {
	var names []string
	params := &libvirt.ConnectListAllDomainsArgs{
		NeedResults: int32(libvirt.ConnectListDomainsPersistent),
		Flags:       libvirt.ConnectListAllDomainsFlags(libvirt.ConnectListDomainsPersistent),
	}
	domains, _, err := l.ConnectListAllDomains(params.NeedResults, params.Flags)
	for _, domain := range domains {
		names = append(names, domain.Name)
	}
	return names, err
}

func DomainConsole(domain libvirt.Domain, devName libvirt.OptString, l *libvirt.Libvirt) error {
	params := &libvirt.DomainOpenConsoleArgs{
		Dom:     domain,
		DevName: devName,
		Flags:   0,
	}
	return l.DomainOpenConsole(params.Dom, params.DevName, os.Stdout, params.Flags)
}
