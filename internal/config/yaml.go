package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadVMConfig reads a YAML file and returns a VMConfig.
func LoadVMConfig(path string) (*VMConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read VM config %q: %w", path, err)
	}
	var cfg VMConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse VM config %q: %w", path, err)
	}
	return &cfg, nil
}

// LoadNetworkConfig reads a YAML file and returns a NetworkConfigYAML.
func LoadNetworkConfig(path string) (*NetworkConfigYAML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read network config %q: %w", path, err)
	}
	var cfg NetworkConfigYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse network config %q: %w", path, err)
	}
	return &cfg, nil
}
