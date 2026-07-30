package agent

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"swift/internal/cloudinit"
	"swift/internal/hv"
	"swift/internal/image"
	agentpb "swift/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Config holds agent configuration.
type Config struct {
	Name     string            `yaml:"name"`
	Address  string            `yaml:"address"`
	Labels   map[string]string `yaml:"labels"`
	Libvirt  string            `yaml:"libvirt"`
	APIPort  int               `yaml:"api_port"`
	CacheDir string            `yaml:"cache_dir"`
}

// Server is the gRPC server wrapping local libvirt operations.
type Server struct {
	agentpb.UnimplementedSwiftAgentServer
	config Config
	hv     *hv.LibvirtHypervisor
	health *HealthMonitor
	cache  *ImageCache
	place  *PlacementEngine
	mu     sync.RWMutex
}

// NewServer creates a new agent server.
func NewServer(cfg Config) (*Server, error) {
	uri := cfg.Libvirt
	if uri == "" {
		uri = "qemu:///system"
	}
	h, err := hv.ConnectURI(uri)
	if err != nil {
		return nil, fmt.Errorf("connect to libvirt: %w", err)
	}

	cacheDir := cfg.CacheDir
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = home + "/.swift/cache"
	}

	s := &Server{
		config: cfg,
		hv:     h,
		cache:  NewImageCache(cacheDir),
		place:  NewPlacementEngine(),
	}
	s.health = NewHealthMonitor(s)
	return s, nil
}

