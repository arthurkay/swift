package errors

import (
	"os/exec"
	"strings"
)

// OneLine normalizes multi-line byte output into a single line.
func OneLine(in []byte) string {
	str := strings.TrimSpace(string(in))
	return strings.ReplaceAll(str, "\n", ". ")
}

// CommandExists checks if an executable exists on the system PATH.
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
