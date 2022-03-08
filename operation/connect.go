package operation

import (
	"fmt"
	"net"
	"time"

	"github.com/digitalocean/go-libvirt"
)

type domainInstance struct {
	Socket string
}

func (d *domainInstance) Dial() (net.Conn, error) {
	c, err := net.DialTimeout("unix", d.Socket, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	return c, nil
}

func Connect() (*libvirt.Libvirt, error) {
	l := libvirt.NewWithDialer(&domainInstance{
		Socket: "/var/run/libvirt/libvirt-sock",
	})

	if err := l.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	return l, nil
}
