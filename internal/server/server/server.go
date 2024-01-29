package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// NewServer creates a new server.
func NewServer(cfg Config, svc fileService, log *zap.Logger) *Server {
	return &Server{
		fileSvc: svc,
		log:     log,
		cfg:     cfg,
	}
}

// Server is the HTTP server.
type Server struct {
	fileSvc fileService
	log     *zap.Logger
	cfg     Config
}

// Config is the configuration for the server.
type Config struct {
	Address    string `envconfig:"HTTP_ADDRESS" default:":8080"`
	TimeoutSec int    `envconfig:"HTTP_TIMEOUT_SEC" default:"10"`
}

func Router(s *Server) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/file/{index}", s.DownloadFile).Methods("GET")
	r.HandleFunc("/file", s.UploadMultiple).Methods("POST")
	r.HandleFunc("/metrics", promhttp.Handler().ServeHTTP).Methods("GET")
	return r
}

// StartServer starts the HTTP server.
func (s *Server) StartServer(r *mux.Router) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	timeout := time.Duration(s.cfg.TimeoutSec) * time.Second
	srv := &http.Server{
		Handler:      r,
		Addr:         s.cfg.Address,
		WriteTimeout: timeout,
		ReadTimeout:  timeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Fatal("Could not listen on", zap.String("address", s.cfg.Address), zap.Error(err))
		}
	}()

	s.log.Info("Server is ready to handle requests", zap.String("address", s.cfg.Address))
	select {
	case killSignal := <-interrupt:
		switch killSignal {
		case os.Interrupt:
			s.log.Debug("Got SIGINT...")
		case syscall.SIGTERM:
			s.log.Debug("Got SIGTERM...")
		}
	}

	s.log.Info("The service is shutting down...")
	if err := srv.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("could not gracefully shutdown the server: %w", err)
	}
	s.log.Info("Server stopped")
	return nil
}
