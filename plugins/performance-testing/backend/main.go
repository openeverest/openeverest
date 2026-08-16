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

// Command main is the entrypoint for the performance-testing plugin
// backend, following the layout of openeverest/generic-plugin-template:
// plain net/http, GET /healthz, plugin API under /api/..., PORT env var
// for the listen port.
package main

import (
	"log"
	"net/http"
	"os"
)

func listenPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}

func main() {
	kube, err := newKubeClient()
	if err != nil {
		log.Fatalf("build kube client: %v", err)
	}

	s := &server{kube: kube, store: newMemoryStore()}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /api/runs", s.handleCreateRun)
	mux.HandleFunc("GET /api/runs", s.handleListRuns)
	mux.HandleFunc("GET /api/runs/{id}", s.handleGetRun)

	port := listenPort()
	log.Printf("performance-testing plugin backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil { //nolint:gosec // PoC backend; no read/write timeouts configured yet, matches generic-plugin-template's own reference main.go.
		log.Fatalf("server error: %v", err)
	}
}
