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

package preset

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
	Namespace     string
	AllNamespaces bool
	Cluster       string
	Context       string
	Provider      string
}

// ListRunner lists InstancePresets via the Everest API.
type ListRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewListRunner creates a new ListRunner.
func NewListRunner(cfg Config, l *zap.SugaredLogger) *ListRunner {
	lr := &ListRunner{config: cfg, l: l.With("component", "preset-list")}
	if cfg.Pretty {
		lr.l = zap.NewNop().Sugar()
	}
	return lr
}

// Run fetches instance presets and prints them to stdout, either as a table
// (pretty mode) or as raw JSON/YAML.
func (lr *ListRunner) Run(ctx context.Context, opts ListOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: lr.config.Pretty}, lr.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	namespaces := []string{opts.Namespace}
	if opts.AllNamespaces {
		if namespaces, err = lr.listNamespaces(ctx, c, opts.Cluster); err != nil {
			return err
		}
	}

	params := &client.ListInstancePresetsParams{}
	if opts.Provider != "" {
		params.Provider = &opts.Provider
	}

	presets := make([]client.InstancePreset, 0, len(namespaces))
	for _, ns := range namespaces {
		items, err := lr.listPresets(ctx, c, opts.Cluster, ns, params)
		if err != nil {
			return err
		}
		presets = append(presets, items...)
	}

	sort.Slice(presets, func(i, j int) bool {
		ni, nj := presetNamespace(&presets[i]), presetNamespace(&presets[j])
		if ni != nj {
			return ni < nj
		}
		return presetName(&presets[i]) < presetName(&presets[j])
	})

	var buf bytes.Buffer
	lr.render(&buf, presets)
	fmt.Fprint(os.Stdout, buf.String())
	return nil
}

func (lr *ListRunner) listNamespaces(ctx context.Context, c *client.ClientWithResponses, cluster string) ([]string, error) {
	resp, err := c.ListNamespacesWithResponse(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response listing namespaces: %s", resp.Status())
	}
	return *resp.JSON200, nil
}

func (lr *ListRunner) listPresets(
	ctx context.Context, c *client.ClientWithResponses, cluster, namespace string, params *client.ListInstancePresetsParams,
) ([]client.InstancePreset, error) {
	resp, err := c.ListInstancePresetsWithResponse(ctx, cluster, namespace, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list instance presets in namespace %q: %w", namespace, err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		if msg, ok := clienterr.Message(resp.JSONDefault); ok {
			return nil, fmt.Errorf("server error: %s", msg)
		}
		return nil, fmt.Errorf("unexpected response listing instance presets in namespace %q: %s", namespace, resp.Status())
	}
	if resp.JSON200.Items == nil {
		return nil, nil
	}
	return *resp.JSON200.Items, nil
}

func (lr *ListRunner) render(w io.Writer, presets []client.InstancePreset) {
	if !lr.config.Pretty {
		_ = cli.Encode(w, presets, lr.config.YAML)
		return
	}
	printPresetTable(w, presets)
}

func printPresetTable(w io.Writer, presets []client.InstancePreset) {
	tbl := table.New("NAME", "NAMESPACE", "PROVIDER", "VERSION", "AGE").WithWriter(w)
	for i := range presets {
		p := &presets[i]
		tbl.AddRow(
			presetName(p),
			presetNamespace(p),
			presetProvider(p),
			presetVersion(p),
			presetAge(p),
		)
	}
	tbl.Print()
}

func presetName(p *client.InstancePreset) string {
	if p.Metadata == nil || p.Metadata.Name == "" {
		return "-"
	}
	return p.Metadata.Name
}

func presetNamespace(p *client.InstancePreset) string {
	if p.Metadata == nil || p.Metadata.Namespace == "" {
		return "-"
	}
	return p.Metadata.Namespace
}

func presetProvider(p *client.InstancePreset) string {
	if p.Spec.ProviderRef.Name == "" {
		return "-"
	}
	return p.Spec.ProviderRef.Name
}

func presetVersion(p *client.InstancePreset) string {
	if p.Spec.Version == nil || *p.Spec.Version == "" {
		return "-"
	}
	return *p.Spec.Version
}

func presetAge(p *client.InstancePreset) string {
	if p.Metadata == nil || p.Metadata.CreationTimestamp.IsZero() {
		return "-"
	}
	return duration.ShortHumanDuration(time.Since(p.Metadata.CreationTimestamp.Time))
}
