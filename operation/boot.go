package operation

import (
	"fmt"

	"github.com/digitalocean/go-libvirt"
)

func domState(state int32) string {
	var stateText string
	switch state {
	case 1:
		stateText = "Running"
	case 2:
		stateText = "Blocked"
	case 3:
		stateText = "Paused"
	case 4:
		stateText = "Shutdown"
	case 5:
		stateText = "Shutoff"
	case 6:
		stateText = "Crashed"
	case 7:
		stateText = "PmSuspended"
	default:
		stateText = "Nostate"
	}
	return stateText
}

func StartUp(name string, l *libvirt.Libvirt) {
	domain, err := l.DomainLookupByName(name)
	if err != nil {
		fmt.Printf("Unable to get domain because: %v\n", err)
	}
	er := l.DomainCreate(domain)
	if er != nil {
		fmt.Printf("Unable to boot up domain because: %v\n", er)
	}
}

func ShutDown(uuid libvirt.UUID, l *libvirt.Libvirt) {
	domain, err := l.DomainLookupByUUID(uuid)
	if err != nil {
		fmt.Printf("Unable to find selected domain: %v\n", err)
	}
	er := l.DomainDestroy(domain)
	if er != nil {
		fmt.Printf("Unable to shutdown VM because %v\n", er)
	}
}

func DomainState(domain libvirt.Domain, l *libvirt.Libvirt) (string, error) {
	params := libvirt.DomainGetStateArgs{
		Dom:   domain,
		Flags: uint32(libvirt.DomainNostate),
	}
	state, _, er := l.DomainGetState(params.Dom, params.Flags)
	if er != nil {
		return "", er
	} else {
		return domState(state), nil
	}
}

func Reboot(uuid libvirt.UUID, l *libvirt.Libvirt) {
	domain, err := l.DomainLookupByUUID(uuid)
	if err != nil {
		fmt.Printf("Unable to get domain %v", err)
	}
	params := libvirt.DomainRebootArgs{
		Dom:   domain,
		Flags: libvirt.DomainRebootDefault,
	}
	er := l.DomainReboot(params.Dom, params.Flags)
	if er != nil {
		fmt.Printf("Unable to reboot node %v", er)
	}
}
