package agent

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// getMemoryStats returns total and used memory in bytes from /proc/meminfo.
func getMemoryStats() (total, used uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	var memTotal, memAvail uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			memTotal = parseMemInfoKB(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			memAvail = parseMemInfoKB(line)
		}
	}
	total = memTotal * 1024
	used = (memTotal - memAvail) * 1024
	return
}

// parseMemInfoKB extracts the kB value from a /proc/meminfo line.
func parseMemInfoKB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	val, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// getDiskStats returns total and used disk space in bytes for the given path.
func getDiskStats(path string) (total, used uint64) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	total = stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used = total - free
	return
}

// getCPUCount returns the number of CPUs on the host.
func getCPUCount() uint32 {
	return uint32(runtime.NumCPU())
}

// HostResources holds resource information about the host.
type HostResources struct {
	TotalMemory uint64
	UsedMemory  uint64
	TotalCPUs   uint32
	UsedCPUs    uint32
	TotalDisk   uint64
	UsedDisk    uint64
}

// String returns a human-readable summary.
func (r *HostResources) String() string {
	return fmt.Sprintf("Mem: %d/%d MiB, CPU: %d/%d, Disk: %d/%d GiB",
		r.UsedMemory/1024/1024, r.TotalMemory/1024/1024,
		r.UsedCPUs, r.TotalCPUs,
		r.UsedDisk/1024/1024/1024, r.TotalDisk/1024/1024/1024)
}
