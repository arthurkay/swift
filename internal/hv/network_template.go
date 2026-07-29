package hv

import (
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"net"
	"strings"

	"libvirt.org/go/libvirtxml"
)

// NetworkConfig holds a single NIC's network name and model for domain XML.
type NetworkConfig struct {
	Name  string
	Model string
}

// NetworkResources holds the parameters for building a libvirt network XML.
type NetworkResources struct {
	Name       string
	CIDR       string
	Mode       string // nat, route, bridge, macvtap, passthrough, isolated, open
	BridgeName string
	Device     string
	DHCP       bool
	DHCPStart  string
	DHCPEnd    string
}

// BuildNetworkXML produces the libvirt XML for this network definition.
func (r NetworkResources) BuildNetworkXML() (string, error) {
	netCfg := &libvirtxml.Network{
		Name: r.Name,
	}

	// Forward
	mode := strings.ToLower(r.Mode)
	if mode == "" {
		mode = "nat"
	}
	if mode != "isolated" && mode != "open" {
		fwd := &libvirtxml.NetworkForward{
			Mode: mode,
		}
		if r.Device != "" {
			fwd.Dev = r.Device
		}
		// Bridge mode needs a bridge name but no <forward> element
		if mode == "bridge" {
			fwd = nil
		}
		if fwd != nil {
			netCfg.Forward = fwd
		}
	}

	// Bridge
	if mode == "bridge" {
		// For bridge mode, the bridge device comes from the host
		if r.BridgeName != "" {
			netCfg.Bridge = &libvirtxml.NetworkBridge{
				Name: r.BridgeName,
			}
		}
	} else if mode != "macvtap" && mode != "passthrough" {
		// macvtap/passthrough don't have a libvirt-managed bridge
		// For nat/route/isolated, auto-generate a bridge name
		bridgeName := r.BridgeName
		if bridgeName == "" {
			bridgeName = fmt.Sprintf("virbr-%s", r.Name)
			if len(bridgeName) > 15 {
				hash := fmt.Sprintf("%x", sha256.Sum256([]byte(bridgeName)))
				bridgeName = bridgeName[:9] + hash[:6]
			}
		}
		netCfg.Bridge = &libvirtxml.NetworkBridge{
			Name:  bridgeName,
			STP:   "on",
			Delay: "0",
		}
	}

	// IP / DHCP
	if r.CIDR != "" {
		ip, prefix, err := parseCIDR(r.CIDR)
		if err != nil {
			return "", fmt.Errorf("invalid CIDR %q: %w", r.CIDR, err)
		}

		netIP := &libvirtxml.NetworkIP{
			Address: ip,
			Prefix:  uint(prefix),
		}

		if r.DHCP {
			dhcp := &libvirtxml.NetworkDHCP{}
			if r.DHCPStart != "" && r.DHCPEnd != "" {
				dhcp.Ranges = []libvirtxml.NetworkDHCPRange{
					{
						Start: r.DHCPStart,
						End:   r.DHCPEnd,
					},
				}
			}
			netIP.DHCP = dhcp
		}

		netCfg.IPs = []libvirtxml.NetworkIP{*netIP}
	}

	xmlBytes, err := xml.MarshalIndent(netCfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal network XML: %w", err)
	}

	return xml.Header + string(xmlBytes), nil
}

// parseCIDR splits "192.168.122.0/24" into ("192.168.122.0", 24).
func parseCIDR(cidr string) (string, int, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", 0, err
	}
	ones, _ := ipNet.Mask.Size()
	return ip.String(), ones, nil
}

// prefixToNetmask converts a CIDR prefix length to a dotted-decimal netmask.
func prefixToNetmask(prefix int) string {
	mask := net.CIDRMask(prefix, 32)
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

// NetworkResourcesDefaults applies sensible defaults to a NetworkResources.
func (r *NetworkResources) NetworkResourcesDefaults() {
	if r.Mode == "" {
		r.Mode = "nat"
	}
}
