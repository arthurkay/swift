package operation

import (
	"fmt"

	"github.com/digitalocean/go-libvirt"
)

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

func DomainState(name string, l *libvirt.Libvirt) (state int32, reason int32, err error) {
	domain, err := l.DomainLookupByName(name)
	if err != nil {
		return 5, 5, fmt.Errorf("unable to get domain instance state, %v", err)
	}
	params := libvirt.DomainGetStateArgs{
		Dom:   domain,
		Flags: uint32(libvirt.DomainNostate),
	}
	return l.DomainGetState(params.Dom, params.Flags)
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