// Run starts the gRPC server and blocks.
func (s *Server) Run() error {
	addr := s.config.Address
	if addr == "" {
		addr = fmt.Sprintf(":%d", s.config.APIPort)
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	grpcServer := grpc.NewServer()
	agentpb.RegisterSwiftAgentServer(grpcServer, s)

	s.health.Start()
	log.Printf("swift-agent %q listening on %s", s.config.Name, addr)
	return grpcServer.Serve(lis)
}

// Close cleans up resources.
func (s *Server) Close() {
	s.health.Stop()
	if s.hv != nil {
		s.hv.Disconnect()
	}
}

// === VM Lifecycle ===

func (s *Server) DefineVM(ctx context.Context, req *agentpb.DefineVMRequest) (*agentpb.DefineVMResponse, error) {
	// Pull backing image if URL provided
	backingPath := ""
	if req.BackingImageUrl != "" {
		path, err := s.cache.EnsureCached(req.BackingImageUrl, req.BackingImageSha256)
		if err != nil {
			return &agentpb.DefineVMResponse{Success: false, Error: fmt.Sprintf("cache backing image: %v", err)}, nil
		}
		backingPath = path
	}

	// Create project directory
	projectDir, err := cloudinit.CreateProjectDir(req.Name)
	if err != nil {
		return &agentpb.DefineVMResponse{Success: false, Error: fmt.Sprintf("create project dir: %v", err)}, nil
	}

	// Generate cloud-init user-data
	if err := cloudinit.WriteCloudInit(projectDir, req.Name, req.CloudInitUserData); err != nil {
		return &agentpb.DefineVMResponse{Success: false, Error: fmt.Sprintf("write cloud-init: %v", err)}, nil
	}

	// Create seed ISO
	isoPath := projectDir + "/" + req.Name + ".img"
	seed := cloudinit.NewSeed(isoPath, projectDir+"/user-data", projectDir+"/meta-data")
	if err := seed.Create(); err != nil {
		return &agentpb.DefineVMResponse{Success: false, Error: fmt.Sprintf("create seed ISO: %v", err)}, nil
	}

	// Create QCOW2 overlay
	diskPath := projectDir + "/" + req.Name + ".qcow2"
	if backingPath != "" {
		img := image.NewImage(diskPath, image.FormatQCOW2, req.DiskSize)
		if err := img.SetBackingFile(backingPath); err != nil {
			return &agentpb.DefineVMResponse{Success: false, Error: fmt.Sprintf("set backing file: %v", err)}, nil
		}
		if err := img.Create(); err != nil {
			return &agentpb.DefineVMResponse{Success: false, Error: fmt.Sprintf("create disk: %v", err)}, nil
		}
	}

	// Build domain XML
	netModel := req.NetworkModel
	if netModel == "" {
		netModel = "e1000"
	}
	resources := hv.DomainResources{
		Name:     req.Name,
		Memory:   uint(req.Memory),
		Unit:     req.MemoryUnit,
		CpuCount: uint(req.CpuCount),
		Arch:     req.Arch,
		BootOS:   diskPath,
		CDRom:    isoPath,
		Networks: []hv.NetworkConfig{{Name: "default", Model: netModel}},
	}
	dom := resources.BuildDomainXML()
	xml, err := dom.Marshal()
	if err != nil {
		return &agentpb.DefineVMResponse{Success: false, Error: fmt.Sprintf("marshal XML: %v", err)}, nil
	}

	if err := s.hv.DefineDomain(xml); err != nil {
		return &agentpb.DefineVMResponse{Success: false, Error: fmt.Sprintf("define domain: %v", err)}, nil
	}

	info, _ := s.hv.LookupDomainByName(req.Name)
	uuid := ""
	if info != nil {
		uuid = info.UUID
	}

	return &agentpb.DefineVMResponse{Success: true, VmUuid: uuid}, nil
}

func (s *Server) DeleteVM(ctx context.Context, req *agentpb.DeleteVMRequest) (*agentpb.DeleteVMResponse, error) {
	info, err := s.hv.LookupDomain(req.NameOrUuid)
	if err != nil {
		return &agentpb.DeleteVMResponse{Success: false, Error: err.Error()}, nil
	}
	if err := s.hv.UndefineDomain(info.Name); err != nil {
		return &agentpb.DeleteVMResponse{Success: false, Error: err.Error()}, nil
	}
	return &agentpb.DeleteVMResponse{Success: true}, nil
}

func (s *Server) StartVM(ctx context.Context, req *agentpb.StartVMRequest) (*agentpb.StartVMResponse, error) {
	info, err := s.hv.LookupDomain(req.NameOrUuid)
	if err != nil {
		return &agentpb.StartVMResponse{Success: false, Error: err.Error()}, nil
	}
	if err := s.hv.StartDomain(info.Name); err != nil {
		return &agentpb.StartVMResponse{Success: false, Error: err.Error()}, nil
	}
	return &agentpb.StartVMResponse{Success: true}, nil
}

func (s *Server) StopVM(ctx context.Context, req *agentpb.StopVMRequest) (*agentpb.StopVMResponse, error) {
	info, err := s.hv.LookupDomain(req.NameOrUuid)
	if err != nil {
		return &agentpb.StopVMResponse{Success: false, Error: err.Error()}, nil
	}
	if err := s.hv.StopDomain(info.Name); err != nil {
		return &agentpb.StopVMResponse{Success: false, Error: err.Error()}, nil
	}
	return &agentpb.StopVMResponse{Success: true}, nil
}

func (s *Server) RebootVM(ctx context.Context, req *agentpb.RebootVMRequest) (*agentpb.RebootVMResponse, error) {
	info, err := s.hv.LookupDomain(req.NameOrUuid)
	if err != nil {
		return &agentpb.RebootVMResponse{Success: false, Error: err.Error()}, nil
	}
	if err := s.hv.RebootDomain(info.Name); err != nil {
		return &agentpb.RebootVMResponse{Success: false, Error: err.Error()}, nil
	}
	return &agentpb.RebootVMResponse{Success: true}, nil
}

// === VM Queries ===

func (s *Server) ListVMs(ctx context.Context, req *agentpb.ListVMsRequest) (*agentpb.ListVMsResponse, error) {
	domains, err := s.hv.ListDomains()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list domains: %v", err)
	}
	vms := make([]*agentpb.VMInfo, 0, len(domains))
	for _, d := range domains {
		vms = append(vms, &agentpb.VMInfo{
			Name:  d.Name,
			Uuid:  d.UUID,
			State: d.State,
			Id:    int32(d.ID),
			Host:  s.config.Name,
		})
	}
	return &agentpb.ListVMsResponse{Vms: vms}, nil
}

func (s *Server) GetVM(ctx context.Context, req *agentpb.GetVMRequest) (*agentpb.GetVMResponse, error) {
	info, err := s.hv.LookupDomain(req.NameOrUuid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "domain not found: %v", err)
	}
	return &agentpb.GetVMResponse{
		Vm: &agentpb.VMInfo{
			Name:  info.Name,
			Uuid:  info.UUID,
			State: info.State,
			Id:    int32(info.ID),
			Host:  s.config.Name,
		},
	}, nil
}

func (s *Server) GetVMXML(ctx context.Context, req *agentpb.GetVMXMLRequest) (*agentpb.GetVMXMLResponse, error) {
	info, err := s.hv.LookupDomain(req.NameOrUuid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "domain not found: %v", err)
	}
	xml, err := s.hv.DomainXML(info.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get XML: %v", err)
	}
	return &agentpb.GetVMXMLResponse{Xml: xml}, nil
}

// === Placement ===

