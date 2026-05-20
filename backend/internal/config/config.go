package config

type Config struct {
	Environment string
	Mode        string
}

func NewConfig() Config {
	return Config{
		Environment: "development",
		Mode:        "debug",
	}
}
