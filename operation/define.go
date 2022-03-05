package operation

import "github.com/digitalocean/go-libvirt"

func Define(xml string, l *libvirt.Libvirt) (domain libvirt.Domain, err error) {
	return l.DomainDefineXML(xml)
}

func Undefine(domain libvirt.Domain, l *libvirt.Libvirt) error {
	return l.DomainUndefine(domain)
}
