package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/eslutz/gluetarr/internal/qbit"
)

type Server struct {
	port       string
	qbitClient *qbit.Client
	isRunning  bool
	server     *http.Server
}

func NewServer(port string, qbitClient *qbit.Client) *Server {
	return &Server{
		port:       port,
		qbitClient: qbitClient,
		isRunning:  true,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":" + s.port
	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	slog.Info("starting http server", "address", addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) SetRunning(running bool) {
	s.isRunning = running
}
