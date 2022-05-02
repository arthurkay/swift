package config

import (
	"fmt"
	"os/exec"
	"swift/utils"
)

type seed struct {
	OutputISO string
	UserData  string
	MetaData  string
}

// NewSeed returns a struct of the provided values for the seed
// to be created
func NewSeed(iso, userData, metaData string) seed {
	return seed{
		OutputISO: iso,
		UserData:  userData,
		MetaData:  metaData,
	}
}

// Create generates the seed iso tp use for vm provisioning
func (s seed) Create() error {
	args := []string{"-output", s.OutputISO}
	args = append(args, "-V")
	args = append(args, "cidata")
	args = append(args, "-r")
	args = append(args, "-J")
	args = append(args, s.UserData)
	if s.MetaData != "" {
		args = append(args, s.MetaData)
	}

	// Check is genisoimage or mkisofs is available on the system
	var cmd *exec.Cmd
	if utils.CommandExists("genisoimage") {
		cmd = exec.Command("genisoimage", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("'genisoimage' output: %s", utils.OneLine(out))
		}
		return nil
	}
	if utils.CommandExists("mkisofs") {
		cmd = exec.Command("mkisofs", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("'mkisofs' output: %s", utils.OneLine(out))
		}
		return nil
	}
	return fmt.Errorf("ISO creation package not found, install mkisofs or genisoimage")
}
