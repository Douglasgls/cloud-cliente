package session

import "time"

type Session struct {
	ID            string    `json:"id"`
	AccessToken   string    `json:"access_token"`
	ContainerName string    `json:"container_name,omitempty"`
	Hostname      string    `json:"hostname,omitempty"`
	TailscaleIP   string    `json:"tailscale_ip,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	LastUsedAt    time.Time `json:"last_used_at"`
}

