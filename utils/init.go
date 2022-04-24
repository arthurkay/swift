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

// DomainInstance defines the structure of the unix socket to use
// when establishing connecton to hypervisor on host
type DomainInstance struct {
	Socket string
}

// Dial makes the call to the unix socket defined in the
// DomainInstance struct
func (d *DomainInstance) Dial() (net.Conn, error) {
	c, err := net.DialTimeout("unix", d.Socket, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	return c, nil
}

// InitLib initialises connection to libvirt socket on host machine
func InitLib() (*libvirt.Libvirt, error) {
	l := libvirt.NewWithDialer(&DomainInstance{
		Socket: "/var/run/libvirt/libvirt-sock",
	})

	if err := l.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	return l, nil
}
