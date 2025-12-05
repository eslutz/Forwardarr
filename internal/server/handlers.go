package server

import (
	"log/slog"
	"net/http"
)

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if !s.isRunning {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service not running"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.qbitClient.Ping(); err != nil {
		slog.Warn("readiness check failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("qBittorrent not reachable"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}
