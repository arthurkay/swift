package config

import (
	libvirtxml "libvirt.org/libvirt-go-xml"
)

type Resource struct {
	Name     string
	Memory   uint
	Unit     string
	CpuCount uint
	Arch     string
	BootOS   string
	CDRom    string
	NetType  string
}

func (r *Resource) DefineDomain() *libvirtxml.Domain {
	return &libvirtxml.Domain{
		Type: "kvm",
		Name: r.Name,
		Memory: &libvirtxml.DomainMemory{
			Value: r.Memory,
			Unit:  r.Unit,
		},
		VCPU: &libvirtxml.DomainVCPU{
			Value: r.CpuCount,
		},
		OS: &libvirtxml.DomainOS{
			BootMenu: &libvirtxml.DomainBootMenu{
				Enable:  "yes",
				Timeout: "15s",
			},
			BootDevices: []libvirtxml.DomainBootDevice{
				{
					Dev: "hd",
				},
				{
					Dev: "cdrom",
				},
			},
			Type: &libvirtxml.DomainOSType{
				Arch: r.Arch,
				Type: "hvm",
			},
		},
		Devices: &libvirtxml.DomainDeviceList{
			Graphics: []libvirtxml.DomainGraphic{
				{
					Spice: &libvirtxml.DomainGraphicSpice{
						AutoPort: "yes",
						Image: &libvirtxml.DomainGraphicSpiceImage{
							Compression: "off",
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
					Target: &libvirtxml.DomainDiskTarget{
						Dev: "hda",
						Bus: "ide",
					},
					Source: &libvirtxml.DomainDiskSource{
						File: &libvirtxml.DomainDiskSourceFile{
							File: r.BootOS, //"/home/arthur/Documents/DEV/RnD/cloudinit-rnd/cloud-img.qcow2",
						},
					},
				},
				{
					Device: "cdrom",
					Target: &libvirtxml.DomainDiskTarget{
						Dev: "hdb",
						Bus: "ide",
					},
					Source: &libvirtxml.DomainDiskSource{
						File: &libvirtxml.DomainDiskSourceFile{
							File: r.CDRom, //"/home/arthur/Documents/DEV/RnD/cloudinit-rnd/ubuntu-seed.img",
						},
					},
				},
			},
			Interfaces: []libvirtxml.DomainInterface{
				{
					Model: &libvirtxml.DomainInterfaceModel{
						Type: r.NetType,
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
