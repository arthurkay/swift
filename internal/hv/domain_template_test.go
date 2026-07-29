package hv

import (
	"strings"
	"testing"
)

func TestBuildDomainXML_DefaultSpice(t *testing.T) {
	r := DomainResources{
		Name:     "test-vm",
		Memory:   2048,
		Unit:     "MiB",
		CpuCount: 2,
		BootOS:   "/var/swift/test-vm/test-vm.qcow2",
		CDRom:    "/var/swift/test-vm/cidata.iso",
		MAC:      "02:ab:cd:ef:01:23",
	}

	dom := r.BuildDomainXML()

	if dom.Name != "test-vm" {
		t.Errorf("expected name test-vm, got %s", dom.Name)
	}
	if dom.Type != "kvm" {
		t.Errorf("expected type kvm, got %s", dom.Type)
	}
	if dom.Memory.Value != 2048 {
		t.Errorf("expected memory 2048, got %d", dom.Memory.Value)
	}
	if dom.Memory.Unit != "MiB" {
		t.Errorf("expected unit MiB, got %s", dom.Memory.Unit)
	}
	if dom.VCPU.Value != 2 {
		t.Errorf("expected vcpu 2, got %d", dom.VCPU.Value)
	}

	// Default should be SPICE
	if len(dom.Devices.Graphics) != 1 {
		t.Fatalf("expected 1 graphics device, got %d", len(dom.Devices.Graphics))
	}
	gfx := dom.Devices.Graphics[0]
	if gfx.Spice == nil {
		t.Fatal("expected SPICE graphics, got nil")
	}
	if gfx.Spice.AutoPort != "yes" {
		t.Errorf("expected autoport yes, got %s", gfx.Spice.AutoPort)
	}
	if gfx.VNC != nil {
		t.Error("expected no VNC graphics")
	}
}

func TestBuildDomainXML_SpiceGraphics(t *testing.T) {
	r := DomainResources{
		Name:         "spice-vm",
		Memory:       1024,
		CpuCount:     1,
		BootOS:       "/test.qcow2",
		CDRom:        "/test.iso",
		GraphicsType: "spice",
	}

	dom := r.BuildDomainXML()

	if len(dom.Devices.Graphics) != 1 {
		t.Fatalf("expected 1 graphics, got %d", len(dom.Devices.Graphics))
	}
	if dom.Devices.Graphics[0].Spice == nil {
		t.Fatal("expected SPICE graphics")
	}
}

func TestBuildDomainXML_VncGraphics(t *testing.T) {
	r := DomainResources{
		Name:         "vnc-vm",
		Memory:       1024,
		CpuCount:     1,
		BootOS:       "/test.qcow2",
		CDRom:        "/test.iso",
		GraphicsType: "vnc",
	}

	dom := r.BuildDomainXML()

	if len(dom.Devices.Graphics) != 1 {
		t.Fatalf("expected 1 graphics, got %d", len(dom.Devices.Graphics))
	}
	gfx := dom.Devices.Graphics[0]
	if gfx.VNC == nil {
		t.Fatal("expected VNC graphics, got nil")
	}
	if gfx.Spice != nil {
		t.Error("expected no SPICE graphics")
	}
	if gfx.VNC.AutoPort != "yes" {
		t.Errorf("expected autoport yes, got %s", gfx.VNC.AutoPort)
	}
	if gfx.VNC.Listen != "0.0.0.0" {
		t.Errorf("expected listen 0.0.0.0, got %s", gfx.VNC.Listen)
	}
	// WebSocket should NOT be set when not requested
	if gfx.VNC.WebSocket != 0 {
		t.Errorf("expected websocket 0 (disabled), got %d", gfx.VNC.WebSocket)
	}
}

func TestBuildDomainXML_VncWithWebSocket(t *testing.T) {
	r := DomainResources{
		Name:         "vnc-ws-vm",
		Memory:       1024,
		CpuCount:     1,
		BootOS:       "/test.qcow2",
		CDRom:        "/test.iso",
		GraphicsType: "vnc",
		WebSocket:    true,
	}

	dom := r.BuildDomainXML()

	gfx := dom.Devices.Graphics[0]
	if gfx.VNC == nil {
		t.Fatal("expected VNC graphics")
	}
	if gfx.VNC.WebSocket != -1 {
		t.Errorf("expected websocket -1 (auto), got %d", gfx.VNC.WebSocket)
	}
}

func TestBuildDomainXML_NoGraphics(t *testing.T) {
	r := DomainResources{
		Name:         "headless-vm",
		Memory:       512,
		CpuCount:     1,
		BootOS:       "/test.qcow2",
		CDRom:        "/test.iso",
		GraphicsType: "none",
	}

	dom := r.BuildDomainXML()

	if len(dom.Devices.Graphics) != 0 {
		t.Errorf("expected 0 graphics devices, got %d", len(dom.Devices.Graphics))
	}
}

