package hv

import (
	"fmt"
	"net"

	"libvirt.org/go/libvirt"
)

// NetworkInfo is a library-agnostic representation of a virtual network.
type NetworkInfo struct {
	Name        string
	UUID        string
	State       string
	BridgeName  string
	ForwardMode string
	CIDR        string
	Autostart   bool
}

// ListNetworks returns all defined networks.
func (h *LibvirtHypervisor) ListNetworks() ([]NetworkInfo, error) {
	networks, err := h.conn.ListAllNetworks(libvirt.CONNECT_LIST_NETWORKS_PERSISTENT)
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	result := make([]NetworkInfo, 0, len(networks))
	for _, n := range networks {
		info, err := networkInfo(&n)
		if err != nil {
			continue
		}
		result = append(result, *info)
	}
	return result, nil
}

// NetworkNames returns just the names of all defined networks.
func (h *LibvirtHypervisor) NetworkNames() ([]string, error) {
	networks, err := h.ListNetworks()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(networks))
	for i, n := range networks {
		names[i] = n.Name
	}
	return names, nil
}

// LookupNetwork finds a network by name.
func (h *LibvirtHypervisor) LookupNetwork(name string) (*NetworkInfo, error) {
	net, err := h.conn.LookupNetworkByName(name)
	if err != nil {
		return nil, fmt.Errorf("lookup network %q: %w", name, err)
	}
	return networkInfo(net)
}

// DefineNetwork registers a new network from its XML definition.
func (h *LibvirtHypervisor) DefineNetwork(xml string) error {
	_, err := h.conn.NetworkDefineXML(xml)
	return err
}

// SetNetworkAutostart enables or disables autostart for a network.
func (h *LibvirtHypervisor) SetNetworkAutostart(name string, autostart bool) error {
	net, err := h.conn.LookupNetworkByName(name)
	if err != nil {
		return fmt.Errorf("lookup network %q for autostart: %w", name, err)
	}
	return net.SetAutostart(autostart)
}

// UndefineNetwork removes a network by name.
func (h *LibvirtHypervisor) UndefineNetwork(name string) error {
	net, err := h.conn.LookupNetworkByName(name)
	if err != nil {
		return fmt.Errorf("lookup network %q for undefine: %w", name, err)
	}
	if err := net.Undefine(); err != nil {
		return fmt.Errorf("undefine network %q: %w", name, err)
	}
	return nil
}

// StartNetwork starts a defined network.
func (h *LibvirtHypervisor) StartNetwork(name string) error {
	net, err := h.conn.LookupNetworkByName(name)
	if err != nil {
		return fmt.Errorf("lookup network %q for start: %w", name, err)
	}
	if err := net.Create(); err != nil {
		return fmt.Errorf("start network %q: %w", name, err)
	}
	return nil
}

// StopNetwork stops a running network.
func (h *LibvirtHypervisor) StopNetwork(name string) error {
	net, err := h.conn.LookupNetworkByName(name)
	if err != nil {
		return fmt.Errorf("lookup network %q for stop: %w", name, err)
	}
	if err := net.Destroy(); err != nil {
		return fmt.Errorf("stop network %q: %w", name, err)
	}
	return nil
}

// NetworkState returns the state of a network.
func (h *LibvirtHypervisor) NetworkState(name string) (string, error) {
	net, err := h.conn.LookupNetworkByName(name)
	if err != nil {
		return "", fmt.Errorf("lookup network %q for state: %w", name, err)
	}
	active, err := net.IsActive()
	if err != nil {
		return "", fmt.Errorf("get state for network %q: %w", name, err)
	}
	if active {
		return "Active", nil
	}
	return "Inactive", nil
}

// NetworkXML returns the full libvirt XML definition of a network.
func (h *LibvirtHypervisor) NetworkXML(name string) (string, error) {
	net, err := h.conn.LookupNetworkByName(name)
	if err != nil {
		return "", fmt.Errorf("lookup network %q for XML: %w", name, err)
	}
	xml, err := net.GetXMLDesc(0)
	if err != nil {
		return "", fmt.Errorf("get XML for network %q: %w", name, err)
	}
	return xml, nil
}

