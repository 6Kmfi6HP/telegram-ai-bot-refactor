package config

import (
	"flag"
	"fmt"
	"os"
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

// Config contains all runtime configuration loaded from the original CLI flags.
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

// LoadFromEnv reads configuration from the original CLI flags.
func LoadFromEnv() (Config, error) {
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	token := flags.String("token", "", "Telegram Bot Token")
	webhook := flags.String("webhook", "", "Webhook URL")
	port := flags.Int("port", DefaultPort, "webhook服务器端口")
	apiKey := flags.String("key", DefaultAPIKey, "API密钥")
	defaultModel := flags.String("model", DefaultModel, "指定默认模型")
	apiURL := flags.String("url", DefaultAPIURL, "API URL")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return Config{}, err
	}

	cfg := Config{
		TelegramToken: strings.TrimSpace(*token),
		WebhookURL:    strings.TrimSpace(*webhook),
		Port:          *port,
		APIKey:        strings.TrimSpace(*apiKey),
		DefaultModel:  strings.TrimSpace(*defaultModel),
		APIURL:        strings.TrimSpace(*apiURL),
		ReadTimeout:   DefaultReadTimeout,
		WriteTimeout:  DefaultWriteTimeout,
		IdleTimeout:   DefaultIdleTimeout,
	}

	if cfg.APIKey == "" {
		return Config{}, fmt.Errorf("请提供API密钥")
	}
	if cfg.TelegramToken == "" {
		return Config{}, fmt.Errorf("请通过-token参数设置Bot Token")
	}
	return cfg, nil
}
