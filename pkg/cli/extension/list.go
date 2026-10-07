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

// Package extension provides CLI operations for managing generic plugins.
package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rodaine/table"
	"go.uber.org/zap"

	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// ListConfig holds configuration for the extension list operation.
type ListConfig struct {
	KubeconfigPath string
	Pretty         bool
}

// PluginInfo contains information about an installed extension.
type PluginInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	BackendURL  string `json:"backendUrl"`
	Enabled     bool   `json:"enabled"`
}

// PluginLister lists installed extensions.
type PluginLister struct {
	cfg        ListConfig
	kubeClient kubernetes.KubernetesConnector
	l          *zap.SugaredLogger
}

// NewPluginLister creates a new PluginLister.
func NewPluginLister(cfg ListConfig, l *zap.SugaredLogger) (*PluginLister, error) {
	pl := &PluginLister{
		cfg: cfg,
		l:   l.With("component", "plugin-lister"),
	}
	if cfg.Pretty {
		pl.l = zap.NewNop().Sugar()
	}

	k, err := cliutils.NewKubeConnector(pl.l, pl.cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	pl.kubeClient = k
	return pl, nil
}

// Run lists all extensions and prints them to stdout, either as a table (pretty mode)
// or as raw JSON.
func (pl *PluginLister) Run(ctx context.Context) error {
	plugins, err := pl.kubeClient.ListPlugins(ctx)
	if err != nil {
		return fmt.Errorf("cannot list plugins: %w", err)
	}

	items := make([]PluginInfo, 0, len(plugins.Items))
	for _, p := range plugins.Items {
		items = append(items, PluginInfo{
			Name:        p.Name,
			DisplayName: p.Spec.DisplayName,
			BackendURL:  formatBackendURL(&p.Spec),
			Enabled:     p.Spec.Enabled,
		})
	}

	pl.render(os.Stdout, items)
	return nil
}

func formatBackendURL(spec *extensionsv1alpha1.PluginSpec) string {
	if spec == nil || spec.Backend == nil {
		return ""
	}
	if spec.Backend.ServiceRef != nil {
		ref := spec.Backend.ServiceRef
		return fmt.Sprintf("%s.%s:%d (in-cluster)", ref.Name, ref.Namespace, ref.Port)
	}
	return spec.Backend.ExternalURL
}

func (pl *PluginLister) render(w io.Writer, plugins []PluginInfo) {
	if !pl.cfg.Pretty {
		if plugins == nil {
			plugins = []PluginInfo{}
		}
		_ = json.NewEncoder(w).Encode(plugins) //nolint:errchkjson
		return
	}
	printPluginsTable(w, plugins)
}

func printPluginsTable(w io.Writer, plugins []PluginInfo) {
	tbl := table.New("NAME", "DISPLAY NAME", "BACKEND URL", "ENABLED").WithWriter(w)
	tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
		return strings.ToUpper(fmt.Sprintf(format, vals...))
	})

	for _, p := range plugins {
		tbl.AddRow(p.Name, p.DisplayName, p.BackendURL, p.Enabled)
	}

	tbl.Print()
}
