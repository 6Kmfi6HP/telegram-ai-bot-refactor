package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPort         = 8080
	DefaultAPIKey       = "12345"
	DefaultModel        = "smart"
	DefaultAPIURL       = "http://127.0.0.1:12345/v1"
	DefaultReadTimeout  = 60 * time.Second
	DefaultWriteTimeout = 60 * time.Second
	DefaultIdleTimeout  = 120 * time.Second
)

// Config contains all runtime configuration. The application loads it from
// environment variables only.
type Config struct {
	TelegramToken string
	WebhookURL    string
	Port          int
	APIKey        string
	DefaultModel  string
	APIURL        string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
}

// LoadFromEnv reads configuration from environment variables.
func LoadFromEnv() (Config, error) {
	cfg := Config{
		TelegramToken: strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		WebhookURL:    strings.TrimSpace(os.Getenv("TELEGRAM_WEBHOOK_URL")),
		Port:          envInt("PORT", DefaultPort),
		APIKey:        envString("AI_API_KEY", DefaultAPIKey),
		DefaultModel:  envString("AI_MODEL", DefaultModel),
		APIURL:        envString("AI_API_URL", DefaultAPIURL),
		ReadTimeout:   envDuration("HTTP_READ_TIMEOUT", DefaultReadTimeout),
		WriteTimeout:  envDuration("HTTP_WRITE_TIMEOUT", DefaultWriteTimeout),
		IdleTimeout:   envDuration("HTTP_IDLE_TIMEOUT", DefaultIdleTimeout),
	}

	if cfg.TelegramToken == "" {
		return Config{}, fmt.Errorf("请通过环境变量 TELEGRAM_BOT_TOKEN 设置 Bot Token")
	}
	if cfg.WebhookURL == "" {
		return Config{}, fmt.Errorf("请通过环境变量 TELEGRAM_WEBHOOK_URL 设置 webhook URL")
	}
	if cfg.APIKey == "" {
		return Config{}, fmt.Errorf("请提供 API 密钥：AI_API_KEY")
	}
	if cfg.DefaultModel == "" {
		return Config{}, fmt.Errorf("请提供默认模型：AI_MODEL")
	}
	if cfg.APIURL == "" {
		return Config{}, fmt.Errorf("请提供 API URL：AI_API_URL")
	}

	return cfg, nil
}

func envString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
