package image

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"swift/pkg/errors"
)

const (
	// Image format constants
	FormatRAW   = "raw"
	FormatQCOW  = "qcow"
	FormatQCOW2 = "qcow2"
	FormatVMDK  = "vmdk"
	FormatVDI   = "vdi"
	FormatVHDX  = "vhdx"
	FormatVPC   = "vpc"

	// GiB is one gibibyte in bytes (2^30).
	GiB = 1073741824
)

// QEMUImage represents a QEMU disk image.
type QEMUImage struct {
	Path        string
	Format      string
	Size        uint64
	backingFile string
}

// NewImage creates a new QEMUImage.
func NewImage(path, format string, size uint64) QEMUImage {
	return QEMUImage{
		Path:   path,
		Format: format,
		Size:   size,
	}
}

// SetBackingFile sets a backing file for copy-on-write images.
// The backing file must exist on disk.
func (i *QEMUImage) SetBackingFile(backingFile string) error {
	if _, err := os.Stat(backingFile); os.IsNotExist(err) {
		return fmt.Errorf("backing file %q does not exist", backingFile)
	}
	i.backingFile = backingFile
	return nil
}

// Create runs qemu-img to create the disk image.
func (i QEMUImage) Create() error {
	args := []string{"create", "-f", i.Format, "-F", i.Format}
	if i.backingFile != "" {
		args = append(args, "-b", i.backingFile)
	}
	args = append(args, i.Path, strconv.FormatUint(i.Size, 10))

	cmd := exec.Command("qemu-img", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-img create: %s", errors.OneLine(out))
	}
	return nil
}

// CreateSnapshot creates a named snapshot of the disk image.
func (i QEMUImage) CreateSnapshot(name string) error {
	cmd := exec.Command("qemu-img", "snapshot", "-c", name, i.Path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-img snapshot: %s", errors.OneLine(out))
	}
	return nil
}
