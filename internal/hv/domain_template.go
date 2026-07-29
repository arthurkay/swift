package hv

import (
	"libvirt.org/go/libvirtxml"
)

// DomainResources holds the compute resources for a VM definition.
type DomainResources struct {
	Name         string
	Memory       uint
	Unit         string
	CpuCount     uint
	Arch         string
	BootOS       string
	CDRom        string
	MAC          string
	Networks     []NetworkConfig
	GraphicsType string // "spice" (default), "vnc", "none"
	WebSocket    bool   // enable WebSocket listener on graphics port
	Serial       bool   // add PTY serial console
}

// BuildDomainXML constructs a libvirtxml.Domain from the given resources.
func (r DomainResources) BuildDomainXML() *libvirtxml.Domain {
	memUnit := r.Unit
	if memUnit == "" {
		memUnit = "MiB"
	}
	arch := r.Arch
	if arch == "" {
		arch = "x86_64"
	}

	// Build network interfaces
	var ifaces []libvirtxml.DomainInterface
	networks := r.Networks
	if len(networks) == 0 {
		networks = []NetworkConfig{{Name: "default", Model: "e1000"}}
	}
	for i, net := range networks {
		model := net.Model
		if model == "" {
			model = "e1000"
		}
		iface := libvirtxml.DomainInterface{
			Model: &libvirtxml.DomainInterfaceModel{
				Type: model,
			},
			Source: &libvirtxml.DomainInterfaceSource{
				Network: &libvirtxml.DomainInterfaceSourceNetwork{
					Network: net.Name,
				},
			},
		}
		if i == 0 && r.MAC != "" {
			iface.MAC = &libvirtxml.DomainInterfaceMAC{
				Address: r.MAC,
			}
		}
		ifaces = append(ifaces, iface)
	}

	// Build graphics device
	var graphics []libvirtxml.DomainGraphic
	graphicsType := r.GraphicsType
	if graphicsType == "" {
		graphicsType = "spice"
	}
	switch graphicsType {
	case "vnc":
		vnc := &libvirtxml.DomainGraphicVNC{
			AutoPort: "yes",
			Listen:   "0.0.0.0",
		}
		if r.WebSocket {
			vnc.WebSocket = -1
		}
		graphics = append(graphics, libvirtxml.DomainGraphic{VNC: vnc})
	case "spice":
		spice := &libvirtxml.DomainGraphicSpice{
			AutoPort: "yes",
			Listen:   "0.0.0.0",
		}
		graphics = append(graphics, libvirtxml.DomainGraphic{Spice: spice})
	case "none":
		// No graphics device
	}

	// Build serial and console devices
	var serials []libvirtxml.DomainSerial
	var consoles []libvirtxml.DomainConsole
	if r.Serial {
		serials = append(serials, libvirtxml.DomainSerial{
			Target: &libvirtxml.DomainSerialTarget{
				Port: uintPtr(0),
			},
		})
		consoles = append(consoles, libvirtxml.DomainConsole{
			Target: &libvirtxml.DomainConsoleTarget{
				Type: "serial",
				Port: uintPtr(0),
			},
		})
	}

	dom := &libvirtxml.Domain{
		Type: "kvm",
		Name: r.Name,
		Memory: &libvirtxml.DomainMemory{
			Unit:  memUnit,
			Value: uint(r.Memory),
		},
		VCPU: &libvirtxml.DomainVCPU{
			Value: uint(r.CpuCount),
		},
		OS: &libvirtxml.DomainOS{
			Type: &libvirtxml.DomainOSType{
				Arch:    arch,
				Machine: "pc",
				Type:    "hvm",
			},
			BootDevices: []libvirtxml.DomainBootDevice{
				{Dev: "hd"},
				{Dev: "cdrom"},
			},
		},
		Devices: &libvirtxml.DomainDeviceList{
			Graphics:   graphics,
			Disks:      buildDisks(r.BootOS, r.CDRom),
			Interfaces: ifaces,
			Serials:    serials,
			Consoles:   consoles,
		},
	}
	return dom
}

// buildDisks returns the standard disk configuration.
func buildDisks(bootOS, cdrom string) []libvirtxml.DomainDisk {
	return []libvirtxml.DomainDisk{
		{
			Device: "disk",
			Driver: &libvirtxml.DomainDiskDriver{
				Name: "qemu",
				Type: "qcow2",
			},
			Source: &libvirtxml.DomainDiskSource{
				File: &libvirtxml.DomainDiskSourceFile{
					File: bootOS,
				},
			},
			Target: &libvirtxml.DomainDiskTarget{
				Dev: "vda",
				Bus: "virtio",
			},
		},
		{
			Device: "cdrom",
			Driver: &libvirtxml.DomainDiskDriver{
				Name: "qemu",
				Type: "raw",
			},
			Source: &libvirtxml.DomainDiskSource{
				File: &libvirtxml.DomainDiskSourceFile{
					File: cdrom,
				},
			},
			Target: &libvirtxml.DomainDiskTarget{
				Dev: "sda",
				Bus: "sata",
			},
		},
	}
}

func uintPtr(v uint) *uint {
	return &v
}
