// everest
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

// Package provider provides CLI business logic for provider management.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/rodaine/table"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
)

// Config holds the shared configuration for provider CLI runners.
type Config struct {
	Pretty bool
}

// ListOptions holds the inputs for the list command.
type ListOptions struct {
	Cluster string
	Context string
}

// ListRunner lists providers via the Everest API.
type ListRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewListRunner creates a new ListRunner.
func NewListRunner(cfg Config, l *zap.SugaredLogger) *ListRunner {
	pl := &ListRunner{config: cfg, l: l.With("component", "provider-list")}
	if cfg.Pretty {
		pl.l = zap.NewNop().Sugar()
	}
	return pl
}

// Run fetches providers and prints them to stdout, either as a table (pretty mode)
// or as raw JSON.
func (pl *ListRunner) Run(ctx context.Context, opts ListOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: pl.config.Pretty}, pl.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	providers, err := pl.listProviders(ctx, c, opts.Cluster)
	if err != nil {
		return err
	}

	sort.Slice(providers, func(i, j int) bool {
		return providerName(&providers[i]) < providerName(&providers[j])
	})

	var buf bytes.Buffer
	pl.render(&buf, providers)
	fmt.Fprint(os.Stdout, buf.String())
	return nil
}

func (pl *ListRunner) listProviders(ctx context.Context, c *client.ClientWithResponses, cluster string) ([]client.Provider, error) {
	resp, err := c.ListProvidersWithResponse(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response listing providers: %s", resp.Status())
	}
	if resp.JSON200.Items == nil {
		return nil, nil
	}
	return *resp.JSON200.Items, nil
}

func (pl *ListRunner) render(w io.Writer, providers []client.Provider) {
	if !pl.config.Pretty {
		_ = json.NewEncoder(w).Encode(providers) //nolint:errchkjson
		return
	}
	printProviderTable(w, providers)
}

func printProviderTable(w io.Writer, providers []client.Provider) {
	tbl := table.New("NAME", "AGE").WithWriter(w)
	for i := range providers {
		prov := &providers[i]
		tbl.AddRow(
			providerName(prov),
			providerAge(prov),
		)
	}
	tbl.Print()
}

func providerName(prov *client.Provider) string {
	if prov.Metadata == nil || prov.Metadata.Name == "" {
		return "-"
	}
	return prov.Metadata.Name
}

func providerAge(prov *client.Provider) string {
	if prov.Metadata == nil || prov.Metadata.CreationTimestamp.IsZero() {
		return "-"
	}
	return duration.ShortHumanDuration(time.Since(prov.Metadata.CreationTimestamp.Time))
}
