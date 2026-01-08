package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GluetunPortFile   string
	QbitAddr          string
	QbitUser          string
	QbitPass          string
	SyncInterval      time.Duration
	MetricsPort       string
	LogLevel          string
	WebhookURL        string
	WebhookEnabled    bool
	WebhookTimeout    time.Duration
	WebhookTemplate   string
	WebhookEvents     []string
}

func Load() *Config {
	webhookURL := getEnv("WEBHOOK_URL", "")
	webhookEvents := getEnv("WEBHOOK_EVENTS", "port_changed")
	return &Config{
		GluetunPortFile: getEnv("GLUETUN_PORT_FILE", "/tmp/gluetun/forwarded_port"),
		QbitAddr:        getEnv("TORRENT_CLIENT_URL", "http://localhost:8080"),
		QbitUser:        getEnv("TORRENT_CLIENT_USER", "admin"),
		QbitPass:        getEnv("TORRENT_CLIENT_PASSWORD", "adminadmin"),
		SyncInterval:    getDurationEnv("SYNC_INTERVAL", 5*time.Minute),
		MetricsPort:     getEnv("METRICS_PORT", "9090"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		WebhookURL:      webhookURL,
		WebhookEnabled:  webhookURL != "",
		WebhookTimeout:  getDurationEnv("WEBHOOK_TIMEOUT", 10*time.Second),
		WebhookTemplate: getEnv("WEBHOOK_TEMPLATE", "json"),
		WebhookEvents:   parseEvents(webhookEvents),
	}
}

func parseEvents(events string) []string {
	if events == "" {
		return []string{"port_changed"}
	}
	parts := strings.Split(events, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}
