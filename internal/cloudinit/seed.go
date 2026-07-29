package cloudinit

import (
	"fmt"
	"os"
	"os/exec"
	"swift/pkg/errors"
)

// Seed represents a cloud-init seed ISO generator.
type Seed struct {
	OutputISO string
	UserData  string
	MetaData  string
}

// NewSeed creates a new Seed for ISO generation.
func NewSeed(iso, userData, metaData string) Seed {
	return Seed{
		OutputISO: iso,
		UserData:  userData,
		MetaData:  metaData,
	}
}

// Create generates the cloud-init seed ISO using genisoimage or mkisofs.
func (s Seed) Create() error {
	os.Remove(s.OutputISO)
	args := []string{
		"-output", s.OutputISO,
		"-V", "cidata",
		"-r", "-J",
		s.UserData,
	}
	if s.MetaData != "" {
		args = append(args, s.MetaData)
	}

	// Try genisoimage first, then mkisofs
	for _, tool := range []string{"genisoimage", "mkisofs"} {
		if !errors.CommandExists(tool) {
			continue
		}
		cmd := exec.Command(tool, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("'%s' output: %s", tool, errors.OneLine(out))
		}
		return nil
	}
	return fmt.Errorf("ISO creation tool not found, install mkisofs or genisoimage")
}
