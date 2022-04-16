package utils

import (
	"fmt"
	"net"
	"time"

	"github.com/digitalocean/go-libvirt"
)

var (
	Green = "\033[4;32m"
	Clear = "\033[0m"
)

type DomainInstance struct {
	Socket string
}

func (d *DomainInstance) Dial() (net.Conn, error) {
	c, err := net.DialTimeout("unix", d.Socket, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	return c, nil
}

func InitLib() (*libvirt.Libvirt, error) {
	l := libvirt.NewWithDialer(&DomainInstance{
		Socket: "/var/run/libvirt/libvirt-sock",
	})

	if err := l.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	return l, nil
}
