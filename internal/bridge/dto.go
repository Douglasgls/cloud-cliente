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

type SessionDTO struct {
	ID            string `json:"id"`
	AccessToken   string `json:"access_token"`
	ContainerName string `json:"container_name"`
	Hostname      string `json:"hostname"`
	TailscaleIP   string `json:"tailscale_ip"`
	CreatedAt     string `json:"created_at"`
	LastUsedAt    string `json:"last_used_at"`
	IsActive      bool   `json:"is_active"`
}
