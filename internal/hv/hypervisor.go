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

// Hypervisor abstracts libvirt domain and network operations for testability.
type Hypervisor interface {
	// Domain operations
	ListDomains() ([]DomainInfo, error)
	DomainNames() ([]string, error)
	LookupDomain(name string) (*DomainInfo, error)
	LookupDomainByUUID(uuid string) (*DomainInfo, error)
	LookupDomainByID(id int) (*DomainInfo, error)
	DefineDomain(xml string) error
	UndefineDomain(name string) error
	SetDomainAutostart(name string, autostart bool) error
	StartDomain(name string) error
	ShutdownDomain(name string) error
	StopDomain(name string) error
	RebootDomain(name string) error
	DomainState(name string) (string, error)
	DomainXML(name string) (string, error)

	// Network operations
	ListNetworks() ([]NetworkInfo, error)
	NetworkNames() ([]string, error)
	LookupNetwork(name string) (*NetworkInfo, error)
	DefineNetwork(xml string) error
	UndefineNetwork(name string) error
	SetNetworkAutostart(name string, autostart bool) error
	StartNetwork(name string) error
	StopNetwork(name string) error
	NetworkState(name string) (string, error)
	NetworkXML(name string) (string, error)
	NetworkBridge(name string) (string, error)
}

// LibvirtHypervisor is the production implementation using libvirt.
type LibvirtHypervisor struct {
	conn *libvirt.Connect
}

// Connect establishes a connection to the local libvirt daemon.
func Connect() (*LibvirtHypervisor, error) {
	return ConnectURI("qemu:///system")
}

// ConnectURI establishes a connection to a libvirt daemon at the given URI.
func ConnectURI(uri string) (*LibvirtHypervisor, error) {
	conn, err := libvirt.NewConnect(uri)
	if err != nil {
		return nil, fmt.Errorf("connect to libvirt %q: %w", uri, err)
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
