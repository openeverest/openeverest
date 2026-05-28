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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/plugin/v1alpha1"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// EnableConfig holds configuration for the plugin enable operation.
type EnableConfig struct {
	KubeconfigPath string
	Pretty         bool
	Name           string
	Namespace      string
}

// PluginEnabler enables a plugin in a namespace by creating a PluginInstallation CR.
type PluginEnabler struct {
	cfg        EnableConfig
	kubeClient kubernetes.KubernetesConnector
	l          *zap.SugaredLogger
}

// NewPluginEnabler creates a new PluginEnabler.
func NewPluginEnabler(cfg EnableConfig, l *zap.SugaredLogger) (*PluginEnabler, error) {
	pe := &PluginEnabler{
		cfg: cfg,
		l:   l.With("component", "plugin-enabler"),
	}
	if cfg.Pretty {
		pe.l = zap.NewNop().Sugar()
	}

	k, err := cliutils.NewKubeConnector(pe.l, pe.cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	pe.kubeClient = k
	return pe, nil
}

// Run creates the PluginInstallation CR.
func (pe *PluginEnabler) Run(ctx context.Context) error {
	pi := &v1alpha1.PluginInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pe.cfg.Name,
			Namespace: pe.cfg.Namespace,
		},
		Spec: v1alpha1.PluginInstallationSpec{
			PluginName: pe.cfg.Name,
			Enabled:    true,
		},
	}

	if _, err := pe.kubeClient.CreatePluginInstallation(ctx, pi); err != nil {
		return fmt.Errorf("cannot enable plugin %q in namespace %q: %w", pe.cfg.Name, pe.cfg.Namespace, err)
	}

	fmt.Printf("Plugin %q enabled in namespace %q.\n", pe.cfg.Name, pe.cfg.Namespace)
	return nil
}
