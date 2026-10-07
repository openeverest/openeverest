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
	"sort"
	"strings"

	"github.com/rodaine/table"
	"go.uber.org/zap"

	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// ListConfig holds configuration for the extension list operation.
type ListConfig struct {
	KubeconfigPath string
	Pretty         bool
}

// Info contains information about an installed extension.
type Info struct {
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

// Run lists all extensions and prints them according to the configured format.
func (pl *PluginLister) Run(ctx context.Context) error {
	plugins, err := pl.kubeClient.ListPlugins(ctx)
	if err != nil {
		return fmt.Errorf("cannot list plugins: %w", err)
	}

	extensions := make([]Info, 0, len(plugins.Items))
	for _, p := range plugins.Items {
		backendURL := ""
		if p.Spec.Backend != nil {
			if p.Spec.Backend.ServiceRef != nil {
				ref := p.Spec.Backend.ServiceRef
				backendURL = fmt.Sprintf("%s.%s:%d (in-cluster)", ref.Name, ref.Namespace, ref.Port)
			} else {
				backendURL = p.Spec.Backend.ExternalURL
			}
		}
		extensions = append(extensions, Info{
			Name:        p.Name,
			DisplayName: p.Spec.DisplayName,
			BackendURL:  backendURL,
			Enabled:     p.Spec.Enabled,
		})
	}

	sort.Slice(extensions, func(i, j int) bool {
		return extensions[i].Name < extensions[j].Name
	})

	pl.Render(os.Stdout, extensions)
	return nil
}

// Render formats extensions to w as either JSON or an ASCII table based on cfg.Pretty.
func (pl *PluginLister) Render(w io.Writer, extensions []Info) {
	if !pl.cfg.Pretty {
		if extensions == nil {
			extensions = []Info{}
		}
		_ = json.NewEncoder(w).Encode(extensions) //nolint:errchkjson
		return
	}
	printExtensionTable(w, extensions)
}

func printExtensionTable(w io.Writer, extensions []Info) {
	tbl := table.New("NAME", "DISPLAY NAME", "BACKEND URL", "ENABLED").WithWriter(w)
	tbl.WithHeaderFormatter(func(format string, vals ...any) string {
		return strings.ToUpper(fmt.Sprintf(format, vals...))
	})

	for _, ext := range extensions {
		tbl.AddRow(ext.Name, ext.DisplayName, ext.BackendURL, ext.Enabled)
	}

	tbl.Print()
}
