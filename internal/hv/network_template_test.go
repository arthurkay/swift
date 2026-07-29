package hv

import (
	"strings"
	"testing"
)

func TestBuildNetworkXML_NAT(t *testing.T) {
	r := NetworkResources{
		Name: "nat-net",
		CIDR: "192.168.122.0/24",
		Mode: "nat",
	}

	xml, err := r.BuildNetworkXML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(xml, "<name>nat-net</name>") {
		t.Error("XML missing network name")
	}
	if !strings.Contains(xml, `mode="nat"`) {
		t.Error("XML missing nat forward mode")
	}
	if !strings.Contains(xml, "virbr-nat-net") {
		t.Error("XML missing auto-generated bridge name")
	}
	if !strings.Contains(xml, `address="192.168.122.0"`) {
		t.Error("XML missing IP address")
	}
}

func TestBuildNetworkXML_Route(t *testing.T) {
	r := NetworkResources{
		Name: "route-net",
		CIDR: "10.0.0.0/24",
		Mode: "route",
	}

	xml, err := r.BuildNetworkXML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(xml, `mode="route"`) {
		t.Error("XML missing route forward mode")
	}
	if !strings.Contains(xml, `address="10.0.0.0"`) {
		t.Error("XML missing IP address")
	}
}

func TestBuildNetworkXML_Isolated(t *testing.T) {
	r := NetworkResources{
		Name: "isolated-net",
		CIDR: "10.10.10.0/24",
		Mode: "isolated",
	}

	xml, err := r.BuildNetworkXML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Isolated should NOT have a <forward> element
	if strings.Contains(xml, "<forward") {
		t.Error("isolated network should not have forward element")
	}
	if !strings.Contains(xml, "virbr-iso") {
		t.Error("XML missing bridge name (truncated)")
	}
	if !strings.Contains(xml, `address="10.10.10.0"`) {
		t.Error("XML missing IP address")
	}
}

func TestBuildNetworkXML_Bridge(t *testing.T) {
	r := NetworkResources{
		Name:       "bridge-net",
		Mode:       "bridge",
		BridgeName: "br0",
	}

	xml, err := r.BuildNetworkXML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bridge mode should NOT have a <forward> element
	if strings.Contains(xml, "<forward") {
		t.Error("bridge network should not have forward element")
	}
	if !strings.Contains(xml, `name="br0"`) {
		t.Error("XML missing bridge device name")
	}
}

func TestBuildNetworkXML_DefaultMode(t *testing.T) {
	r := NetworkResources{
		Name: "default-net",
		CIDR: "192.168.100.0/24",
	}

	xml, err := r.BuildNetworkXML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty mode should default to nat
	if !strings.Contains(xml, `mode="nat"`) {
		t.Error("XML should default to nat mode")
	}
}

func TestBuildNetworkXML_NATWithDHCP(t *testing.T) {
	r := NetworkResources{
		Name:      "dhcp-net",
		CIDR:      "192.168.200.0/24",
		Mode:      "nat",
		DHCP:      true,
		DHCPStart: "192.168.200.100",
		DHCPEnd:   "192.168.200.200",
	}

	xml, err := r.BuildNetworkXML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(xml, "<dhcp") {
		t.Error("XML missing DHCP element")
	}
	if !strings.Contains(xml, `start="192.168.200.100"`) {
		t.Error("XML missing DHCP start range")
	}
	if !strings.Contains(xml, `end="192.168.200.200"`) {
		t.Error("XML missing DHCP end range")
	}
}

func TestBuildNetworkXML_NATWithoutDHCP(t *testing.T) {
	r := NetworkResources{
		Name: "nodhcp-net",
		CIDR: "192.168.200.0/24",
		Mode: "nat",
		DHCP: false,
	}

	xml, err := r.BuildNetworkXML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(xml, "<dhcp") {
		t.Error("XML should not have DHCP when disabled")
	}
}

