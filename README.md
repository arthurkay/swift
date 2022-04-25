# Swift

A swift way of provisioning infrastructure in seconds

To get started with this tool, make sure you have the following tools on your machine:

***Libvirt*** installed and the livbirt daemon running

***qemu-img*** Client installed: Used for creating QCOW2 disks for system

***genisoimage*** Used to generatte cidata seed images for cloud init

***cloud-init*** Used to configure cloud images in seconds

The user using the swift software to provision vm instances hould be part of the libvirt group.

```bash
sudo usermod -aG libvirt <user>
```