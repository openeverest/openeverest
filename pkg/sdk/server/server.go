package server

// Provider HTTP Server
//
// This file implements the HTTP server that providers run alongside their controller.
// The server exposes:
//
// 1. Schema Endpoint (/schema) - Returns OpenAPI schemas for components, topologies, global config
// 2. Validation Webhook (/validate) - Accepts admission review requests and validates Workloads
// 3. Health Endpoint (/healthz) - Kubernetes health check
// 4. Ready Endpoint (/readyz) - Kubernetes readiness check
//
// The server is integrated with the reconciler and runs in the same process.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openeverest/openeverest/v2/pkg/apis/v2alpha1"
)

// =============================================================================
// SERVER CONFIGURATION
// =============================================================================

// ServerConfig configures the provider HTTP server.
type ServerConfig struct {
	// Port is the port to listen on (default: 8080)
	Port int

	// SchemaPath is the path for the schema endpoint (default: /schema)
	SchemaPath string

	// ValidationPath is the path for the validation webhook (default: /validate)
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
		Port:           8080,
		SchemaPath:     "/schema",
		ValidationPath: "/validate",
		HealthPath:     "/healthz",
		ReadyPath:      "/readyz",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
	}
}

// =============================================================================
// PROVIDER SERVER
// =============================================================================

// ValidatorFunc is a function that validates a Workload.
// It receives the context, a Kubernetes client (for fetching related resources),
// and the Workload to validate.
// Return nil if validation passes, or an error with a user-friendly message.
type ValidatorFunc func(ctx context.Context, c client.Client, dc *v2alpha1.Workload) error

// Server is the HTTP server for a provider.
type Server struct {
	config    ServerConfig
	registry  *SchemaRegistry
	validator ValidatorFunc
	client    client.Client

	server *http.Server
	ready  bool
	mu     sync.RWMutex
}

// NewServer creates a new provider server.
func NewServer(config ServerConfig, registry *SchemaRegistry, validator ValidatorFunc) *Server {
	if config.Port == 0 {
		config.Port = 8080
	}
	if config.SchemaPath == "" {
		config.SchemaPath = "/schema"
	}
	if config.ValidationPath == "" {
		config.ValidationPath = "/validate"
	}
	if config.HealthPath == "" {
		config.HealthPath = "/healthz"
	}
	if config.ReadyPath == "" {
		config.ReadyPath = "/readyz"
	}

	return &Server{
		config:    config,
		registry:  registry,
		validator: validator,
	}
}

// SetClient sets the Kubernetes client (called by reconciler after manager is ready).
func (s *Server) SetClient(c client.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = c
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
	mux.HandleFunc(s.config.SchemaPath, s.handleSchema)
	mux.HandleFunc(s.config.ValidationPath, s.handleValidation)
	mux.HandleFunc(s.config.HealthPath, s.handleHealth)
	mux.HandleFunc(s.config.ReadyPath, s.handleReady)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	logger := log.FromContext(ctx)
	logger.Info("Starting provider server",
		"port", s.config.Port,
		"schemaPath", s.config.SchemaPath,
		"validationPath", s.config.ValidationPath,
	)

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

// handleSchema serves the OpenAPI schema for all registered types.
func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the OpenAPI document with all schemas
	doc := s.registry.AllSchemas()

	// Marshal to JSON
	data, err := doc.MarshalJSON()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate schema: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// handleValidation handles validation webhook requests.
// It expects a ValidationRequest and returns a ValidationResponse.
func (s *Server) handleValidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeValidationResponse(w, &ValidationResponse{
			Allowed: false,
			Message: fmt.Sprintf("Failed to parse request: %v", err),
		})
		return
	}

	// Get client
	s.mu.RLock()
	c := s.client
	s.mu.RUnlock()

	if c == nil {
		s.writeValidationResponse(w, &ValidationResponse{
			Allowed: false,
			Message: "Server not ready: client not initialized",
		})
		return
	}

	// Run validation
	ctx := r.Context()
	var validationErr error
	if s.validator != nil {
		validationErr = s.validator(ctx, c, &req.Object)
	}

	// Return response
	if validationErr != nil {
		s.writeValidationResponse(w, &ValidationResponse{
			Allowed: false,
			Message: validationErr.Error(),
		})
		return
	}

	s.writeValidationResponse(w, &ValidationResponse{
		Allowed: true,
	})
}

func (s *Server) writeValidationResponse(w http.ResponseWriter, resp *ValidationResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

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

// =============================================================================
// VALIDATION REQUEST/RESPONSE TYPES
// =============================================================================

// ValidationRequest is the request body for the validation webhook.
// This is a simplified version - in production you might use Kubernetes
// admission review types directly.
type ValidationRequest struct {
	// Object is the Workload being validated
	Object v2alpha1.Workload `json:"object"`

	// OldObject is the existing Workload (for UPDATE operations)
	// May be nil for CREATE operations
	OldObject *v2alpha1.Workload `json:"oldObject,omitempty"`

	// Operation is the operation being performed (CREATE, UPDATE, DELETE)
	Operation string `json:"operation,omitempty"`
}

// ValidationResponse is the response body for the validation webhook.
type ValidationResponse struct {
	// Allowed indicates whether the request is allowed
	Allowed bool `json:"allowed"`

	// Message is the reason for denial (if not allowed)
	Message string `json:"message,omitempty"`

	// Warnings are non-blocking warnings to return to the user
	Warnings []string `json:"warnings,omitempty"`
}
