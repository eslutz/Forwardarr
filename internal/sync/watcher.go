package sync

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/eslutz/gluetarr/internal/qbit"
)

type Watcher struct {
	portFile     string
	qbitClient   *qbit.Client
	syncInterval time.Duration
	lastPort     int
	watcher      *fsnotify.Watcher
}

func NewWatcher(portFile string, qbitClient *qbit.Client, syncInterval time.Duration) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	w := &Watcher{
		portFile:     portFile,
		qbitClient:   qbitClient,
		syncInterval: syncInterval,
		watcher:      watcher,
	}

	dir := filepath.Dir(portFile)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch directory %s: %w", dir, err)
	}

	slog.Info("watching for port file changes", "directory", dir, "file", portFile)
	return w, nil
}

func (w *Watcher) Start() error {
	ticker := time.NewTicker(w.syncInterval)
	defer ticker.Stop()
	defer w.watcher.Close()

	if err := w.syncPort(); err != nil {
		slog.Warn("initial sync failed", "error", err)
	}

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher channel closed")
			}

			if event.Name == w.portFile && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				slog.Debug("port file changed", "event", event.Op.String())
				if err := w.syncPort(); err != nil {
					slog.Error("failed to sync port after file change", "error", err)
					IncrementSyncErrors()
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			slog.Error("file watcher error", "error", err)

		case <-ticker.C:
			slog.Debug("periodic sync triggered")
			if err := w.syncPort(); err != nil {
				slog.Warn("periodic sync failed", "error", err)
			}
		}
	}
}

func (w *Watcher) syncPort() error {
	gluetunPort, err := w.readPortFromFile()
	if err != nil {
		return fmt.Errorf("failed to read Gluetun port: %w", err)
	}

	qbitPort, err := w.qbitClient.GetPort()
	if err != nil {
		return fmt.Errorf("failed to get qBittorrent port: %w", err)
	}

	slog.Debug("port status", "gluetun_port", gluetunPort, "qbit_port", qbitPort)

	if gluetunPort != qbitPort {
		slog.Info("port mismatch detected, updating...", "old_port", qbitPort, "new_port", gluetunPort)
		if err := w.qbitClient.SetPort(gluetunPort); err != nil {
			IncrementSyncErrors()
			return fmt.Errorf("failed to set qBittorrent port: %w", err)
		}

		w.lastPort = gluetunPort
		SetCurrentPort(gluetunPort)
		IncrementSyncTotal()
		UpdateLastSyncTimestamp()
	} else {
		slog.Debug("ports are in sync", "port", gluetunPort)
	}

	return nil
}

func (w *Watcher) readPortFromFile() (int, error) {
	content, err := os.ReadFile(w.portFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read port file: %w", err)
	}

	portStr := strings.TrimSpace(string(content))
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port value: %s", portStr)
	}

	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port out of valid range: %d", port)
	}

	return port, nil
}
