package cli

import (
	"fmt"
	"strconv"
	"swift/domain"
	"swift/utils"

	"github.com/digitalocean/go-libvirt"
)

func vmInstance(arg string) (libvirt.Domain, error) {
	switch _, err := strconv.Atoi(arg); {
	case err != nil:
		return nameUUID(arg)
	case err == nil:
		id, _ := strconv.Atoi(arg)
		return iD(id)
	default:
		return empty()
	}
}

func nameUUID(arg string) (libvirt.Domain, error) {
	l, err := utils.InitLib()
	if err != nil {
		return libvirt.Domain{}, fmt.Errorf("%v", err)
	}
	defer l.Disconnect()
	// libvirt.UUID is an hex value of length 32
	// If a string provided is 32 characters long, start by checking
	// if the instance exists by UUID otherwise, check by name
	if len(arg) == 32 {
		var uuid libvirt.UUID
		copy(uuid[:], []byte(arg))
		dom, err := l.DomainLookupByUUID(uuid)
		if err != nil {
			return l.DomainLookupByName(arg)
		}
		return dom, err
	}
	return l.DomainLookupByName(arg)
}

func iD(id int) (libvirt.Domain, error) {
	l, err := utils.InitLib()
	if err != nil {
		return libvirt.Domain{}, fmt.Errorf("%v", err)
	}
	defer l.Disconnect()
	domains, err := domain.DefinedDomains(l)
	if err != nil {
		return libvirt.Domain{}, fmt.Errorf("%v", err)
	}
	idIndex := id - 1
	if id > len(domains) || id <= 0 {
		return libvirt.Domain{}, fmt.Errorf("there is no vm instance with ID %d", id)
	}
	return domains[idIndex], nil
}

func empty() (libvirt.Domain, error) {
	return libvirt.Domain{}, fmt.Errorf("%s", "unknown vm instance data")
}