func TestBuildNetworkXML_WithDevice(t *testing.T) {
	r := NetworkResources{
		Name:   "dev-net",
		CIDR:   "172.16.0.0/24",
		Mode:   "nat",
		Device: "eth0",
	}

	xml, err := r.BuildNetworkXML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(xml, `dev="eth0"`) {
		t.Error("XML missing host device")
	}
}

func TestBuildNetworkXML_InvalidCIDR(t *testing.T) {
	r := NetworkResources{
		Name: "bad-net",
		CIDR: "not-a-cidr",
		Mode: "nat",
	}

	_, err := r.BuildNetworkXML()
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

func TestBuildNetworkXML_CIDRPrefixCalculation(t *testing.T) {
	tests := []struct {
		cidr           string
		expectedIP     string
		expectedPrefix string
	}{
		{"192.168.1.0/24", "192.168.1.0", "24"},
		{"10.0.0.0/8", "10.0.0.0", "8"},
		{"172.16.0.0/16", "172.16.0.0", "16"},
		{"10.10.10.0/28", "10.10.10.0", "28"},
	}

	for _, tc := range tests {
		t.Run(tc.cidr, func(t *testing.T) {
			r := NetworkResources{
				Name: "test",
				CIDR: tc.cidr,
				Mode: "nat",
			}

			xml, err := r.BuildNetworkXML()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(xml, `address="`+tc.expectedIP+`"`) {
				t.Errorf("expected IP %s in XML", tc.expectedIP)
			}
			if !strings.Contains(xml, `prefix="`+tc.expectedPrefix+`"`) {
				t.Errorf("expected prefix %s in XML", tc.expectedPrefix)
			}
		})
	}
}

func TestBuildNetworkXML_SameCIDRDifferentNames(t *testing.T) {
	r1 := NetworkResources{Name: "net-a", CIDR: "192.168.100.0/24", Mode: "isolated"}
	r2 := NetworkResources{Name: "net-b", CIDR: "192.168.100.0/24", Mode: "isolated"}

	xml1, _ := r1.BuildNetworkXML()
	xml2, _ := r2.BuildNetworkXML()

	// Same CIDR, different names should produce different but valid XML
	if strings.Contains(xml1, "<name>net-a</name>") == false {
		t.Error("xml1 missing name net-a")
	}
	if strings.Contains(xml2, "<name>net-b</name>") == false {
		t.Error("xml2 missing name net-b")
	}
	if strings.Contains(xml1, "virbr-net-a") == false {
		t.Error("xml1 missing bridge virbr-net-a")
	}
	if strings.Contains(xml2, "virbr-net-b") == false {
		t.Error("xml2 missing bridge virbr-net-b")
	}
}

func TestNetworkResourcesDefaults(t *testing.T) {
	r := NetworkResources{}
	r.NetworkResourcesDefaults()

	if r.Mode != "nat" {
		t.Errorf("expected default mode nat, got %s", r.Mode)
	}
}

func TestNetworkResourcesDefaults_PreservesExistingMode(t *testing.T) {
	r := NetworkResources{Mode: "route"}
	r.NetworkResourcesDefaults()

	if r.Mode != "route" {
		t.Errorf("expected mode route preserved, got %s", r.Mode)
	}
}

func TestParseCIDR(t *testing.T) {
	ip, prefix, err := parseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "192.168.1.0" {
		t.Errorf("expected ip 192.168.1.0, got %s", ip)
	}
	if prefix != 24 {
		t.Errorf("expected prefix 24, got %d", prefix)
	}
}

func TestParseCIDR_Invalid(t *testing.T) {
	_, _, err := parseCIDR("invalid")
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

func TestPrefixToNetmask(t *testing.T) {
	tests := []struct {
		prefix   int
		expected string
	}{
		{24, "255.255.255.0"},
		{8, "255.0.0.0"},
		{16, "255.255.0.0"},
		{28, "255.255.255.240"},
		{32, "255.255.255.255"},
	}

	for _, tc := range tests {
		result := prefixToNetmask(tc.prefix)
		if result != tc.expected {
			t.Errorf("prefix %d: expected %s, got %s", tc.prefix, tc.expected, result)
		}
	}
}
