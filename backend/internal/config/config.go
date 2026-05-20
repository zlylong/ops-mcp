package config

import (
	"os"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
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
		Mode:         "mock",
		DatabaseURL:  "",
		Addr:         ":8080",
		SeedMockData: true,
	}
}

// Load reads configuration from environment variables.
// It prefers the new DARWIN_OPS_MCP_* prefix, then falls back to
// the legacy OPS_MCP_* and MCP_* prefixes for compatibility.
func Load() Config {
	cfg := NewConfig()

	if env := firstEnv("DARWIN_OPS_MCP_ENV", "OPS_MCP_ENV", "MCP_ENVIRONMENT"); env != "" {
		cfg.Environment = domain.Environment(env)
	}
	if mode := firstEnv("DARWIN_OPS_MCP_MODE", "OPS_MCP_MODE", "MCP_MODE"); mode != "" {
		cfg.Mode = mode
	}
	if dbURL := firstEnv("DARWIN_OPS_MCP_DATABASE_URL", "OPS_MCP_DATABASE_URL", "MCP_DATABASE_URL", "DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}
	if addr := firstEnv("DARWIN_OPS_MCP_ADDR", "OPS_MCP_ADDR", "MCP_ADDR"); addr != "" {
		cfg.Addr = addr
	}
	if seed := firstEnv("DARWIN_OPS_MCP_SEED_MOCK", "OPS_MCP_SEED_MOCK", "MCP_SEED_MOCK_DATA"); seed != "" {
		cfg.SeedMockData = seed == "true" || seed == "1"
	}

	return cfg
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}
