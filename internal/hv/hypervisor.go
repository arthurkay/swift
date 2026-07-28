package hv

import (
	"fmt"

	"libvirt.org/go/libvirt"
)

// DomainInfo is a library-agnostic representation of a VM domain.
type DomainInfo struct {
	Name  string
	UUID  string
	State string
	ID    int
}

// Hypervisor abstracts libvirt domain operations for testability.
type Hypervisor interface {
	ListDomains() ([]DomainInfo, error)
	LookupDomain(name string) (*DomainInfo, error)
	LookupDomainByUUID(uuid string) (*DomainInfo, error)
	LookupDomainByID(id int) (*DomainInfo, error)
	DefineDomain(xml string) error
	UndefineDomain(name string) error
	StartDomain(name string) error
	StopDomain(uuid string) error
	RebootDomain(uuid string) error
	DomainState(name string) (string, error)
	DomainXML(name string) (string, error)
}

// LibvirtHypervisor is the production implementation using libvirt.
type LibvirtHypervisor struct {
	conn *libvirt.Connect
}

// Connect establishes a connection to the libvirt daemon.
func Connect() (*LibvirtHypervisor, error) {
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return nil, fmt.Errorf("connect to libvirt: %w", err)
	}
	return &LibvirtHypervisor{conn: conn}, nil
}

// Disconnect closes the libvirt connection.
func (h *LibvirtHypervisor) Disconnect() {
	if h.conn != nil {
		h.conn.Close()
	}
}

// RawConn returns the underlying libvirt connection for advanced usage.
func (h *LibvirtHypervisor) RawConn() *libvirt.Connect {
	return h.conn
}
