package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateMAC(t *testing.T) {
	mac, err := GenerateMAC()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// MAC should be 17 chars: xx:xx:xx:xx:xx:xx
	if len(mac) != 17 {
		t.Errorf("expected MAC length 17, got %d (%s)", len(mac), mac)
	}

	// Should have colons at positions 2, 5, 8, 11, 14
	for _, pos := range []int{2, 5, 8, 11, 14} {
		if mac[pos] != ':' {
			t.Errorf("expected colon at position %d, got %c", pos, mac[pos])
		}
	}
}

func TestGenerateMAC_LocallyAdministered(t *testing.T) {
	for i := 0; i < 100; i++ {
		mac, err := GenerateMAC()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// First byte should have bit 1 set (locally administered)
		// and bit 0 clear (unicast)
		var b1 byte
		for j := 0; j < 2; j++ {
			c := mac[j]
			if c >= '0' && c <= '9' {
				b1 = b1*16 + (c - '0')
			} else if c >= 'a' && c <= 'f' {
				b1 = b1*16 + (c - 'a' + 10)
			}
		}

		if b1&0x02 == 0 {
			t.Errorf("MAC %s is not locally administered (bit 1 not set)", mac)
		}
		if b1&0x01 != 0 {
			t.Errorf("MAC %s is not unicast (bit 0 is set)", mac)
		}
	}
}

func TestGenerateMAC_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		mac, err := GenerateMAC()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if seen[mac] {
			t.Errorf("duplicate MAC generated: %s", mac)
		}
		seen[mac] = true
	}
}

func TestLoadVMConfig(t *testing.T) {
	content := `
name: test-vm
disk: /path/to/ubuntu.qcow2
memory: 2048
cpu: 4
storage: 20G
username: admin
password: secret123
ssh_key: ssh-rsa AAAA...
graphics: vnc
websocket: true
serial: true
networks:
  - name: default
    model: e1000
  - name: mynet
    model: virtio
`
	dir := t.TempDir()
	path := filepath.Join(dir, "vm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	cfg, err := LoadVMConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "test-vm" {
		t.Errorf("expected name test-vm, got %s", cfg.Name)
	}
	if cfg.Disk != "/path/to/ubuntu.qcow2" {
		t.Errorf("expected disk path, got %s", cfg.Disk)
	}
	if cfg.Memory != 2048 {
		t.Errorf("expected memory 2048, got %d", cfg.Memory)
	}
	if cfg.CPU != 4 {
		t.Errorf("expected cpu 4, got %d", cfg.CPU)
	}
	if cfg.Storage != "20G" {
		t.Errorf("expected storage 20G, got %s", cfg.Storage)
	}
	if cfg.Username != "admin" {
		t.Errorf("expected username admin, got %s", cfg.Username)
	}
	if cfg.Password != "secret123" {
		t.Errorf("expected password secret123, got %s", cfg.Password)
	}
	if cfg.SSHKey != "ssh-rsa AAAA..." {
		t.Errorf("expected ssh_key, got %s", cfg.SSHKey)
	}
	if cfg.Graphics != "vnc" {
		t.Errorf("expected graphics vnc, got %s", cfg.Graphics)
	}
	if !cfg.WebSocket {
		t.Error("expected websocket true")
	}
	if !cfg.Serial {
		t.Error("expected serial true")
	}
	if len(cfg.Networks) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(cfg.Networks))
	}
	if cfg.Networks[0].Name != "default" {
		t.Errorf("expected first network default, got %s", cfg.Networks[0].Name)
	}
	if cfg.Networks[1].Model != "virtio" {
		t.Errorf("expected second network model virtio, got %s", cfg.Networks[1].Model)
	}
}

func TestLoadVMConfig_Minimal(t *testing.T) {
	content := `
name: minimal-vm
disk: /path/to/image.qcow2
`
	dir := t.TempDir()
	path := filepath.Join(dir, "vm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	cfg, err := LoadVMConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "minimal-vm" {
		t.Errorf("expected name, got %s", cfg.Name)
	}
	if cfg.Graphics != "" {
		t.Errorf("expected empty graphics, got %s", cfg.Graphics)
	}
	if cfg.WebSocket {
		t.Error("expected websocket false")
	}
	if cfg.Serial {
		t.Error("expected serial false")
	}
	if len(cfg.Networks) != 0 {
		t.Errorf("expected 0 networks, got %d", len(cfg.Networks))
	}
}

func TestLoadVMConfig_InvalidFile(t *testing.T) {
	_, err := LoadVMConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadVMConfig_InvalidYAML(t *testing.T) {
	content := `{{{{invalid yaml`
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, err := LoadVMConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadNetworkConfig(t *testing.T) {
	content := `
name: test-net
cidr: 192.168.100.0/24
mode: nat
bridge: virbr100
device: eth1
dhcp: true
dhcp_start: 192.168.100.100
dhcp_end: 192.168.100.200
`
	dir := t.TempDir()
	path := filepath.Join(dir, "net.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	cfg, err := LoadNetworkConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "test-net" {
		t.Errorf("expected name test-net, got %s", cfg.Name)
	}
	if cfg.CIDR != "192.168.100.0/24" {
		t.Errorf("expected cidr, got %s", cfg.CIDR)
	}
	if cfg.Mode != "nat" {
		t.Errorf("expected mode nat, got %s", cfg.Mode)
	}
	if cfg.Bridge != "virbr100" {
		t.Errorf("expected bridge, got %s", cfg.Bridge)
	}
	if cfg.Device != "eth1" {
		t.Errorf("expected device, got %s", cfg.Device)
	}
	if !cfg.DHCP {
		t.Error("expected dhcp true")
	}
	if cfg.DHCPStart != "192.168.100.100" {
		t.Errorf("expected dhcp_start, got %s", cfg.DHCPStart)
	}
	if cfg.DHCPEnd != "192.168.100.200" {
		t.Errorf("expected dhcp_end, got %s", cfg.DHCPEnd)
	}
}

func TestLoadNetworkConfig_Isolated(t *testing.T) {
	content := `
name: isolated-net
cidr: 10.10.10.0/24
mode: isolated
dhcp: false
`
	dir := t.TempDir()
	path := filepath.Join(dir, "net.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	cfg, err := LoadNetworkConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Mode != "isolated" {
		t.Errorf("expected mode isolated, got %s", cfg.Mode)
	}
	if cfg.DHCP {
		t.Error("expected dhcp false")
	}
}

func TestLoadNetworkConfig_InvalidFile(t *testing.T) {
	_, err := LoadNetworkConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
