package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	WSServerURL     string
	APIKey          string
	CryptoPanicKey  string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		WSServerURL:    os.Getenv("WS_SERVER_URL"),
		APIKey:         os.Getenv("WS_API_KEY"),
		CryptoPanicKey: os.Getenv("CRYPTOPANIC_API_KEY"),
	}

	if cfg.WSServerURL == "" {
		return nil, fmt.Errorf("WS_SERVER_URL не задан в .env")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("WS_API_KEY не задан в .env")
	}
	return cfg, nil
}
