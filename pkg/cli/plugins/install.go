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

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// InstallConfig holds configuration for the plugin install operation.
type InstallConfig struct {
	KubeconfigPath string
	Pretty         bool
	Name           string
	DisplayName    string
	BackendURL     string
	BundlePath     string
	Enabled        bool
}

// PluginInstaller installs a plugin by creating a Plugin CR.
type PluginInstaller struct {
	cfg        InstallConfig
	kubeClient kubernetes.KubernetesConnector
	l          *zap.SugaredLogger
}

// NewPluginInstaller creates a new PluginInstaller.
func NewPluginInstaller(cfg InstallConfig, l *zap.SugaredLogger) (*PluginInstaller, error) {
	pi := &PluginInstaller{
		cfg: cfg,
		l:   l.With("component", "plugin-installer"),
	}
	if cfg.Pretty {
		pi.l = zap.NewNop().Sugar()
	}

	k, err := cliutils.NewKubeConnector(pi.l, pi.cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	pi.kubeClient = k
	return pi, nil
}

// Run creates the Plugin CR.
func (pi *PluginInstaller) Run(ctx context.Context) error {
	displayName := pi.cfg.DisplayName
	if displayName == "" {
		displayName = pi.cfg.Name
	}

	bundlePath := pi.cfg.BundlePath
	if bundlePath == "" {
		bundlePath = "/main.js"
	}

	plugin := &v1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: pi.cfg.Name,
		},
		Spec: v1alpha1.PluginSpec{
			DisplayName: displayName,
			BackendURL:  pi.cfg.BackendURL,
			BundlePath:  bundlePath,
			Enabled:     pi.cfg.Enabled,
		},
	}

	if _, err := pi.kubeClient.CreatePlugin(ctx, plugin); err != nil {
		return fmt.Errorf("cannot install plugin %q: %w", pi.cfg.Name, err)
	}

	fmt.Printf("Plugin %q installed successfully.\n", pi.cfg.Name)
	return nil
}
