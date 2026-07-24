package forwarding

type Forwarding struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	RemotePort int    `json:"remote_port"`
	LocalPort  int    `json:"local_port"`
	Enabled    bool   `json:"enabled"`
	IsDefault  bool   `json:"is_default"`
}

type ForwardingState struct {
	Forwarding Forwarding `json:"forwarding"`
	Running    bool       `json:"running"`
	LastError  string     `json:"last_error,omitempty"`
}

func DefaultForwardings() []Forwarding {
	return []Forwarding{
		{
			ID:         "ssh",
			Name:       "SSH",
			RemotePort: 22,
			LocalPort:  2222,
			Enabled:    true,
			IsDefault:  true,
		},
		{
			ID:         "http",
			Name:       "HTTP",
			RemotePort: 80,
			LocalPort:  8080,
			Enabled:    true,
			IsDefault:  true,
		},
		{
			ID:         "https",
			Name:       "HTTPS",
			RemotePort: 443,
			LocalPort:  8443,
			Enabled:    true,
			IsDefault:  true,
		},
	}
}
