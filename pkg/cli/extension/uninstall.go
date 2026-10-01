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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// UninstallConfig holds configuration for the extension uninstall operation.
type UninstallConfig struct {
	KubeconfigPath string
	Pretty         bool
	Name           string
}

// PluginUninstaller uninstalls an extension by deleting its InstalledExtension
// and Plugin CRs.
type PluginUninstaller struct {
	cfg        UninstallConfig
	kubeClient kubernetes.KubernetesConnector
	l          *zap.SugaredLogger
}

// NewPluginUninstaller creates a new PluginUninstaller.
func NewPluginUninstaller(cfg UninstallConfig, l *zap.SugaredLogger) (*PluginUninstaller, error) {
	pu := &PluginUninstaller{
		cfg: cfg,
		l:   l.With("component", "plugin-uninstaller"),
	}
	if cfg.Pretty {
		pu.l = zap.NewNop().Sugar()
	}

	k, err := cliutils.NewKubeConnector(pu.l, pu.cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	pu.kubeClient = k
	return pu, nil
}

// Run deletes the InstalledExtension and Plugin CRs created by install.
func (pu *PluginUninstaller) Run(ctx context.Context) error {
	plugin, err := pu.kubeClient.GetPlugin(ctx, ctrlclient.ObjectKey{Name: pu.cfg.Name})
	if err != nil {
		return fmt.Errorf("plugin %q not found: %w", pu.cfg.Name, err)
	}

	// The InstalledExtension is created next to the Plugin by install, is
	// cluster-scoped and carries no owner reference, so nothing else will ever
	// reclaim it. Remove it first, so it doesn't briefly report the Plugin as
	// missing before it goes away itself.
	if err := pu.deleteInstalledExtension(ctx); err != nil {
		return err
	}

	if err := pu.kubeClient.DeletePlugin(ctx, plugin); err != nil {
		return fmt.Errorf("cannot uninstall plugin %q: %w", pu.cfg.Name, err)
	}

	fmt.Printf("Plugin %q uninstalled successfully.\n", pu.cfg.Name)
	return nil
}

// deleteInstalledExtension removes the InstalledExtension named after the
// plugin. A missing one is not an error: the Plugin may have been created
// without going through install.
func (pu *PluginUninstaller) deleteInstalledExtension(ctx context.Context) error {
	ie, err := pu.kubeClient.GetInstalledExtension(ctx, ctrlclient.ObjectKey{Name: pu.cfg.Name})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot read InstalledExtension %q: %w", pu.cfg.Name, err)
	}

	if err := pu.kubeClient.DeleteInstalledExtension(ctx, ie); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("cannot delete InstalledExtension %q: %w", pu.cfg.Name, err)
	}
	return nil
}
