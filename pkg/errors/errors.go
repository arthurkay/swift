package errors

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Wrap returns a new error with context wrapping the original error.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// LogError logs an error. In debug mode, it includes file and line information.
func LogError(err error) {
	if err == nil {
		return
	}
	if os.Getenv("DEBUG") == "true" {
		_, fn, line, _ := runtime.Caller(1)
		fmt.Fprintf(os.Stderr, "[error] %s:%d %v\n", fn, line, err)
		return
	}
	fmt.Fprintf(os.Stderr, "[error] %v\n", err)
}

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
