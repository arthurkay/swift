package cli

import (
	"log"
	"os"
	"swift/config"
	"swift/operation"
	"swift/qemu"
	"swift/utils"

	"github.com/spf13/cobra"
)

// CreateDomain defines the cobra cmmand
func CreateDomain() *cobra.Command {
	var name, unit, arch, disk, cdRom, netType string
	var memory, cpuCount, storage uint
	cmd := &cobra.Command{
		Use:   "define",
		Short: "Define the vm instance template",
		Long:  "Define, defines the vm instance compute resources, distribution to configure this instance with",
		Run: func(cmd *cobra.Command, args []string) {
			if name == "" || disk == "" || cdRom == "" {
				cmd.Usage()
				return
			}
			dir, err := os.Getwd()
			if err != nil {
				log.Printf("[error Current Dir] %v", err)
			}
			image := qemu.NewImage(dir+"/config_data/"+name+".iso", qemu.ImageFormatQCOW2, uint64(storage*qemu.GiB))
			if erro := image.SetBackingFile(disk); erro != nil {
				log.Printf("[error] %v", erro)
				return
			}
			if err := image.Create(); err != nil {
				log.Printf("[error] %v", err)
			}
			// OOing: create seed img with genisoimage
			seedISO := config.NewSeed(cdRom, dir+"/config_data/user-data", dir+"/config_data/meta-data")
			if err := seedISO.Create(); err != nil {
				log.Printf("[error ISO Create] %v", err)
			}
			resources := config.Resource{
				Name:     name,
				Memory:   memory,
				Unit:     unit,
				CpuCount: cpuCount,
				Arch:     arch,
				BootOS:   image.Path,
				CDRom:    cdRom,
				NetType:  netType,
			}
			dom := resources.DefineDomain()
			libvirt, err := utils.InitLib()
			if err != nil {
				log.Printf("[error Define Domin XML] %v", err)
				return
			}
			defer libvirt.Disconnect()
			newDomain, err := dom.Marshal()
			if err != nil {
				log.Printf("[error Defined Domain XML] %v", err)
				return
			}
			createdDomain, er := operation.Define(newDomain, libvirt)
			if er != nil {
				log.Printf("[error Create Domain Instance] %v", er)
				return
			}
			e := libvirt.DomainCreate(createdDomain)
			if e != nil {
				log.Printf("[error Run Domain Instance] %v", e)
				return
			}
		},
		Example: `
		To create a VM, only three flags are mandatory, i.e:
		< --name | -n > < --disk -d > and < --rom, -r >

		The above translates to:

		rs -n TestVm -d /var/rs/vm/ubuntu.img -r /tmp/ubuntu.img`,
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "A name for the vm instance")
	cmd.Flags().UintVarP(&memory, "memory", "m", 1024, "The amount of Ram to allocate the vm instance")
	cmd.Flags().StringVarP(&unit, "unit", "u", "MiB", "Memory units, supported values: KiB MiB GiB")
	cmd.Flags().StringVarP(&arch, "arch", "a", "x86_64", "The vm instance system architecture")
	cmd.Flags().StringVarP(&disk, "disk", "d", "", "Location of the disk the OS to boot in")
	cmd.Flags().StringVarP(&cdRom, "rom", "r", "", "Location of the image to get OS during installation")
	cmd.Flags().UintVarP(&cpuCount, "cpu", "c", 1, "Number of CPU's to allocate to vm instance")
	cmd.Flags().StringVarP(&netType, "interface", "i", "e1000", "Network interface model")
	cmd.Flags().UintVarP(&storage, "storage", "s", 10, "Disk size in Gigabytes")
	cmd.MarkFlagRequired(name)
	cmd.MarkFlagRequired(disk)
	cmd.MarkFlagRequired(cdRom)
	return cmd
}
