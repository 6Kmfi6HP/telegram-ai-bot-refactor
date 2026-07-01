package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadFromEnvPreservesOriginalFlags(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{
		"bot",
		"-token", "telegram-token",
		"-webhook", "https://example.test/webhook",
		"-port", "9090",
		"-key", "api-key",
		"-model", "gpt-test",
		"-url", "http://ai.test/v1",
	}
	t.Cleanup(func() { os.Args = oldArgs })

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if cfg.TelegramToken != "telegram-token" {
		t.Fatalf("TelegramToken=%q, want telegram-token", cfg.TelegramToken)
	}
	if cfg.WebhookURL != "https://example.test/webhook" {
		t.Fatalf("WebhookURL=%q, want https://example.test/webhook", cfg.WebhookURL)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port=%d, want 9090", cfg.Port)
	}
	if cfg.APIKey != "api-key" {
		t.Fatalf("APIKey=%q, want api-key", cfg.APIKey)
	}
	if cfg.DefaultModel != "gpt-test" {
		t.Fatalf("DefaultModel=%q, want gpt-test", cfg.DefaultModel)
	}
	if cfg.APIURL != "http://ai.test/v1" {
		t.Fatalf("APIURL=%q, want http://ai.test/v1", cfg.APIURL)
	}
}

func TestLoadFromEnvRequiresOriginalTokenFlag(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"bot", "-webhook", "https://example.test/webhook"}
	t.Cleanup(func() { os.Args = oldArgs })
	t.Setenv("TELEGRAM_BOT_TOKEN", "env-token")

	_, err := LoadFromEnv()
	if err == nil || !strings.Contains(err.Error(), "-token") {
		t.Fatalf("error=%v, want missing -token", err)
	}
}

func TestLoadFromEnvAllowsMissingWebhookUntilStartup(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"bot", "-token", "telegram-token"}
	t.Cleanup(func() { os.Args = oldArgs })

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}
	if cfg.WebhookURL != "" {
		t.Fatalf("WebhookURL=%q, want empty", cfg.WebhookURL)
	}
}
