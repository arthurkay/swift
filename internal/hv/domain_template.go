package hv

import (
	"libvirt.org/go/libvirtxml"
)

// DomainResources holds the compute resources for a VM definition.
type DomainResources struct {
	Name     string
	Memory   uint
	Unit     string
	CpuCount uint
	Arch     string
	BootOS   string
	CDRom    string
	NetType  string
}

// DefineDomain constructs a libvirtxml.Domain from the given resources.
func (r DomainResources) DefineDomain() *libvirtxml.Domain {
	memUnit := r.Unit
	if memUnit == "" {
		memUnit = "MiB"
	}
	arch := r.Arch
	if arch == "" {
		arch = "x86_64"
	}
	netModel := r.NetType
	if netModel == "" {
		netModel = "e1000"
	}

	return &libvirtxml.Domain{
		Type: "kvm",
		Name: r.Name,
		Memory: &libvirtxml.DomainMemory{
			Unit: memUnit,
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
			Graphics: []libvirtxml.DomainGraphic{
				{
				Spice: &libvirtxml.DomainGraphicSpice{
					AutoPort: "on",
					Image: &libvirtxml.DomainGraphicSpiceImage{
						Compression: "on",
					},
				},
				},
			},
			Disks: []libvirtxml.DomainDisk{
				{
					Device: "disk",
					Driver: &libvirtxml.DomainDiskDriver{
						Name: "qemu",
						Type: "qcow2",
					},
					Source: &libvirtxml.DomainDiskSource{
						File: &libvirtxml.DomainDiskSourceFile{
							File: r.BootOS,
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
							File: r.CDRom,
						},
					},
					Target: &libvirtxml.DomainDiskTarget{
						Dev: "sda",
						Bus: "sata",
					},
				},
			},
			Interfaces: []libvirtxml.DomainInterface{
				{
					Model: &libvirtxml.DomainInterfaceModel{
						Type: netModel,
					},
					Source: &libvirtxml.DomainInterfaceSource{
						Network: &libvirtxml.DomainInterfaceSourceNetwork{
							Network: "default",
						},
					},
				},
			},
		},
	}
}
