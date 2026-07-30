package cluster

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigDir  = ".swift"
	defaultConfigFile = "config.yaml"
	defaultAPIPort    = 9800
)

// HostConfig defines a single host in the cluster.
type HostConfig struct {
	Name     string            `yaml:"name"`
	Address  string            `yaml:"address"`
	Labels   map[string]string `yaml:"labels"`
	Libvirt  string            `yaml:"libvirt"`
	APIPort  int               `yaml:"api_port"`
	CacheDir string            `yaml:"cache_dir"`
}

// ClusterConfig is the top-level cluster configuration.
type ClusterConfig struct {
	Cluster struct {
		Name string `yaml:"name"`
	} `yaml:"cluster"`

	DefaultHost string       `yaml:"default_host"`
	Hosts       []HostConfig `yaml:"hosts"`

	Placement struct {
		Strategy string              `yaml:"strategy"`
		Exclude  map[string]string   `yaml:"exclude_labels"`
	} `yaml:"placement"`
}

// ConfigPath returns the path to the cluster config file.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, defaultConfigDir, defaultConfigFile)
}

// ConfigDir returns the swift config directory.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, defaultConfigDir)
}

// LoadConfig reads the cluster config file.
func LoadConfig() (*ClusterConfig, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var cfg ClusterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	return &cfg, nil
}

// ConfigExists checks if a config file exists.
func ConfigExists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

// SaveConfig writes the cluster config to disk.
func SaveConfig(cfg *ClusterConfig) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	path := ConfigPath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// GetHostByName finds a host by name.
func (c *ClusterConfig) GetHostByName(name string) (*HostConfig, error) {
	for i := range c.Hosts {
		if c.Hosts[i].Name == name {
			return &c.Hosts[i], nil
		}
	}
	return nil, fmt.Errorf("host %q not found", name)
}

// GetDefaultHost returns the default host config.
func (c *ClusterConfig) GetDefaultHost() (*HostConfig, error) {
	if c.DefaultHost != "" {
		return c.GetHostByName(c.DefaultHost)
	}
	if len(c.Hosts) > 0 {
		return &c.Hosts[0], nil
	}
	return nil, fmt.Errorf("no hosts configured")
}

// HostAddress returns the gRPC address for a host.
func (h *HostConfig) HostAddress() string {
	if h.APIPort > 0 {
		return fmt.Sprintf("%s:%d", h.Address, h.APIPort)
	}
	return h.Address
}
