package config

// VMConfig represents a YAML-defined VM configuration.
type VMConfig struct {
	Name      string           `yaml:"name"`
	Disk      string           `yaml:"disk"`
	Memory    uint             `yaml:"memory"`
	CPU       uint             `yaml:"cpu"`
	Storage   string           `yaml:"storage"`
	Username  string           `yaml:"username"`
	Password  string           `yaml:"password"`
	SSHKey    string           `yaml:"ssh_key"`
	Graphics  string           `yaml:"graphics"` // spice, vnc, none
	WebSocket bool             `yaml:"websocket"`
	Serial    bool             `yaml:"serial"`
	Networks  []VMNetworkEntry `yaml:"networks"`
}

// VMNetworkEntry is a single network reference in a VM config.
type VMNetworkEntry struct {
	Name  string `yaml:"name"`
	Model string `yaml:"model"`
}

// NetworkConfigYAML represents a YAML-defined network configuration.
type NetworkConfigYAML struct {
	Name      string `yaml:"name"`
	CIDR      string `yaml:"cidr"`
	Mode      string `yaml:"mode"`
	Bridge    string `yaml:"bridge"`
	Device    string `yaml:"device"`
	DHCP      bool   `yaml:"dhcp"`
	DHCPStart string `yaml:"dhcp_start"`
	DHCPEnd   string `yaml:"dhcp_end"`
}
