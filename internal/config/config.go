package config

import (
	"os"
	"time"
)

type Config struct {
	BaseURL    string
	Timeout    time.Duration
	ClientName string
	Version    string
	UserAgent  string
}

const (
	DefaultBaseURL = "http://192.168.0.181:8090"
	DefaultTimeout = 15 * time.Second
	ClientName     = "cloud-client"
	Version        = "0.1.0"
)

func Load() *Config {
	baseURL := os.Getenv("CLOUD_API_URL")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Config{
		BaseURL:    baseURL,
		Timeout:    DefaultTimeout,
		ClientName: ClientName,
		Version:    Version,
		UserAgent:  ClientName + "/" + Version,
	}
}
