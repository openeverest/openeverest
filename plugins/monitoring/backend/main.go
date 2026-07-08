// This is an EXAMPLE backend written in Go. The plugin backend is
// language-agnostic — you can replace this entire directory with any HTTP
// server (Python, Node.js, Rust, Java, etc.) as long as it:
//
//  1. Serves GET /main.js        → returns the frontend bundle
//  2. Serves GET /healthz        → returns 200 OK
//  3. Handles your API routes    → under /api/...
//  4. Reads X-Everest-User header → authenticated user JWT
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// dist/main.js is copied from the frontend build during the Docker build.
//
//go:embed dist/main.js
var distFS embed.FS

//go:embed dist/icon.png
var iconData []byte

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

func everestAPIURL() string {
	// Explicit override wins (useful for local dev or air-gapped setups).
	if v := os.Getenv("EVEREST_API_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}

	// Kubernetes automatically injects <SVCNAME>_SERVICE_HOST / _SERVICE_PORT
	// for every Service in the same namespace. Look for the "everest" service.
	host := os.Getenv("EVEREST_SERVICE_HOST")
	port := os.Getenv("EVEREST_SERVICE_PORT")
	if host != "" && port != "" {
		return fmt.Sprintf("http://%s:%s", host, port)
	}

	// Fallback to the conventional in-cluster DNS name.
	return "http://everest-server.everest-system.svc.cluster.local:8080"
}

func listenPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON error: %v", err)
	}
}

func apiError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// GET /api/hello — example endpoint.
func handleHello(w http.ResponseWriter, r *http.Request) {
	// The X-Everest-User header contains the JWT of the authenticated user.
	// Use it when calling the OpenEverest API on behalf of the user.
	_ = r.Header.Get("X-Everest-User")

	writeJSON(w, map[string]string{
		"message": "Hello from my-plugin backend!",
	})
}

// GET /healthz — liveness/readiness probe.
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// GET /main.js — serves the frontend bundle.
func handleBundle(w http.ResponseWriter, _ *http.Request) {
	data, err := distFS.ReadFile("dist/main.js")
	if err != nil {
		http.Error(w, "bundle not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(data)
}

// GET /icon.png — serves the plugin icon.
func handleIcon(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(iconData)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	mux := http.NewServeMux()

	// Frontend bundle.
	mux.HandleFunc("GET /main.js", handleBundle)

	// Plugin icon.
	mux.HandleFunc("GET /icon.png", handleIcon)

	// Health check.
	mux.HandleFunc("GET /healthz", handleHealthz)

	// Plugin API routes — add your handlers here.
	mux.HandleFunc("GET /api/hello", handleHello)

	port := listenPort()
	log.Printf("my-plugin backend listening on :%s (everest API: %s)", port, everestAPIURL())
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
