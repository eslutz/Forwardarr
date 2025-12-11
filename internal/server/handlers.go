package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/eslutz/forwardarr/pkg/version"
)

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if !s.isRunning {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("Service not running"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.qbitClient.Ping(); err != nil {
		slog.Warn("readiness check failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("qBittorrent not reachable"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Ready"))
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := struct {
		Status               string `json:"status"`
		Version              string `json:"version"`
		QBittorrentReachable bool   `json:"qbittorrent_reachable"`
	}{
		Status:               "running",
		Version:              version.Version,
		QBittorrentReachable: s.qbitClient.Ping() == nil,
	}

	if !s.isRunning {
		status.Status = "stopping"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		slog.Error("failed to encode status response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
