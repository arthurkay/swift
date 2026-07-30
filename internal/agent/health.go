package agent

import (
	"log"
	"sync"
	"time"

	"swift/internal/hv"
)

// HealthMonitor tracks agent health.
type HealthMonitor struct {
	server  *Server
	healthy bool
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(server *Server) *HealthMonitor {
	return &HealthMonitor{
		server:  server,
		healthy: true,
		stopCh:  make(chan struct{}),
	}
}

// Start begins periodic health checks.
func (h *HealthMonitor) Start() {
	go h.loop()
}

// Stop halts health monitoring.
func (h *HealthMonitor) Stop() {
	close(h.stopCh)
}

// IsHealthy returns current health status.
func (h *HealthMonitor) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.healthy
}

func (h *HealthMonitor) loop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.check()
		}
	}
}

func (h *HealthMonitor) check() {
	hv := h.server.hv
	if hv == nil {
		h.setHealthy(false)
		return
	}

	// Check if we can list domains
	_, err := hv.ListDomains()
	if err != nil {
		log.Printf("[health] libvirt check failed: %v", err)
		h.setHealthy(false)
		return
	}

	h.setHealthy(true)
}

func (h *HealthMonitor) setHealthy(v bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.healthy != v {
		log.Printf("[health] status changed: %v -> %v", h.healthy, v)
		h.healthy = v
	}
}

// PlacementEngine tracks host resources for placement decisions.
type PlacementEngine struct {
	mu sync.RWMutex
}

// NewPlacementEngine creates a new placement engine.
func NewPlacementEngine() *PlacementEngine {
	return &PlacementEngine{}
}

// GetResources returns current host resource usage.
func (p *PlacementEngine) GetResources(h *hv.LibvirtHypervisor) (*HostResources, error) {
	totalMem, usedMem := getMemoryStats()
	totalDisk, usedDisk := getDiskStats("/var/swift")

	// Count running VMs for CPU estimation
	domains, err := h.ListDomains()
	if err != nil {
		return nil, err
	}

	var usedCPUs uint32
	for _, d := range domains {
		if d.State == "Running" {
			usedCPUs += 1 // simplified — would query per-VM vCPU count
		}
	}

	return &HostResources{
		TotalMemory: totalMem,
		UsedMemory:  usedMem,
		TotalCPUs:   getCPUCount(),
		UsedCPUs:    usedCPUs,
		TotalDisk:   totalDisk,
		UsedDisk:    usedDisk,
	}, nil
}
