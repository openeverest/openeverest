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

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"k8s.io/client-go/rest"

	"github.com/openeverest/openeverest/v2/internal/agent"
)

const (
	readHeaderTimeout = 5 * time.Second
	idleTimeout       = 2 * time.Minute
	shutdownTimeout   = 10 * time.Second
)

type agentConfig struct {
	ListenAddress string `default:":9443" envconfig:"LISTEN_ADDRESS"`
	BearerToken   string `required:"true" envconfig:"BEARER_TOKEN"`
	TLSCertPath   string `required:"true" envconfig:"TLS_CERT_PATH"`
	TLSKeyPath    string `required:"true" envconfig:"TLS_KEY_PATH"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var config agentConfig
	if err := envconfig.Process("", &config); err != nil {
		log.Fatal(err)
	}

	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	proxy, err := agent.NewProxy(kubeConfig, config.BearerToken)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", proxy)

	server := &http.Server{
		Addr:              config.ListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("agent shutdown failed: %v", err)
		}
	}()

	log.Printf("OpenEverest agent listening on %s", config.ListenAddress)
	err = server.ListenAndServeTLS(config.TLSCertPath, config.TLSKeyPath)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
