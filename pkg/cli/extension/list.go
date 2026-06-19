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

// Package plugins provides CLI operations for managing generic plugins.
package extension

import (
	"context"
	"fmt"
	"strings"

	"github.com/rodaine/table"
	"go.uber.org/zap"

	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// ListConfig holds configuration for the plugin list operation.
type ListConfig struct {
	KubeconfigPath string
	Pretty         bool
}

// PluginLister lists installed plugins.
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

// Run lists all plugins and prints them as a table.
func (pl *PluginLister) Run(ctx context.Context) error {
	plugins, err := pl.kubeClient.ListPlugins(ctx)
	if err != nil {
		return fmt.Errorf("cannot list plugins: %w", err)
	}

	tbl := table.New("NAME", "DISPLAY NAME", "BACKEND URL", "ENABLED")
	tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
		return strings.ToUpper(fmt.Sprintf(format, vals...))
	})

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
		tbl.AddRow(p.Name, p.Spec.DisplayName, backendURL, p.Spec.Enabled)
	}

	tbl.Print()
	return nil
}
