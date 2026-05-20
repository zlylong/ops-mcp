package config

import (
	"os"

	"github.com/zlylong/ops-mcp/backend/internal/domain"
)

type Config struct {
	Environment  domain.Environment
	Mode         string
	DatabaseURL  string
	Addr         string
	SeedMockData bool
}

func NewConfig() Config {
	return Config{
		Environment:  domain.EnvDevelopment,
		Mode:         "debug",
		DatabaseURL:  "",
		Addr:         ":8080",
		SeedMockData: true,
	}
}

func Load() (Config, error) {
	cfg := NewConfig()

	if env := os.Getenv("MCP_ENVIRONMENT"); env != "" {
		cfg.Environment = domain.Environment(env)
	}
	if mode := os.Getenv("MCP_MODE"); mode != "" {
		cfg.Mode = mode
	}
	if dbURL := os.Getenv("MCP_DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}
	if addr := os.Getenv("MCP_ADDR"); addr != "" {
		cfg.Addr = addr
	}
	seed := os.Getenv("MCP_SEED_MOCK_DATA")
	if seed == "true" || seed == "1" {
		cfg.SeedMockData = true
	}

	return cfg, nil
}
