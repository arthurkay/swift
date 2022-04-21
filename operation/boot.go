package operation

import (
	"fmt"

	"github.com/digitalocean/go-libvirt"
)

func domState(state int32) string {
	var stateText string
	switch state {
	case 1:
		stateText = "DomainRunning"
	case 2:
		stateText = "DomainBlocked"
	case 3:
		stateText = "DomainPaused"
	case 4:
		stateText = "DomainShutdown"
	case 5:
		stateText = "DomainShutoff"
	case 6:
		stateText = "DomainCrashed"
	case 7:
		stateText = "DomainPmSuspended"
	default:
		stateText = "DomainNostate"
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

func DomainState(name string, l *libvirt.Libvirt) {
	domain, err := l.DomainLookupByName(name)
	if err != nil {
		fmt.Printf("Unable to get domain instance state, %v\n", err)
	}
	params := libvirt.DomainGetStateArgs{
		Dom:   domain,
		Flags: uint32(libvirt.DomainNostate),
	}
	state, reason, er := l.DomainGetState(params.Dom, params.Flags)
	if er != nil {
		fmt.Printf("Unable to get state:  %v\n", er)
	} else {
		fmt.Printf("%s, %d\n", domState(state), reason)
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