// NetworkBridge returns the bridge device name for a network.
func (h *LibvirtHypervisor) NetworkBridge(name string) (string, error) {
	net, err := h.conn.LookupNetworkByName(name)
	if err != nil {
		return "", fmt.Errorf("lookup network %q for bridge: %w", name, err)
	}
	bridge, err := net.GetBridgeName()
	if err != nil {
		return "", fmt.Errorf("get bridge for network %q: %w", name, err)
	}
	return bridge, nil
}

// networkInfo extracts NetworkInfo from a libvirt Network.
func networkInfo(net *libvirt.Network) (*NetworkInfo, error) {
	name, err := net.GetName()
	if err != nil {
		return nil, fmt.Errorf("get network name: %w", err)
	}
	uuid, err := net.GetUUIDString()
	if err != nil {
		return nil, fmt.Errorf("get network UUID: %w", err)
	}
	active, err := net.IsActive()
	if err != nil {
		return nil, fmt.Errorf("get network state: %w", err)
	}
	state := "Inactive"
	if active {
		state = "Active"
	}
	autostart, err := net.GetAutostart()
	if err != nil {
		autostart = false
	}

	bridgeName, _ := net.GetBridgeName()

	// Parse XML to get forward mode and CIDR
	xmlDesc, _ := net.GetXMLDesc(0)
	forwardMode, cidR := parseNetworkXML(xmlDesc)

	return &NetworkInfo{
		Name:        name,
		UUID:        uuid,
		State:       state,
		BridgeName:  bridgeName,
		ForwardMode: forwardMode,
		CIDR:        cidR,
		Autostart:   autostart,
	}, nil
}

// parseNetworkXML extracts forward mode and CIDR from network XML.
func parseNetworkXML(xmlDesc string) (forwardMode, cidr string) {
	// Simple XML parsing for forward mode
	if contains(xmlDesc, "<forward") {
		if idx := indexOf(xmlDesc, "mode='"); idx != -1 {
			start := idx + 7
			end := indexOf(xmlDesc[start:], "'")
			if end != -1 {
				forwardMode = xmlDesc[start : start+end]
			}
		}
	} else {
		forwardMode = "isolated"
	}

	// Extract IP address and netmask to compute CIDR
	if ipIdx := indexOf(xmlDesc, "<ip address='"); ipIdx != -1 {
		ipStart := ipIdx + 13
		ipEnd := indexOf(xmlDesc[ipStart:], "'")
		if ipEnd != -1 {
			ipAddr := xmlDesc[ipStart : ipStart+ipEnd]

			netmask := ""
			if maskIdx := indexOf(xmlDesc[ipStart+ipEnd:], "netmask='"); maskIdx != -1 {
				mStart := ipStart + ipEnd + maskIdx + 9
				mEnd := indexOf(xmlDesc[mStart:], "'")
				if mEnd != -1 {
					netmask = xmlDesc[mStart : mStart+mEnd]
				}
			}

			if ipAddr != "" && netmask != "" {
				if cidrVal, err := netmaskToCIDR(ipAddr, netmask); err == nil {
					cidr = cidrVal
				}
			}
		}
	}

	return forwardMode, cidr
}

// netmaskToCIDR converts an IP address and netmask to CIDR notation.
func netmaskToCIDR(ip, netmask string) (string, error) {
	mask := net.ParseIP(netmask)
	if mask == nil {
		return "", fmt.Errorf("invalid netmask: %s", netmask)
	}
	mask4 := mask.To4()
	if mask4 == nil {
		return "", fmt.Errorf("non-IPv4 netmask: %s", netmask)
	}
	ones, _ := net.IPMask(mask4).Size()
	return fmt.Sprintf("%s/%d", ip, ones), nil
}

func contains(s, substr string) bool {
	return indexOf(s, substr) != -1
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
