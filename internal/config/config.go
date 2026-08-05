package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
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
	DefaultBaseURL = "http://192.168.0.181:7070"
	DefaultTimeout = 15 * time.Second
	ClientName     = "cloud-client"
	Version        = "0.1.0"
)

func Load() *Config {
	findAndLoadDotEnv()

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

// findAndLoadDotEnv procura o arquivo .env no diretório atual, diretórios pai e pasta do executável
func findAndLoadDotEnv() {
	candidates := []string{".env", "../.env", "../../.env"}

	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		candidates = append(candidates,
			filepath.Join(execDir, ".env"),
			filepath.Join(execDir, "../.env"),
			filepath.Join(execDir, "../../.env"),
		)
	}

	for _, path := range candidates {
		if loadDotEnv(path) {
			break
		}
	}
}

// loadDotEnv lê um arquivo .env se ele existir e carrega as variáveis no ambiente
func loadDotEnv(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	loaded := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
				loaded = true
			}
		}
	}
	return loaded
}


