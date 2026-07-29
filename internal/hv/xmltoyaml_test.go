package hv

import (
	"strings"
	"testing"
)

func TestXMLToYAML_SimpleDomain(t *testing.T) {
	input := `<?xml version="1.0"?>
<domain type="kvm">
  <name>test-vm</name>
  <memory unit="KiB">524288</memory>
  <vcpu placement="static">1</vcpu>
</domain>`

	yamlStr, err := XMLToYAML(input)
	if err != nil {
		t.Fatalf("XMLToYAML failed: %v", err)
	}

	if !strings.Contains(yamlStr, "domain:") {
		t.Error("expected 'domain:' key in output")
	}
	if !strings.Contains(yamlStr, "name: test-vm") {
		t.Error("expected 'name: test-vm'")
	}
	if !strings.Contains(yamlStr, "'@unit': KiB") {
		t.Errorf("expected '@unit: KiB' attribute, got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "_: \"524288\"") {
		t.Errorf("expected '_: \"524288\"' text content, got:\n%s", yamlStr)
	}
}

func TestXMLToYAML_AttrAndContent(t *testing.T) {
	input := `<memory unit="KiB">524288</memory>`
	yamlStr, err := xmlToYAML(input)
	if err != nil {
		t.Fatalf("xmlToYAML failed: %v", err)
	}

	if !strings.Contains(yamlStr, "'@unit': KiB") {
		t.Errorf("expected '@unit: KiB', got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "_: \"524288\"") {
		t.Errorf("expected '_: \"524288\"', got:\n%s", yamlStr)
	}
}

func TestXMLToYAML_OnlyAttrs(t *testing.T) {
	input := `<driver name="qemu" type="qcow2"/>`
	yamlStr, err := xmlToYAML(input)
	if err != nil {
		t.Fatalf("xmlToYAML failed: %v", err)
	}

	if !strings.Contains(yamlStr, "'@name': qemu") {
		t.Errorf("expected '@name: qemu', got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "'@type': qcow2") {
		t.Errorf("expected '@type: qcow2', got:\n%s", yamlStr)
	}
}

func TestXMLToYAML_OnlyText(t *testing.T) {
	input := `<name>myvm</name>`
	yamlStr, err := xmlToYAML(input)
	if err != nil {
		t.Fatalf("xmlToYAML failed: %v", err)
	}

	if !strings.Contains(yamlStr, "name: myvm") {
		t.Errorf("expected 'name: myvm', got:\n%s", yamlStr)
	}
}

func TestXMLToYAML_EmptyElement(t *testing.T) {
	input := `<readonly/>`
	yamlStr, err := xmlToYAML(input)
	if err != nil {
		t.Fatalf("xmlToYAML failed: %v", err)
	}

	if !strings.Contains(yamlStr, "readonly: null") {
		t.Errorf("expected 'readonly: null', got:\n%s", yamlStr)
	}
}

func TestXMLToYAML_NestedChildren(t *testing.T) {
	input := `<disk type="file" device="disk">
  <driver name="qemu" type="qcow2"/>
  <source file="/path/to/disk.qcow2"/>
  <target dev="vda" bus="virtio"/>
</disk>`

	yamlStr, err := xmlToYAML(input)
	if err != nil {
		t.Fatalf("xmlToYAML failed: %v", err)
	}

	if !strings.Contains(yamlStr, "'@type': file") {
		t.Errorf("expected '@type: file', got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "driver:") {
		t.Errorf("expected 'driver:' child, got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "'@name': qemu") {
		t.Errorf("expected '@name: qemu', got:\n%s", yamlStr)
	}
}

func TestXMLToYAML_MultipleSameChildren(t *testing.T) {
	input := `<devices>
  <disk type="file"><target dev="vda"/></disk>
  <disk type="file"><target dev="vdb"/></disk>
</devices>`

	yamlStr, err := xmlToYAML(input)
	if err != nil {
		t.Fatalf("xmlToYAML failed: %v", err)
	}

	// disk should appear as a sequence (list)
	if !strings.Contains(yamlStr, "disk:") {
		t.Errorf("expected 'disk:' key, got:\n%s", yamlStr)
	}
	if strings.Count(yamlStr, "'@dev': vda") != 1 {
		t.Errorf("expected exactly one '@dev: vda', got:\n%s", yamlStr)
	}
	if strings.Count(yamlStr, "'@dev': vdb") != 1 {
		t.Errorf("expected exactly one '@dev: vdb', got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "- ") {
		t.Errorf("expected YAML sequence items (- ), got:\n%s", yamlStr)
	}
}

func TestYAMLToXML_SimpleElement(t *testing.T) {
	input := `name: myvm`
	xmlStr, err := yamlToXML(input)
	if err != nil {
		t.Fatalf("yamlToXML failed: %v", err)
	}

	if !strings.Contains(xmlStr, "<name>myvm</name>") {
		t.Errorf("expected '<name>myvm</name>', got:\n%s", xmlStr)
	}
}

func TestYAMLToXML_AttrsAndContent(t *testing.T) {
	input := `memory:
  "@unit": KiB
  _: "524288"`
	xmlStr, err := yamlToXML(input)
	if err != nil {
		t.Fatalf("yamlToXML failed: %v", err)
	}

	if !strings.Contains(xmlStr, "unit=\"KiB\"") {
		t.Errorf("expected 'unit=\"KiB\"', got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, ">524288<") {
		t.Errorf("expected '>524288<', got:\n%s", xmlStr)
	}
}

func TestYAMLToXML_OnlyAttrs(t *testing.T) {
	input := `driver:
  "@name": qemu
  "@type": qcow2`
	xmlStr, err := yamlToXML(input)
	if err != nil {
		t.Fatalf("yamlToXML failed: %v", err)
	}

	if !strings.Contains(xmlStr, "name=\"qemu\"") {
		t.Errorf("expected 'name=\"qemu\"', got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, "type=\"qcow2\"") {
		t.Errorf("expected 'type=\"qcow2\"', got:\n%s", xmlStr)
	}
}

func TestYAMLToXML_EmptyElement(t *testing.T) {
	input := `readonly: null`
	xmlStr, err := yamlToXML(input)
	if err != nil {
		t.Fatalf("yamlToXML failed: %v", err)
	}

	if !strings.Contains(xmlStr, "<readonly/>") {
		t.Errorf("expected '<readonly/>', got:\n%s", xmlStr)
	}
}

func TestRoundTrip_SimpleDomain(t *testing.T) {
	input := `<?xml version="1.0"?>
<domain type="kvm">
  <name>test-vm</name>
  <memory unit="KiB">524288</memory>
  <vcpu placement="static">1</vcpu>
  <os>
    <type arch="x86_64">hvm</type>
    <boot dev="hd"/>
  </os>
  <devices>
    <disk type="file" device="disk">
      <driver name="qemu" type="qcow2"/>
      <source file="/var/swift/test-vm/test-vm.qcow2"/>
      <target dev="vda" bus="virtio"/>
    </disk>
    <graphics type="spice" autoport="yes"/>
  </devices>
</domain>`

	yamlStr, err := XMLToYAML(input)
	if err != nil {
		t.Fatalf("XMLToYAML failed: %v", err)
	}

	xmlStr, err := YAMLToXML(yamlStr)
	if err != nil {
		t.Fatalf("YAMLToXML failed: %v", err)
	}

	// Verify key elements are preserved
	checks := []string{
		`type="kvm"`,
		`<name>test-vm</name>`,
		`unit="KiB"`,
		`>524288<`,
		`arch="x86_64"`,
		`<boot dev="hd"/>`,
		`name="qemu"`,
		`type="qcow2"`,
		`file="/var/swift/test-vm/test-vm.qcow2"`,
		`dev="vda"`,
		`bus="virtio"`,
		`type="spice"`,
		`autoport="yes"`,
	}

	for _, check := range checks {
		if !strings.Contains(xmlStr, check) {
			t.Errorf("round-trip lost %q, got:\n%s", check, xmlStr)
		}
	}
}

func TestRoundTrip_Network(t *testing.T) {
	input := `<?xml version="1.0"?>
<network>
  <name>mynet</name>
  <forward mode="nat"/>
  <bridge name="virbr1" stp="on" delay="0"/>
  <ip address="192.168.100.1" netmask="255.255.255.0">
    <dhcp>
      <range start="192.168.100.128" end="192.168.100.254"/>
    </dhcp>
  </ip>
</network>`

	yamlStr, err := XMLToYAML(input)
	if err != nil {
		t.Fatalf("XMLToYAML failed: %v", err)
	}

	xmlStr, err := YAMLToXML(yamlStr)
	if err != nil {
		t.Fatalf("YAMLToXML failed: %v", err)
	}

	checks := []string{
		`<name>mynet</name>`,
		`mode="nat"`,
		`name="virbr1"`,
		`address="192.168.100.1"`,
		`netmask="255.255.255.0"`,
		`start="192.168.100.128"`,
		`end="192.168.100.254"`,
	}

	for _, check := range checks {
		if !strings.Contains(xmlStr, check) {
			t.Errorf("round-trip lost %q, got:\n%s", check, xmlStr)
		}
	}
}

func TestIsXMLMirroringYAML(t *testing.T) {
	domainYAML := []byte(`domain:
  type: kvm
  name: myvm`)
	if !IsXMLMirroringYAML(domainYAML) {
		t.Error("expected true for domain YAML")
	}

	networkYAML := []byte(`network:
  name: mynet`)
	if !IsXMLMirroringYAML(networkYAML) {
		t.Error("expected true for network YAML")
	}

	simplifiedYAML := []byte(`name: myvm
disk: /path/to/image.qcow2
memory: 512`)
	if IsXMLMirroringYAML(simplifiedYAML) {
		t.Error("expected false for simplified YAML")
	}

	invalidYAML := []byte(`not valid: yaml: [`)
	if IsXMLMirroringYAML(invalidYAML) {
		t.Error("expected false for invalid YAML")
	}

	// String value should not match (e.g. "domain: production")
	stringValueYAML := []byte(`name: myvm
domain: production`)
	if IsXMLMirroringYAML(stringValueYAML) {
		t.Error("expected false when 'domain' value is a string, not a map")
	}

	// Empty map should still match
	emptyMapYAML := []byte(`domain: {}`)
	if !IsXMLMirroringYAML(emptyMapYAML) {
		t.Error("expected true for empty domain map")
	}
}

func TestXMLToYAML_NoProlog(t *testing.T) {
	input := `<domain type="kvm"><name>vm</name></domain>`
	yamlStr, err := xmlToYAML(input)
	if err != nil {
		t.Fatalf("xmlToYAML failed: %v", err)
	}
	if !strings.Contains(yamlStr, "name: vm") {
		t.Errorf("expected 'name: vm', got:\n%s", yamlStr)
	}
}

func TestXMLToYAML_XMLEscaping(t *testing.T) {
	input := `<desc>Tom &amp; Jerry &lt;3</desc>`
	yamlStr, err := xmlToYAML(input)
	if err != nil {
		t.Fatalf("xmlToYAML failed: %v", err)
	}

	// YAML should contain the unescaped text
	if !strings.Contains(yamlStr, "Tom & Jerry <3") {
		t.Errorf("expected unescaped text, got:\n%s", yamlStr)
	}

	// Round-trip should re-escape
	xmlStr, err := yamlToXML(yamlStr)
	if err != nil {
		t.Fatalf("yamlToXML failed: %v", err)
	}
	if !strings.Contains(xmlStr, "Tom &amp; Jerry &lt;3") {
		t.Errorf("expected re-escaped text, got:\n%s", xmlStr)
	}
}

func TestWarnMissingFiles(t *testing.T) {
	// XML with a reference to a file that definitely doesn't exist
	xmlStr := `<domain type="kvm">
  <devices>
    <disk type="file">
      <source file="/nonexistent/path/disk.qcow2"/>
    </disk>
  </devices>
</domain>`

	// Should return false (file missing) without panicking
	result := WarnMissingFiles(xmlStr)
	if result {
		t.Error("expected false for nonexistent file")
	}

	// XML with no source files should return true
	emptyXML := `<domain type="kvm"><name>vm</name></domain>`
	result = WarnMissingFiles(emptyXML)
	if !result {
		t.Error("expected true when no source files referenced")
	}
}
