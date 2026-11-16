// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

// Provider HTTP Server
//
// This file implements the HTTP server that providers run alongside their controller.
// The server exposes:
//
// 1. Health Endpoint (/healthz) - Kubernetes health check
// 2. Ready Endpoint (/readyz) - Kubernetes readiness check
//
// The server is integrated with the reconciler and runs in the same process.

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// =============================================================================
// SERVER CONFIGURATION
// =============================================================================

// ServerConfig configures the provider HTTP server.
type ServerConfig struct {
	// Port is the port to listen on (default: 8080)
	Port int

	// Deprecated: ValidationPath is ignored; the server serves no validation
	// endpoint.
	ValidationPath string

	// HealthPath is the path for health checks (default: /healthz)
	HealthPath string

	// ReadyPath is the path for readiness checks (default: /readyz)
	ReadyPath string

	// ReadTimeout is the maximum duration for reading the entire request (default: 10s)
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes (default: 10s)
	WriteTimeout time.Duration
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:         8080,
		HealthPath:   "/healthz",
		ReadyPath:    "/readyz",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

// =============================================================================
// PROVIDER SERVER
// =============================================================================

// Server is the HTTP server for a provider.
type Server struct {
	config ServerConfig

	server *http.Server
	ready  bool
	mu     sync.RWMutex
}

// NewServer creates a new provider server.
func NewServer(config ServerConfig) *Server {
	if config.Port == 0 {
		config.Port = 8080
	}
	if config.HealthPath == "" {
		config.HealthPath = "/healthz"
	}
	if config.ReadyPath == "" {
		config.ReadyPath = "/readyz"
	}

	return &Server{config: config}
}

// SetReady marks the server as ready to serve traffic.
func (s *Server) SetReady(ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = ready
}

// Start starts the HTTP server (blocking).
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Register endpoints
	mux.HandleFunc(s.config.HealthPath, s.handleHealth)
	mux.HandleFunc(s.config.ReadyPath, s.handleReady)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	logger := log.FromContext(ctx)
	logger.Info("Starting provider server", "port", s.config.Port)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		logger.Info("Shutting down provider server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// =============================================================================
// HTTP HANDLERS
// =============================================================================

// handleHealth returns 200 if the server is running.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleReady returns 200 if the server is ready to serve traffic.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	ready := s.ready
	s.mu.RUnlock()

	if ready {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("not ready"))
	}
}
