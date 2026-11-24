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

package clusterpreset

import (
	"bytes"
	"context"
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
	"github.com/openeverest/openeverest/v2/pkg/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
)

// ListOptions holds the inputs for the list command.
type ListOptions struct {
	Cluster  string
	Context  string
	Provider string
}

// ListRunner lists ClusterInstancePresets via the Everest API.
type ListRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewListRunner creates a new ListRunner.
func NewListRunner(cfg Config, l *zap.SugaredLogger) *ListRunner {
	lr := &ListRunner{config: cfg, l: l.With("component", "cluster-preset-list")}
	if cfg.Pretty {
		lr.l = zap.NewNop().Sugar()
	}
	return lr
}

// Run fetches cluster instance presets and prints them to stdout, either as a
// table (pretty mode) or as raw JSON/YAML.
func (lr *ListRunner) Run(ctx context.Context, opts ListOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: lr.config.Pretty}, lr.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	presets, err := lr.listPresets(ctx, c, opts)
	if err != nil {
		return err
	}

	sort.Slice(presets, func(i, j int) bool {
		return presetName(&presets[i]) < presetName(&presets[j])
	})

	var buf bytes.Buffer
	lr.render(&buf, presets)
	fmt.Fprint(os.Stdout, buf.String())
	return nil
}

func (lr *ListRunner) listPresets(ctx context.Context, c *client.ClientWithResponses, opts ListOptions) ([]client.ClusterInstancePreset, error) {
	params := &client.ListClusterInstancePresetsParams{}
	if opts.Provider != "" {
		params.Provider = &opts.Provider
	}
	resp, err := c.ListClusterInstancePresetsWithResponse(ctx, opts.Cluster, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster instance presets: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		if msg, ok := clienterr.Message(resp.JSONDefault); ok {
			return nil, fmt.Errorf("server error: %s", msg)
		}
		return nil, fmt.Errorf("unexpected response listing cluster instance presets: %s", resp.Status())
	}
	if resp.JSON200.Items == nil {
		return nil, nil
	}
	return *resp.JSON200.Items, nil
}

func (lr *ListRunner) render(w io.Writer, presets []client.ClusterInstancePreset) {
	if !lr.config.Pretty {
		_ = cli.Encode(w, presets, lr.config.YAML)
		return
	}
	printPresetTable(w, presets)
}

func printPresetTable(w io.Writer, presets []client.ClusterInstancePreset) {
	tbl := table.New("NAME", "PROVIDER", "VERSION", "AGE").WithWriter(w)
	for i := range presets {
		p := &presets[i]
		tbl.AddRow(
			presetName(p),
			presetProvider(p),
			presetVersion(p),
			presetAge(p),
		)
	}
	tbl.Print()
}

func presetName(p *client.ClusterInstancePreset) string {
	if p.Metadata == nil || p.Metadata.Name == "" {
		return "-"
	}
	return p.Metadata.Name
}

func presetProvider(p *client.ClusterInstancePreset) string {
	if p.Spec.ProviderRef.Name == "" {
		return "-"
	}
	return p.Spec.ProviderRef.Name
}

func presetVersion(p *client.ClusterInstancePreset) string {
	if p.Spec.Version == nil || *p.Spec.Version == "" {
		return "-"
	}
	return *p.Spec.Version
}

func presetAge(p *client.ClusterInstancePreset) string {
	if p.Metadata == nil || p.Metadata.CreationTimestamp.IsZero() {
		return "-"
	}
	return duration.ShortHumanDuration(time.Since(p.Metadata.CreationTimestamp.Time))
}
