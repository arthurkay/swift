package hv

import (
	"fmt"
	"libvirt.org/go/libvirt"
)

// ListDomains returns all persistent domains.
func (h *LibvirtHypervisor) ListDomains() ([]DomainInfo, error) {
	domains, err := h.conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_PERSISTENT)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	result := make([]DomainInfo, 0, len(domains))
	for _, d := range domains {
		info, err := domainInfo(&d)
		if err != nil {
			continue
		}
		result = append(result, *info)
	}
	return result, nil
}

// LookupDomain finds a domain by name, UUID, or numeric ID.
func (h *LibvirtHypervisor) LookupDomain(arg string) (*DomainInfo, error) {
	// Try numeric ID
	if id, err := parseID(arg); err == nil {
		return h.LookupDomainByID(id)
	}
	// Try UUID (32 hex chars without dashes)
	if len(arg) == 32 {
		info, err := h.LookupDomainByUUID(arg)
		if err == nil {
			return info, nil
		}
	}
	// Fall back to name
	return h.LookupDomainByName(arg)
}

// LookupDomainByName finds a domain by its name.
func (h *LibvirtHypervisor) LookupDomainByName(name string) (*DomainInfo, error) {
	dom, err := h.conn.LookupDomainByName(name)
	if err != nil {
		return nil, fmt.Errorf("lookup domain %q: %w", name, err)
	}
	return domainInfo(dom)
}

// LookupDomainByUUID finds a domain by its UUID string.
func (h *LibvirtHypervisor) LookupDomainByUUID(uuid string) (*DomainInfo, error) {
	dom, err := h.conn.LookupDomainByUUIDString(uuid)
	if err != nil {
		return nil, fmt.Errorf("lookup domain UUID %q: %w", uuid, err)
	}
	return domainInfo(dom)
}

// LookupDomainByID finds a domain by its numeric ID.
func (h *LibvirtHypervisor) LookupDomainByID(id int) (*DomainInfo, error) {
	dom, err := h.conn.LookupDomainById(uint32(id))
	if err != nil {
		return nil, fmt.Errorf("lookup domain ID %d: %w", id, err)
	}
	return domainInfo(dom)
}

// DefineDomain registers a new domain from its XML definition.
func (h *LibvirtHypervisor) DefineDomain(xml string) error {
	_, err := h.conn.DomainDefineXML(xml)
	if err != nil {
		return fmt.Errorf("define domain: %w", err)
	}
	return nil
}

// UndefineDomain removes a domain by name.
func (h *LibvirtHypervisor) UndefineDomain(name string) error {
	dom, err := h.conn.LookupDomainByName(name)
	if err != nil {
		return fmt.Errorf("lookup domain %q for undefine: %w", name, err)
	}
	if err := dom.Undefine(); err != nil {
		return fmt.Errorf("undefine domain %q: %w", name, err)
	}
	return nil
}

// DomainNames returns just the names of all persistent domains.
func (h *LibvirtHypervisor) DomainNames() ([]string, error) {
	domains, err := h.ListDomains()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(domains))
	for i, d := range domains {
		names[i] = d.Name
	}
	return names, nil
}

// domainInfo extracts DomainInfo from a libvirt Domain.
func domainInfo(dom *libvirt.Domain) (*DomainInfo, error) {
	name, err := dom.GetName()
	if err != nil {
		return nil, fmt.Errorf("get domain name: %w", err)
	}
	uuid, err := dom.GetUUIDString()
	if err != nil {
		return nil, fmt.Errorf("get domain UUID: %w", err)
	}
	state, _, err := dom.GetState()
	if err != nil {
		return nil, fmt.Errorf("get domain state: %w", err)
	}
	id, _ := dom.GetID()
	return &DomainInfo{
		Name:  name,
		UUID:  uuid,
		State: domainStateStr(state),
		ID:    int(id),
	}, nil
}
