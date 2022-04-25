package utils

import (
	"fmt"
	"os"
)

// SwiftHome returns apps configs directory
// Creates one if it does not exist
func SwiftHome() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("%v", err)
	}
	path := dir + "/swift"
	if _, err := os.Stat(path); err != nil {
		if er := os.Mkdir(path, 0755); er != nil {
			return "", fmt.Errorf("%v", er)
		}
	}
	return path, nil
}
