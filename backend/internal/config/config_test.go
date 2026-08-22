package config

import (
	"testing"
	"time"
)

func TestLoadRequiresCoreSecrets(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("API_KEY_ENCRYPTION_KEY", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected missing DATABASE_URL to fail")
	}
}

func TestLoadUsesDefaultsAndEnvOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://freedinner:secret@localhost:5432/freedinner_agent")
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("API_KEY_ENCRYPTION_KEY", "encryption-secret")
	t.Setenv("APP_ENV", "test")
	t.Setenv("SERVER_PORT", "18080")
	t.Setenv("WORKSPACE_ROOT", "/tmp/freedinner-test-workspaces")
	t.Setenv("SCHEDULER_WORKER_ENABLED", "false")
	t.Setenv("SCHEDULER_POLL_INTERVAL", "30s")
	t.Setenv("CHANNEL_SENDER_ENABLED", "false")
	t.Setenv("CHANNEL_SENDER_INTERVAL", "45")
	t.Setenv("CHANNEL_SENDER_BATCH_SIZE", "7")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppEnv != "test" {
		t.Fatalf("AppEnv = %q", cfg.AppEnv)
	}
	if cfg.ServerPort != "18080" {
		t.Fatalf("ServerPort = %q", cfg.ServerPort)
	}
	if cfg.WorkspaceRoot != "/tmp/freedinner-test-workspaces" {
		t.Fatalf("WorkspaceRoot = %q", cfg.WorkspaceRoot)
	}
	if cfg.SchedulerWorkerEnabled {
		t.Fatal("SchedulerWorkerEnabled should be false")
	}
	if cfg.SchedulerPollInterval != 30*time.Second {
		t.Fatalf("SchedulerPollInterval = %s", cfg.SchedulerPollInterval)
	}
	if cfg.ChannelSenderEnabled {
		t.Fatal("ChannelSenderEnabled should be false")
	}
	if cfg.ChannelSenderInterval != 45*time.Second {
		t.Fatalf("ChannelSenderInterval = %s", cfg.ChannelSenderInterval)
	}
	if cfg.ChannelSenderBatchSize != 7 {
		t.Fatalf("ChannelSenderBatchSize = %d", cfg.ChannelSenderBatchSize)
	}
}

func TestEnvParsersFallbackOnInvalidValues(t *testing.T) {
	t.Setenv("BOOL_VALUE", "not-bool")
	t.Setenv("INT_VALUE", "not-int")
	t.Setenv("DURATION_VALUE", "-3s")

	if !getBoolEnv("BOOL_VALUE", true) {
		t.Fatal("invalid bool should use fallback")
	}
	if getIntEnv("INT_VALUE", 12) != 12 {
		t.Fatal("invalid int should use fallback")
	}
	if getDurationEnv("DURATION_VALUE", time.Minute) != time.Minute {
		t.Fatal("invalid duration should use fallback")
	}
}
