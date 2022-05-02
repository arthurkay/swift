package utils

import (
	"fmt"
	"os"
)

// SwiftHome returns apps configs directory
// Creates one if it does not exist
func SwiftHome() (string, error) {
	path := "/var/swift"

	// Only create the directory if it doesn't exist
	if _, err := os.Stat(path); err != nil {
		if er := os.Mkdir(path, 0775); er != nil {
			return "", fmt.Errorf("%v", er)
		}
		if er := os.Chown(path, os.Getuid(), os.Getgid()); er != nil {
			return "", fmt.Errorf("%v", er)
		}
	}
	return path, nil
}
