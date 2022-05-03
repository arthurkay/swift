# Swift Getting Started

A swift way of provisioning infrastructure in seconds

To get started with this tool, make sure you have the following tools on your machine:

***Libvirt*** installed and the livbirt daemon running

***qemu-img*** Client installed: Used for creating QCOW2 disks for system

***genisoimage*** Used to generate cidata seed images for cloud init

***cloud-init*** Used to configure cloud images in seconds

The user using the swift software to provision vm instances hould be part of the libvirt group.

```bash
sudo usermod -aG libvirt <user>
```

This software needs to create ISO9660 images, and as such, genisoimage or mkisofs is needed.

On debian:

```bash
sudo apt install genisoimage
```

On opensuse:

```bash
sudo zypper install mkisofs
```

The software stores its vm images and configs in the directory, /var/swift.

Start by creating this directory i.e (/var/swift) and give it the permission 0775 and change ownership to the user running the software.

>E.G:

```bash
sudo mkdir /var/swift
sudo chmod -R 0775 /var/swift
sudo chown -R <user>:<user> /var/swift
```

Happy Hacking