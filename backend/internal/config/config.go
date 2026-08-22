package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                 string
	ServerPort             string
	DatabaseURL            string
	JWTSecret              string
	APIKeyEncryptionKey    string
	WorkspaceRoot          string
	WorkspaceSandboxImage  string
	WorkspaceDockerBinary  string
	WorkspacePodmanBinary  string
	WorkspaceNsJailBinary  string
	SchedulerWorkerEnabled bool
	SchedulerPollInterval  time.Duration
	ChannelSenderEnabled   bool
	ChannelSenderInterval  time.Duration
	ChannelSenderBatchSize int
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		ServerPort:             getEnv("SERVER_PORT", "8080"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		APIKeyEncryptionKey:    os.Getenv("API_KEY_ENCRYPTION_KEY"),
		WorkspaceRoot:          getEnv("WORKSPACE_ROOT", "./.workspaces"),
		WorkspaceSandboxImage:  getEnv("WORKSPACE_SANDBOX_IMAGE", "freedinner-agent-sandbox:latest"),
		WorkspaceDockerBinary:  getEnv("WORKSPACE_DOCKER_BINARY", "docker"),
		WorkspacePodmanBinary:  getEnv("WORKSPACE_PODMAN_BINARY", "podman"),
		WorkspaceNsJailBinary:  getEnv("WORKSPACE_NSJAIL_BINARY", "nsjail"),
		SchedulerWorkerEnabled: getBoolEnv("SCHEDULER_WORKER_ENABLED", true),
		SchedulerPollInterval:  getDurationEnv("SCHEDULER_POLL_INTERVAL", time.Minute),
		ChannelSenderEnabled:   getBoolEnv("CHANNEL_SENDER_ENABLED", true),
		ChannelSenderInterval:  getDurationEnv("CHANNEL_SENDER_INTERVAL", 15*time.Second),
		ChannelSenderBatchSize: getIntEnv("CHANNEL_SENDER_BATCH_SIZE", 20),
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

func getBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err == nil && parsed > 0 {
		return parsed
	}
	seconds, err := strconv.Atoi(value)
	if err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
