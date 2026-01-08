package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eslutz/forwardarr/internal/config"
	"github.com/eslutz/forwardarr/internal/qbit"
	"github.com/eslutz/forwardarr/internal/server"
	"github.com/eslutz/forwardarr/internal/sync"
	"github.com/eslutz/forwardarr/internal/webhook"
	_ "github.com/eslutz/forwardarr/pkg/version"
)

func main() {
	cfg := config.Load()
	setupLogging(cfg.LogLevel)

	slog.Info("starting forwardarr",
		"gluetun_port_file", cfg.GluetunPortFile,
		"qbit_addr", cfg.QbitAddr,
		"sync_interval", cfg.SyncInterval,
		"metrics_port", cfg.MetricsPort,
		"webhook_enabled", cfg.WebhookEnabled,
	)

	qbitClient, err := qbit.NewClient(cfg.QbitAddr, cfg.QbitUser, cfg.QbitPass)
	if err != nil {
		slog.Error("failed to create qBittorrent client", "error", err)
		os.Exit(1)
	}

	var webhookClient *webhook.Client
	if cfg.WebhookEnabled {
		webhookClient = webhook.NewClient(
			cfg.WebhookURL,
			cfg.WebhookTimeout,
			webhook.Template(cfg.WebhookTemplate),
			cfg.WebhookEvents,
		)
		slog.Info("webhook notifications enabled",
			"url", cfg.WebhookURL,
			"timeout", cfg.WebhookTimeout,
			"template", cfg.WebhookTemplate,
			"events", cfg.WebhookEvents,
		)
	}

	watcher, err := sync.NewWatcher(cfg.GluetunPortFile, qbitClient, webhookClient, cfg.SyncInterval)
	if err != nil {
		slog.Error("failed to create file watcher", "error", err)
		os.Exit(1)
	}

	srv := server.NewServer(cfg.MetricsPort, qbitClient)

	// Start HTTP server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start watcher in goroutine
	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- watcher.Start()
	}()

	// Wait for shutdown signal or watcher error
	select {
	case <-ctx.Done():
		slog.Info("received shutdown signal, gracefully stopping...")

		// Give time for cleanup
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}

		slog.Info("shutdown complete")

	case err := <-watcherDone:
		if err != nil {
			slog.Error("watcher failed", "error", err)
			os.Exit(1)
		}
	}
}

func setupLogging(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
}
