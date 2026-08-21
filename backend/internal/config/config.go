package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv              string
	ServerPort          string
	DatabaseURL         string
	JWTSecret           string
	APIKeyEncryptionKey string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		ServerPort:          getEnv("SERVER_PORT", "8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		APIKeyEncryptionKey: os.Getenv("API_KEY_ENCRYPTION_KEY"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	if cfg.APIKeyEncryptionKey == "" {
		return Config{}, errors.New("API_KEY_ENCRYPTION_KEY is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
