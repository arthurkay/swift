package cluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	agentpb "swift/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientPool manages gRPC connections to multiple agents.
type ClientPool struct {
	clients map[string]*agentpb.SwiftAgentClient
	conns   map[string]*grpc.ClientConn
	mu      sync.RWMutex
}

// NewClientPool creates a new client pool.
func NewClientPool() *ClientPool {
	return &ClientPool{
		clients: make(map[string]*agentpb.SwiftAgentClient),
		conns:   make(map[string]*grpc.ClientConn),
	}
}

// Get returns a gRPC client for the given host address.
func (p *ClientPool) Get(address string) (*agentpb.SwiftAgentClient, error) {
	p.mu.RLock()
	if c, ok := p.clients[address]; ok {
		p.mu.RUnlock()
		return c, nil
	}
	p.mu.RUnlock()

	return p.connect(address)
}

func (p *ClientPool) connect(address string) (*agentpb.SwiftAgentClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if c, ok := p.clients[address]; ok {
		return c, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", address, err)
	}

	client := agentpb.NewSwiftAgentClient(conn)
	p.clients[address] = &client
	p.conns[address] = conn
	return &client, nil
}

// CloseAll closes all connections.
func (p *ClientPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for addr, conn := range p.conns {
		conn.Close()
		delete(p.clients, addr)
		delete(p.conns, addr)
	}
}

// ConnectToHost creates a single gRPC client to a specific host.
func ConnectToHost(host *HostConfig) (*agentpb.SwiftAgentClient, error) {
	pool := NewClientPool()
	return pool.Get(host.HostAddress())
}
