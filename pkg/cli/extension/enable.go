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

package extension

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// EnableConfig holds configuration for adding a namespace to an extension.
type EnableConfig struct {
	KubeconfigPath string
	Pretty         bool
	Name           string
	Namespace      string
}

// PluginEnabler adds a namespace to an InstalledExtension's
// spec.plugin.namespaces[] list.
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

// Run appends pe.cfg.Namespace to the InstalledExtension's
// spec.plugin.namespaces[] list (idempotent).
func (pe *PluginEnabler) Run(ctx context.Context) error {
	ie, err := pe.kubeClient.GetInstalledExtension(ctx, ctrlclient.ObjectKey{Name: pe.cfg.Name})
	if err != nil {
		return fmt.Errorf("InstalledExtension %q not found: %w", pe.cfg.Name, err)
	}
	if ie.Spec.Type != corev1alpha1.InstalledExtensionTypePlugin || ie.Spec.Plugin == nil {
		return fmt.Errorf("InstalledExtension %q is not a plugin install", pe.cfg.Name)
	}
	for _, nsCfg := range ie.Spec.Plugin.Namespaces {
		if nsCfg.Name == pe.cfg.Namespace {
			fmt.Printf("Plugin %q already enabled in namespace %q.\n", pe.cfg.Name, pe.cfg.Namespace)
			return nil
		}
	}
	ie.Spec.Plugin.Namespaces = append(ie.Spec.Plugin.Namespaces, corev1alpha1.PluginNamespaceConfig{
		Name: pe.cfg.Namespace,
	})
	if _, err := pe.kubeClient.UpdateInstalledExtension(ctx, ie); err != nil {
		return fmt.Errorf("cannot enable plugin %q in namespace %q: %w", pe.cfg.Name, pe.cfg.Namespace, err)
	}
	fmt.Printf("Plugin %q enabled in namespace %q.\n", pe.cfg.Name, pe.cfg.Namespace)
	return nil
}
