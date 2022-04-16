package qemu

import (
	"fmt"
	"os"
	"os/exec"
	"rs/utils"
	"strconv"
	"time"
)

const (
	ImageFormatRAW   = "raw"
	ImageFormatCLOOP = "cloop"
	ImageFormatCOW   = "cow"
	ImageFormatQCOW  = "qcow"
	ImageFormatQCOW2 = "qcow2"
	ImageFormatVDMK  = "vdmk"
	ImageFormatVDI   = "vdi"
	ImageFormatVHDX  = "vhdx"
	ImageFormatVPC   = "vpc"
	GiB              = 1073741824 // 1 GiB = 2^30 bytes
)

// Snapshot represents a QEMU image snapshot
// Snapshots are snapshots of the complete virtual machine including CPU state
// RAM, device state and the content of all the writable disks
type Snapshot struct {
	ID      int
	Name    string
	Date    time.Time
	VMClock time.Time
}

// Image represents a QEMU disk image
type Image struct {
	Path   string // Image location (file)
	Format string // Image format
	Size   uint64 // Image size in bytes

	backingFile string
	//	snapshots   []Snapshot
}

func NewImage(path, format string, size uint64) Image {
	var img Image
	img.Path = path
	img.Format = format
	img.Size = size

	return img
}

func (i Image) Create() error {
	args := []string{"create", "-f", i.Format}

	if len(i.backingFile) > 0 {
		args = append(args, "-o")
		args = append(args, fmt.Sprintf("backing_file=%s", i.backingFile))
	}

	args = append(args, i.Path)
	args = append(args, strconv.FormatUint(i.Size, 10))

	cmd := exec.Command("qemu-img", args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("'qemu-img create' output: %s", utils.OneLine(out))
	}

	return nil
}

// CreateSnapshot creates a snapshot of the image
// with the specified name
func (i *Image) CreateSnapshot(name string) error {
	cmd := exec.Command("qemu-img", "snapshot", "-c", name, i.Path)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("'qemu-img snapshot' output: %s", utils.OneLine(out))
	}

	return nil
}

// SetBackingFile sets a backing file for the image
// If it is specified, the image will only record the
// differences from the backing file
func (i *Image) SetBackingFile(backingFile string) error {
	if _, err := os.Stat(backingFile); os.IsNotExist(err) {
		return err
	}

	i.backingFile = backingFile
	return nil
}
