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

	cmd := exec.Command("genisoimage", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("'genisoimage' output: %s", utils.OneLine(out))
	}
	return nil
}
