package utils

import (
	"crypto/rand"
	"fmt"
)

func NewMacAddress() string {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return ""
	}
	// Local bit to make it unicast mac address
	buf[0] = (buf[0] | 2) & 0xfe
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x\n", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
}
