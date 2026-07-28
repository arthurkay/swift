package config

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateMAC generates a random MAC address with a locally administered,
// unicast prefix (02:xx:xx:xx:xx:xx).
func GenerateMAC() (string, error) {
	mac := make([]byte, 6)
	for i := range mac {
		n, err := rand.Int(rand.Reader, big.NewInt(256))
		if err != nil {
			return "", fmt.Errorf("generate MAC: %w", err)
		}
		mac[i] = byte(n.Int64())
	}
	// Set locally administered, unicast bits
	mac[0] = (mac[0] | 0x02) & 0xFE
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]), nil
}
