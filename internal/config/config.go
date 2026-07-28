package config

import (
	"fmt"
	"os"
)

const (
	// SwiftHomeDir is the root directory for all swift VM data.
	SwiftHomeDir = "/var/swift"

	// DefaultUser is the default cloud-init user for VMs.
	DefaultUser = "swift"

	// DefaultPassword is the default password for VMs.
	DefaultPassword = "swift1234"

	// DirPermission is the permission used for swift directories.
	DirPermission = 0775

	// FilePermission is the permission used for swift files.
	FilePermission = 0755
)

// Home returns the swift home directory, creating it if it does not exist.
func Home() (string, error) {
	if _, err := os.Stat(SwiftHomeDir); err != nil {
		if err := os.Mkdir(SwiftHomeDir, DirPermission); err != nil {
			return "", fmt.Errorf("create swift home: %w", err)
		}
		if err := os.Chown(SwiftHomeDir, os.Getuid(), os.Getgid()); err != nil {
			return "", fmt.Errorf("chown swift home: %w", err)
		}
	}
	return SwiftHomeDir, nil
}

// VMPath returns the path to a VM's project directory.
func VMPath(slug string) (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}
	return home + "/" + slug, nil
}
