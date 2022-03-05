package config

import (
	"bytes"
	"fmt"
	"html/template"
	"petricoh/operation"

	"github.com/digitalocean/go-libvirt"
)

type ComputeResources struct {
	Name     string
	Memory   int32
	Vcpu     int32
	ImageIso string
	BootIso  string
	Image    string
	MacAddr  string
}

type DomainTemplate struct {
	Template string
}

func (d *DomainTemplate) Create(l *libvirt.Libvirt) {
	domain, err := operation.Define(d.Template, l)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}
	operation.StartUp(domain.Name, l)
}

func (r *ComputeResources) VirtInstall(l *libvirt.Libvirt) {
	var xml bytes.Buffer
	t, err := template.ParseFiles("config/assets/domain.xml")
	if err != nil {
		fmt.Printf("Unable to get domain template file %v", err)
		return
	}

	er := t.Execute(&xml, r)
	if er != nil {
		fmt.Printf("Unable to create XML template %v", er)
		return
	}

	domainTemplate := DomainTemplate{
		Template: xml.String(),
	}
	domainTemplate.Create(l)
}
