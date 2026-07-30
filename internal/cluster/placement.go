package cluster

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	agentpb "swift/internal/proto"
)

// PlacementStrategy determines how VMs are placed across hosts.
type PlacementStrategy string

const (
	StrategyLeastLoaded PlacementStrategy = "least-loaded"
	StrategyRoundRobin  PlacementStrategy = "round-robin"
	StrategyRandom      PlacementStrategy = "random"
)

// PlacementRequest describes a VM that needs to be placed.
type PlacementRequest struct {
	Memory         uint32
	CpuCount       uint32
	DiskSize       uint64
	RequiredLabels map[string]string
}

// PlacementResult is the outcome of a placement decision.
type PlacementResult struct {
	Host      string
	Address   string
	Reason    string
	Projected *agentpb.ResourceUsage
}

// Place finds the best host for a VM using the configured strategy.
func (o *Orchestrator) Place(req *PlacementRequest) (*PlacementResult, error) {
	strategy := StrategyLeastLoaded
	if o.config.Placement.Strategy != "" {
		strategy = PlacementStrategy(o.config.Placement.Strategy)
	}

	// Gather placement responses from all hosts
	candidates := o.evaluateHosts(req)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no host can satisfy the placement request")
	}

	// Filter out excluded hosts
	candidates = o.filterExcluded(candidates)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("all eligible hosts are excluded by placement policy")
	}

	switch strategy {
	case StrategyRoundRobin:
		return o.placeRoundRobin(candidates), nil
	case StrategyRandom:
		return o.placeRandom(candidates), nil
	default:
		return o.placeLeastLoaded(candidates), nil
	}
}

// evaluateHosts queries each host to see if it can satisfy the request.
func (o *Orchestrator) evaluateHosts(req *PlacementRequest) []*PlacementResult {
	results := make([]*PlacementResult, 0, len(o.config.Hosts))

	for _, host := range o.config.Hosts {
		client, err := o.pool.Get(host.HostAddress())
		if err != nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := (*client).PlaceVM(ctx, &agentpb.PlaceVMRequest{
			Memory:         req.Memory,
			CpuCount:       req.CpuCount,
			DiskSize:       req.DiskSize,
			RequiredLabels: req.RequiredLabels,
		})
		cancel()

		if err != nil || !resp.Placed {
			continue
		}

		results = append(results, &PlacementResult{
			Host:      resp.Host,
			Address:   host.HostAddress(),
			Reason:    resp.Reason,
			Projected: resp.Projected,
		})
	}
	return results
}

// filterExcluded removes hosts that match exclusion labels.
func (o *Orchestrator) filterExcluded(candidates []*PlacementResult) []*PlacementResult {
	if len(o.config.Placement.Exclude) == 0 {
		return candidates
	}

	// Look up host labels from config
	var filtered []*PlacementResult
	for _, c := range candidates {
		excluded := false
		for _, host := range o.config.Hosts {
			if host.Name == c.Host {
				for k, v := range o.config.Placement.Exclude {
					if host.Labels[k] == v {
						excluded = true
						break
					}
				}
				break
			}
		}
		if !excluded {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// placeLeastLoaded selects the host with the most free resources.
func (o *Orchestrator) placeLeastLoaded(candidates []*PlacementResult) *PlacementResult {
	sort.Slice(candidates, func(i, j int) bool {
		pi := candidates[i].Projected
		pj := candidates[j].Projected
		if pi == nil || pj == nil {
			return pi == nil
		}
		freeI := pi.TotalMemory - pi.UsedMemory
		freeJ := pj.TotalMemory - pj.UsedMemory
		return freeI > freeJ
	})
	return candidates[0]
}

// placeRoundRobin selects hosts in order (simple counter-based).
func (o *Orchestrator) placeRoundRobin(candidates []*PlacementResult) *PlacementResult {
	// Simple: use timestamp mod for round-robin feel
	idx := time.Now().UnixNano() % int64(len(candidates))
	return candidates[idx]
}

// placeRandom selects a random host from candidates.
func (o *Orchestrator) placeRandom(candidates []*PlacementResult) *PlacementResult {
	idx := rand.Intn(len(candidates))
	return candidates[idx]
}
