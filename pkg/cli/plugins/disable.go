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

package plugins

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// DisableConfig holds configuration for the plugin disable operation.
type DisableConfig struct {
	KubeconfigPath string
	Pretty         bool
	Name           string
	Namespace      string
}

// PluginDisabler disables a plugin in a namespace by deleting its PluginInstallation CR.
type PluginDisabler struct {
	cfg        DisableConfig
	kubeClient kubernetes.KubernetesConnector
	l          *zap.SugaredLogger
}

// NewPluginDisabler creates a new PluginDisabler.
func NewPluginDisabler(cfg DisableConfig, l *zap.SugaredLogger) (*PluginDisabler, error) {
	pd := &PluginDisabler{
		cfg: cfg,
		l:   l.With("component", "plugin-disabler"),
	}
	if cfg.Pretty {
		pd.l = zap.NewNop().Sugar()
	}

	k, err := cliutils.NewKubeConnector(pd.l, pd.cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	pd.kubeClient = k
	return pd, nil
}

// Run deletes the PluginInstallation CR.
func (pd *PluginDisabler) Run(ctx context.Context) error {
	pi, err := pd.kubeClient.GetPluginInstallation(ctx, ctrlclient.ObjectKey{
		Name:      pd.cfg.Name,
		Namespace: pd.cfg.Namespace,
	})
	if err != nil {
		return fmt.Errorf("plugin %q is not enabled in namespace %q: %w", pd.cfg.Name, pd.cfg.Namespace, err)
	}

	if err := pd.kubeClient.DeletePluginInstallation(ctx, pi); err != nil {
		return fmt.Errorf("cannot disable plugin %q in namespace %q: %w", pd.cfg.Name, pd.cfg.Namespace, err)
	}

	fmt.Printf("Plugin %q disabled in namespace %q.\n", pd.cfg.Name, pd.cfg.Namespace)
	return nil
}
