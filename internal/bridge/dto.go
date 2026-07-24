package bridge

type ConnectionInfoDTO struct {
	ConnectionID  string `json:"connection_id"`
	Hostname      string `json:"hostname"`
	TailscaleIP   string `json:"tailscale_ip"`
	TailscaleIPv6 string `json:"tailscale_ipv6"`
}

type ForwardingDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	RemotePort int    `json:"remote_port"`
	LocalPort  int    `json:"local_port"`
	Enabled    bool   `json:"enabled"`
	IsDefault  bool   `json:"is_default"`
	Running    bool   `json:"running"`
	LastError  string `json:"last_error,omitempty"`
}
