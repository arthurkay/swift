package hv

import (
	"fmt"
	"strconv"

	"libvirt.org/go/libvirt"
)

// StartDomain boots a domain by name.
func (h *LibvirtHypervisor) StartDomain(name string) error {
	dom, err := h.conn.LookupDomainByName(name)
	if err != nil {
		return fmt.Errorf("lookup domain %q for start: %w", name, err)
	}
	if err := dom.Create(); err != nil {
		return fmt.Errorf("start domain %q: %w", name, err)
	}
	return nil
}

// StopDomain force-stops a domain by UUID.
func (h *LibvirtHypervisor) StopDomain(uuid string) error {
	dom, err := h.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("lookup domain UUID %q for stop: %w", uuid, err)
	}
	if err := dom.Destroy(); err != nil {
		return fmt.Errorf("stop domain UUID %q: %w", uuid, err)
	}
	return nil
}

// RebootDomain reboots a domain by UUID.
func (h *LibvirtHypervisor) RebootDomain(uuid string) error {
	dom, err := h.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return fmt.Errorf("lookup domain UUID %q for reboot: %w", uuid, err)
	}
	if err := dom.Reboot(libvirt.DOMAIN_REBOOT_DEFAULT); err != nil {
		return fmt.Errorf("reboot domain UUID %q: %w", uuid, err)
	}
	return nil
}

// DomainState returns the human-readable state of a domain by name.
func (h *LibvirtHypervisor) DomainState(name string) (string, error) {
	dom, err := h.conn.LookupDomainByName(name)
	if err != nil {
		return "", fmt.Errorf("lookup domain %q for state: %w", name, err)
	}
	state, _, err := dom.GetState()
	if err != nil {
		return "", fmt.Errorf("get state for domain %q: %w", name, err)
	}
	return domainStateStr(state), nil
}

// DomainXML returns the full libvirt XML definition of a domain by name.
func (h *LibvirtHypervisor) DomainXML(name string) (string, error) {
	dom, err := h.conn.LookupDomainByName(name)
	if err != nil {
		return "", fmt.Errorf("lookup domain %q for XML: %w", name, err)
	}
	xml, err := dom.GetXMLDesc(0)
	if err != nil {
		return "", fmt.Errorf("get XML for domain %q: %w", name, err)
	}
	return xml, nil
}

// domainStateStr converts a libvirt domain state to a human-readable string.
func domainStateStr(state libvirt.DomainState) string {
	switch state {
	case libvirt.DOMAIN_NOSTATE:
		return "Nostate"
	case libvirt.DOMAIN_RUNNING:
		return "Running"
	case libvirt.DOMAIN_BLOCKED:
		return "Blocked"
	case libvirt.DOMAIN_PAUSED:
		return "Paused"
	case libvirt.DOMAIN_SHUTDOWN:
		return "Shutdown"
	case libvirt.DOMAIN_SHUTOFF:
		return "Shutoff"
	case libvirt.DOMAIN_CRASHED:
		return "Crashed"
	case libvirt.DOMAIN_PMSUSPENDED:
		return "PmSuspended"
	default:
		return "Unknown"
	}
}

// parseID attempts to parse a string as a numeric domain ID.
func parseID(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return id, nil
}
