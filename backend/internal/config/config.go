package config

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/zlylong/ops-mcp/backend/internal/domain"
)

type Config struct {
	Addr        string             `json:"addr"`
	Mode        string             `json:"mode"`
	Environment domain.Environment `json:"environment"`
	DatabaseURL string             `json:"databaseUrl"`
	RedisURL    string             `json:"redisUrl"`
	ConfigFile  string             `json:"-"`
}

func Default() Config {
	return Config{Addr: ":8080", Mode: "mock", Environment: domain.EnvDevelopment}
}

func Load() (Config, error) {
	cfg := Default()
	path := os.Getenv("OPS_MCP_CONFIG")
	if path == "" {
		path = flagConfigPath()
	}
	if path != "" {
		loaded, err := LoadFile(path)
		if err != nil {
			return Config{}, err
		}
		cfg = merge(cfg, loaded)
		cfg.ConfigFile = path
	}
	applyEnv(&cfg)
	return cfg, nil
}

func LoadFile(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		return Config{}, errors.New("config path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func flagConfigPath() string {
	for i, arg := range os.Args {
		if arg == "--config" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(arg, "--config=") {
			return strings.TrimPrefix(arg, "--config=")
		}
	}
	return ""
}

func merge(base Config, override Config) Config {
	if override.Addr != "" {
		base.Addr = override.Addr
	}
	if override.Mode != "" {
		base.Mode = override.Mode
	}
	if override.Environment != "" {
		base.Environment = override.Environment
	}
	if override.DatabaseURL != "" {
		base.DatabaseURL = override.DatabaseURL
	}
	if override.RedisURL != "" {
		base.RedisURL = override.RedisURL
	}
	return base
}

func applyEnv(cfg *Config) {
	if value := os.Getenv("OPS_MCP_ADDR"); value != "" {
		cfg.Addr = value
	}
	if value := os.Getenv("OPS_MCP_MODE"); value != "" {
		cfg.Mode = value
	}
	if value := os.Getenv("OPS_MCP_ENV"); value != "" {
		cfg.Environment = domain.Environment(value)
	}
	if value := os.Getenv("DATABASE_URL"); value != "" {
		cfg.DatabaseURL = value
	}
	if value := os.Getenv("REDIS_URL"); value != "" {
		cfg.RedisURL = value
	}
}
