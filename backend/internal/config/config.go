package config

import "os"

type Config struct {
	Addr        string
	Mode        string
	Environment string
	DatabaseURL string
	RedisURL    string
}

func FromEnv() Config {
	return Config{
		Addr:        env("OPS_MCP_ADDR", ":8080"),
		Mode:        env("OPS_MCP_MODE", "mock"),
		Environment: env("OPS_MCP_ENV", "development"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
