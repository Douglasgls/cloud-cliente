package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithEnv(t *testing.T) {
	os.Setenv("CLOUD_API_URL", "http://test-host:9999")
	defer os.Unsetenv("CLOUD_API_URL")

	cfg := Load()
	if cfg.BaseURL != "http://test-host:9999" {
		t.Errorf("esperava http://test-host:9999, obteve %s", cfg.BaseURL)
	}
}

func TestLoadDefault(t *testing.T) {
	os.Unsetenv("CLOUD_API_URL")

	cfg := Load()
	if cfg.BaseURL == "" {
		t.Errorf("esperava BaseURL padrão não nula")
	}
}

func TestLoadDotEnvFileParsing(t *testing.T) {
	// Limpa variável de ambiente
	os.Unsetenv("CLOUD_API_URL")

	// Cria arquivo temporário .env
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envPath, []byte("CLOUD_API_URL=http://10.0.0.1:8090\n"), 0644)
	if err != nil {
		t.Fatalf("falha ao criar arquivo .env temporário: %v", err)
	}

	// Executa loadDotEnv diretamente
	loadDotEnv(envPath)

	if got := os.Getenv("CLOUD_API_URL"); got != "http://10.0.0.1:8090" {
		t.Errorf("loadDotEnv falhou em carregar a variável. Esperado: http://10.0.0.1:8090, Obtido: %s", got)
	}

	// Limpa após o teste
	os.Unsetenv("CLOUD_API_URL")
}

func TestRealDotEnvFile(t *testing.T) {
	os.Unsetenv("CLOUD_API_URL")

	// Testa carregar a partir da raiz do projeto (../../.env)
	loadDotEnv("../../.env")

	got := os.Getenv("CLOUD_API_URL")
	if got == "" {
		t.Errorf("leitura do .env real da raiz falhou, variável CLOUD_API_URL não foi definida")
	}

	t.Logf("URL carregada com sucesso do .env: %s", got)
	os.Unsetenv("CLOUD_API_URL")
}