func (s *Server) PlaceVM(ctx context.Context, req *agentpb.PlaceVMRequest) (*agentpb.PlaceVMResponse, error) {
	resources, err := s.place.GetResources(s.hv)
	if err != nil {
		return &agentpb.PlaceVMResponse{
			Placed: false,
			Host:   s.config.Name,
			Reason: fmt.Sprintf("failed to get resources: %v", err),
		}, nil
	}

	availMem := resources.TotalMemory - resources.UsedMemory
	availCPU := resources.TotalCPUs - resources.UsedCPUs
	availDisk := resources.TotalDisk - resources.UsedDisk

	if uint64(req.Memory)*1024*1024 > availMem {
		return &agentpb.PlaceVMResponse{Placed: false, Host: s.config.Name, Reason: "insufficient memory"}, nil
	}
	if uint32(req.CpuCount) > availCPU {
		return &agentpb.PlaceVMResponse{Placed: false, Host: s.config.Name, Reason: "insufficient CPU"}, nil
	}
	if req.DiskSize > availDisk {
		return &agentpb.PlaceVMResponse{Placed: false, Host: s.config.Name, Reason: "insufficient disk"}, nil
	}
	for k, v := range req.RequiredLabels {
		if s.config.Labels[k] != v {
			return &agentpb.PlaceVMResponse{
				Placed: false,
				Host:   s.config.Name,
				Reason: fmt.Sprintf("missing required label %s=%s", k, v),
			}, nil
		}
	}

	return &agentpb.PlaceVMResponse{
		Placed: true,
		Host:   s.config.Name,
		Reason: "resources available",
		Projected: &agentpb.ResourceUsage{
			TotalMemory: resources.TotalMemory,
			UsedMemory:  resources.UsedMemory + uint64(req.Memory)*1024*1024,
			TotalCpus:   resources.TotalCPUs,
			UsedCpus:    resources.UsedCPUs + uint32(req.CpuCount),
			TotalDisk:   resources.TotalDisk,
			UsedDisk:    resources.UsedDisk + req.DiskSize,
		},
	}, nil
}

// === Health ===

func (s *Server) GetHostInfo(ctx context.Context, req *agentpb.GetHostInfoRequest) (*agentpb.HostInfo, error) {
	resources, err := s.place.GetResources(s.hv)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get resources: %v", err)
	}

	domains, _ := s.hv.ListDomains()
	vms := make([]*agentpb.VMInfo, 0, len(domains))
	for _, d := range domains {
		vms = append(vms, &agentpb.VMInfo{
			Name:  d.Name,
			Uuid:  d.UUID,
			State: d.State,
			Id:    int32(d.ID),
			Host:  s.config.Name,
		})
	}

	hostname, _ := os.Hostname()
	name := s.config.Name
	if name == "" {
		name = hostname
	}

	return &agentpb.HostInfo{
		Name:    name,
		Address: s.config.Address,
		Labels:  s.config.Labels,
		Healthy: s.health.IsHealthy(),
		Resources: &agentpb.ResourceUsage{
			TotalMemory: resources.TotalMemory,
			UsedMemory:  resources.UsedMemory,
			TotalCpus:   resources.TotalCPUs,
			UsedCpus:    resources.UsedCPUs,
			TotalDisk:   resources.TotalDisk,
			UsedDisk:    resources.UsedDisk,
		},
		Vms: vms,
	}, nil
}

func (s *Server) WatchHealth(stream agentpb.SwiftAgent_WatchHealthServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
		}
	}
}

// === Image Cache ===

func (s *Server) PullImage(req *agentpb.PullImageRequest, stream agentpb.SwiftAgent_PullImageServer) error {
	progressCh := s.cache.Pull(req.Url, req.ExpectedSha256)
	for p := range progressCh {
		if err := stream.Send(&agentpb.PullImageProgress{
			State:          p.State,
			BytesDownloaded: p.BytesDownloaded,
			TotalBytes:     p.TotalBytes,
			LocalPath:      p.LocalPath,
			Error:          p.Error,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) CacheStatus(ctx context.Context, req *agentpb.CacheStatusRequest) (*agentpb.CacheStatusResponse, error) {
	images, totalSize := s.cache.Status()
	var cached []*agentpb.CachedImage
	for _, img := range images {
		cached = append(cached, &agentpb.CachedImage{
			Url:       img.URL,
			LocalPath: img.LocalPath,
			Sha256:    img.SHA256,
			Size:      img.Size,
			CachedAt:  img.CachedAt,
		})
	}
	return &agentpb.CacheStatusResponse{Images: cached, TotalSize: totalSize}, nil
}
