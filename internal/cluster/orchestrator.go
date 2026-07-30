package cluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	agentpb "swift/internal/proto"
)

// Orchestrator manages operations across multiple cluster hosts.
type Orchestrator struct {
	config *ClusterConfig
	pool   *ClientPool
}

// NewOrchestrator creates a new orchestrator from the cluster config.
func NewOrchestrator(cfg *ClusterConfig) *Orchestrator {
	return &Orchestrator{
		config: cfg,
		pool:   NewClientPool(),
	}
}

// Close cleans up all connections.
func (o *Orchestrator) Close() {
	o.pool.CloseAll()
}

// GetPool returns the client pool for direct agent access.
func (o *Orchestrator) GetPool() *ClientPool {
	return o.pool
}

// HostInfoResult holds the result of a GetHostInfo call.
type HostInfoResult struct {
	Host string
	Info *agentpb.HostInfo
	Err  error
}

// ListHosts returns info from all configured hosts concurrently.
func (o *Orchestrator) ListHosts() []HostInfoResult {
	results := make([]HostInfoResult, len(o.config.Hosts))
	var wg sync.WaitGroup

	for i, host := range o.config.Hosts {
		wg.Add(1)
		go func(idx int, h HostConfig) {
			defer wg.Done()
			results[idx] = o.getHostInfo(h)
		}(i, host)
	}

	wg.Wait()
	return results
}

// getHostInfo fetches info from a single host.
func (o *Orchestrator) getHostInfo(host HostConfig) HostInfoResult {
	client, err := o.pool.Get(host.HostAddress())
	if err != nil {
		return HostInfoResult{Host: host.Name, Err: err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := (*client).GetHostInfo(ctx, &agentpb.GetHostInfoRequest{})
	if err != nil {
		return HostInfoResult{Host: host.Name, Err: err}
	}
	return HostInfoResult{Host: host.Name, Info: info}
}

// VMInfoResult holds the result of a VM listing from one host.
type VMInfoResult struct {
	Host string
	VMs  []*agentpb.VMInfo
	Err  error
}

// ListAllVMs returns VMs from all hosts.
func (o *Orchestrator) ListAllVMs() []VMInfoResult {
	results := make([]VMInfoResult, len(o.config.Hosts))
	var wg sync.WaitGroup

	for i, host := range o.config.Hosts {
		wg.Add(1)
		go func(idx int, h HostConfig) {
			defer wg.Done()
			results[idx] = o.listVMs(h)
		}(i, host)
	}

	wg.Wait()
	return results
}

func (o *Orchestrator) listVMs(host HostConfig) VMInfoResult {
	client, err := o.pool.Get(host.HostAddress())
	if err != nil {
		return VMInfoResult{Host: host.Name, Err: err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := (*client).ListVMs(ctx, &agentpb.ListVMsRequest{})
	if err != nil {
		return VMInfoResult{Host: host.Name, Err: err}
	}
	return VMInfoResult{Host: host.Name, VMs: resp.Vms}
}

// FindVM locates a VM across all hosts by name or UUID.
func (o *Orchestrator) FindVM(nameOrUUID string) (*agentpb.VMInfo, string, error) {
	for _, host := range o.config.Hosts {
		client, err := o.pool.Get(host.HostAddress())
		if err != nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		resp, err := (*client).GetVM(ctx, &agentpb.GetVMRequest{NameOrUuid: nameOrUUID})
		cancel()

		if err == nil && resp.Vm != nil {
			return resp.Vm, host.HostAddress(), nil
		}
	}
	return nil, "", fmt.Errorf("VM %q not found on any host", nameOrUUID)
}

// ResolveHost finds a host config by name, returning the default if empty.
func (o *Orchestrator) ResolveHost(name string) (*HostConfig, error) {
	if name == "" {
		return o.config.GetDefaultHost()
	}
	return o.config.GetHostByName(name)
}

// StartVM starts a VM by finding it across hosts.
func (o *Orchestrator) StartVM(nameOrUUID string) error {
	_, addr, err := o.FindVM(nameOrUUID)
	if err != nil {
		return err
	}

	client, err := o.pool.Get(addr)
	if err != nil {
		return fmt.Errorf("connect to host: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := (*client).StartVM(ctx, &agentpb.StartVMRequest{NameOrUuid: nameOrUUID})
	if err != nil {
		return fmt.Errorf("start VM: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("start VM: %s", resp.Error)
	}
	return nil
}

// StopVM stops a VM by finding it across hosts.
func (o *Orchestrator) StopVM(nameOrUUID string) error {
	_, addr, err := o.FindVM(nameOrUUID)
	if err != nil {
		return err
	}

	client, err := o.pool.Get(addr)
	if err != nil {
		return fmt.Errorf("connect to host: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := (*client).StopVM(ctx, &agentpb.StopVMRequest{NameOrUuid: nameOrUUID})
	if err != nil {
		return fmt.Errorf("stop VM: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("stop VM: %s", resp.Error)
	}
	return nil
}

// DeleteVM deletes a VM by finding it across hosts.
func (o *Orchestrator) DeleteVM(nameOrUUID string) error {
	_, addr, err := o.FindVM(nameOrUUID)
	if err != nil {
		return err
	}

	client, err := o.pool.Get(addr)
	if err != nil {
		return fmt.Errorf("connect to host: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := (*client).DeleteVM(ctx, &agentpb.DeleteVMRequest{NameOrUuid: nameOrUUID})
	if err != nil {
		return fmt.Errorf("delete VM: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("delete VM: %s", resp.Error)
	}
	return nil
}

// GetVMXML returns the XML definition of a VM.
func (o *Orchestrator) GetVMXML(nameOrUUID string) (string, error) {
	_, addr, err := o.FindVM(nameOrUUID)
	if err != nil {
		return "", err
	}

	client, err := o.pool.Get(addr)
	if err != nil {
		return "", fmt.Errorf("connect to host: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := (*client).GetVMXML(ctx, &agentpb.GetVMXMLRequest{NameOrUuid: nameOrUUID})
	if err != nil {
		return "", fmt.Errorf("get XML: %w", err)
	}
	return resp.Xml, nil
}