func TestBuildDomainXML_EmptyGraphicsDefaultsToSpice(t *testing.T) {
	r := DomainResources{
		Name:     "default-vm",
		Memory:   512,
		CpuCount: 1,
		BootOS:   "/test.qcow2",
		CDRom:    "/test.iso",
	}

	dom := r.BuildDomainXML()

	if len(dom.Devices.Graphics) != 1 {
		t.Fatalf("expected 1 graphics, got %d", len(dom.Devices.Graphics))
	}
	if dom.Devices.Graphics[0].Spice == nil {
		t.Fatal("expected default SPICE graphics")
	}
}

func TestBuildDomainXML_SerialConsole(t *testing.T) {
	r := DomainResources{
		Name:     "serial-vm",
		Memory:   512,
		CpuCount: 1,
		BootOS:   "/test.qcow2",
		CDRom:    "/test.iso",
		Serial:   true,
	}

	dom := r.BuildDomainXML()

	if len(dom.Devices.Serials) != 1 {
		t.Fatalf("expected 1 serial, got %d", len(dom.Devices.Serials))
	}
	serial := dom.Devices.Serials[0]
	if serial.Target == nil {
		t.Fatal("expected serial target")
	}
	if serial.Target.Port == nil || *serial.Target.Port != 0 {
		t.Error("expected serial target port 0")
	}

	if len(dom.Devices.Consoles) != 1 {
		t.Fatalf("expected 1 console, got %d", len(dom.Devices.Consoles))
	}
	console := dom.Devices.Consoles[0]
	if console.Target == nil {
		t.Fatal("expected console target")
	}
	if console.Target.Type != "serial" {
		t.Errorf("expected console target type serial, got %s", console.Target.Type)
	}
}

func TestBuildDomainXML_NoSerial(t *testing.T) {
	r := DomainResources{
		Name:     "noserial-vm",
		Memory:   512,
		CpuCount: 1,
		BootOS:   "/test.qcow2",
		CDRom:    "/test.iso",
		Serial:   false,
	}

	dom := r.BuildDomainXML()

	if len(dom.Devices.Serials) != 0 {
		t.Errorf("expected 0 serials, got %d", len(dom.Devices.Serials))
	}
	if len(dom.Devices.Consoles) != 0 {
		t.Errorf("expected 0 consoles, got %d", len(dom.Devices.Consoles))
	}
}

func TestBuildDomainXML_AllOptions(t *testing.T) {
	r := DomainResources{
		Name:         "full-vm",
		Memory:       4096,
		CpuCount:     4,
		BootOS:       "/var/swift/full-vm/full-vm.qcow2",
		CDRom:        "/var/swift/full-vm/cidata.iso",
		MAC:          "02:aa:bb:cc:dd:ee",
		GraphicsType: "vnc",
		WebSocket:    true,
		Serial:       true,
		Networks: []NetworkConfig{
			{Name: "default", Model: "e1000"},
			{Name: "mynet", Model: "virtio"},
		},
	}

	dom := r.BuildDomainXML()

	// Verify everything
	if dom.Name != "full-vm" {
		t.Errorf("expected name full-vm, got %s", dom.Name)
	}
	if dom.Memory.Value != 4096 {
		t.Errorf("expected memory 4096, got %d", dom.Memory.Value)
	}
	if dom.VCPU.Value != 4 {
		t.Errorf("expected vcpu 4, got %d", dom.VCPU.Value)
	}

	// VNC + WebSocket
	if dom.Devices.Graphics[0].VNC == nil {
		t.Fatal("expected VNC graphics")
	}
	if dom.Devices.Graphics[0].VNC.WebSocket != -1 {
		t.Error("expected websocket enabled")
	}

	// Serial
	if len(dom.Devices.Serials) != 1 {
		t.Error("expected serial console")
	}

	// Two NICs
	if len(dom.Devices.Interfaces) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(dom.Devices.Interfaces))
	}
	if dom.Devices.Interfaces[0].Source.Network.Network != "default" {
		t.Errorf("expected first NIC on default, got %s", dom.Devices.Interfaces[0].Source.Network.Network)
	}
	if dom.Devices.Interfaces[1].Source.Network.Network != "mynet" {
		t.Errorf("expected second NIC on mynet, got %s", dom.Devices.Interfaces[1].Source.Network.Network)
	}
	if dom.Devices.Interfaces[0].MAC == nil {
		t.Error("expected MAC on first interface")
	}
	if dom.Devices.Interfaces[1].MAC != nil {
		t.Error("expected no MAC on second interface")
	}

	// Disks
	if len(dom.Devices.Disks) != 2 {
		t.Fatalf("expected 2 disks, got %d", len(dom.Devices.Disks))
	}
}

func TestBuildDomainXML_DefaultNetworkFallback(t *testing.T) {
	r := DomainResources{
		Name:     "default-net-vm",
		Memory:   512,
		CpuCount: 1,
		BootOS:   "/test.qcow2",
		CDRom:    "/test.iso",
		Networks: nil,
	}

	dom := r.BuildDomainXML()

	// Should fall back to default network
	if len(dom.Devices.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(dom.Devices.Interfaces))
	}
	if dom.Devices.Interfaces[0].Source.Network.Network != "default" {
		t.Errorf("expected default network, got %s", dom.Devices.Interfaces[0].Source.Network.Network)
	}
}

