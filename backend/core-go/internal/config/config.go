package config

type Config struct {
	HTTPHost string
	HTTPPort int
}

func Load() Config {
	return Config{
		HTTPHost: "0.0.0.0",
		HTTPPort: 8080,
	}
}