func TestBuildDomainXML_DefaultArch(t *testing.T) {
	r := DomainResources{
		Name:     "arch-vm",
		Memory:   512,
		CpuCount: 1,
		BootOS:   "/test.qcow2",
		CDRom:    "/test.iso",
		Arch:     "",
	}

	dom := r.BuildDomainXML()

	if dom.OS.Type.Arch != "x86_64" {
		t.Errorf("expected default arch x86_64, got %s", dom.OS.Type.Arch)
	}
}

func TestBuildDomainXML_CustomArch(t *testing.T) {
	r := DomainResources{
		Name:     "aarch64-vm",
		Memory:   512,
		CpuCount: 1,
		BootOS:   "/test.qcow2",
		CDRom:    "/test.iso",
		Arch:     "aarch64",
	}

	dom := r.BuildDomainXML()

	if dom.OS.Type.Arch != "aarch64" {
		t.Errorf("expected arch aarch64, got %s", dom.OS.Type.Arch)
	}
}

func TestBuildDomainXML_DiskConfiguration(t *testing.T) {
	r := DomainResources{
		Name:     "disk-vm",
		Memory:   512,
		CpuCount: 1,
		BootOS:   "/path/to/disk.qcow2",
		CDRom:    "/path/to/cidata.iso",
	}

	dom := r.BuildDomainXML()

	if len(dom.Devices.Disks) != 2 {
		t.Fatalf("expected 2 disks, got %d", len(dom.Devices.Disks))
	}

	// Boot disk
	disk := dom.Devices.Disks[0]
	if disk.Device != "disk" {
		t.Errorf("expected device disk, got %s", disk.Device)
	}
	if disk.Driver.Name != "qemu" || disk.Driver.Type != "qcow2" {
		t.Errorf("expected qcow2 driver, got %s/%s", disk.Driver.Name, disk.Driver.Type)
	}
	if disk.Source.File.File != "/path/to/disk.qcow2" {
		t.Errorf("expected boot disk path, got %s", disk.Source.File.File)
	}
	if disk.Target.Bus != "virtio" {
		t.Errorf("expected virtio bus, got %s", disk.Target.Bus)
	}

	// CD-ROM
	cdrom := dom.Devices.Disks[1]
	if cdrom.Device != "cdrom" {
		t.Errorf("expected device cdrom, got %s", cdrom.Device)
	}
	if cdrom.Driver.Type != "raw" {
		t.Errorf("expected raw type for cdrom, got %s", cdrom.Driver.Type)
	}
	if cdrom.Target.Bus != "sata" {
		t.Errorf("expected sata bus for cdrom, got %s", cdrom.Target.Bus)
	}
}

func TestBuildDomainXML_BootDevices(t *testing.T) {
	r := DomainResources{
		Name:     "boot-vm",
		Memory:   512,
		CpuCount: 1,
		BootOS:   "/test.qcow2",
		CDRom:    "/test.iso",
	}

	dom := r.BuildDomainXML()

	if len(dom.OS.BootDevices) != 2 {
		t.Fatalf("expected 2 boot devices, got %d", len(dom.OS.BootDevices))
	}
	if dom.OS.BootDevices[0].Dev != "hd" {
		t.Errorf("expected first boot device hd, got %s", dom.OS.BootDevices[0].Dev)
	}
	if dom.OS.BootDevices[1].Dev != "cdrom" {
		t.Errorf("expected second boot device cdrom, got %s", dom.OS.BootDevices[1].Dev)
	}
}

func TestBuildDomainXML_MarshalProducesValidXML(t *testing.T) {
	r := DomainResources{
		Name:         "xml-test-vm",
		Memory:       2048,
		CpuCount:     2,
		BootOS:       "/test.qcow2",
		CDRom:        "/test.iso",
		MAC:          "02:aa:bb:cc:dd:ee",
		GraphicsType: "vnc",
		WebSocket:    true,
		Serial:       true,
		Networks:     []NetworkConfig{{Name: "default", Model: "e1000"}},
	}

	dom := r.BuildDomainXML()
	xml, err := dom.Marshal()
	if err != nil {
		t.Fatalf("failed to marshal domain XML: %v", err)
	}

	if !strings.Contains(xml, "<name>xml-test-vm</name>") {
		t.Error("XML missing VM name")
	}
	if !strings.Contains(xml, `type="kvm"`) {
		t.Error("XML missing kvm type")
	}
	if !strings.Contains(xml, `type="vnc"`) {
		t.Error("XML missing vnc graphics type")
	}
	if !strings.Contains(xml, `websocket="-1"`) {
		t.Error("XML missing websocket attribute")
	}
	if !strings.Contains(xml, "<serial") {
		t.Error("XML missing serial element")
	}
	if !strings.Contains(xml, "<console") {
		t.Error("XML missing console element")
	}
	if !strings.Contains(xml, `type="serial"`) {
		t.Error("XML missing serial console target type")
	}
}
