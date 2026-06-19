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
	"io"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// InstallConfig holds configuration for the extension install operation.
type InstallConfig struct {
	KubeconfigPath string
	Pretty         bool
	// File is a path or URL to a Plugin CR YAML manifest.
	File string
	// Inline flags (used when -f is not provided).
	Name        string
	DisplayName string
	BackendURL  string
	BundlePath  string
	Enabled     bool
	// AllowClusterScope opts into cluster-wide RBAC. When true, the
	// resulting InstalledExtension is created with spec.plugin.scope=Cluster
	// and spec.plugin.allowClusterScope=true.
	AllowClusterScope bool
}

// PluginInstaller installs an extension by creating a Plugin CR (when needed)
// and an InstalledExtension CR.
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

// Run creates the Plugin CR and the matching InstalledExtension.
func (pi *PluginInstaller) Run(ctx context.Context) error {
	var plugin *corev1alpha1.Plugin

	if pi.cfg.File != "" {
		p, err := readPluginManifest(pi.cfg.File)
		if err != nil {
			return fmt.Errorf("cannot read plugin manifest: %w", err)
		}
		plugin = p
	} else {
		displayName := pi.cfg.DisplayName
		if displayName == "" {
			displayName = pi.cfg.Name
		}

		bundlePath := pi.cfg.BundlePath
		if bundlePath == "" {
			bundlePath = "/main.js"
		}

		plugin = &corev1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Name: pi.cfg.Name,
			},
			Spec: corev1alpha1.PluginSpec{
				DisplayName: displayName,
				Backend:     &corev1alpha1.PluginBackend{ExternalURL: pi.cfg.BackendURL},
				Frontend:    &corev1alpha1.PluginFrontend{BundlePath: bundlePath},
				Enabled:     pi.cfg.Enabled,
			},
		}
	}

	if plugin.Name == "" {
		return fmt.Errorf("plugin name is required (set metadata.name in the manifest or use --name)")
	}
	if plugin.Spec.Backend == nil || (plugin.Spec.Backend.ExternalURL == "" && plugin.Spec.Backend.ServiceRef == nil) {
		return fmt.Errorf("backend URL is required (set spec.backend.externalUrl / spec.backend.serviceRef in the manifest or use --backend-url)")
	}

	if _, err := pi.kubeClient.CreatePlugin(ctx, plugin); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("cannot install plugin %q: %w", plugin.Name, err)
		}
	}

	scope := corev1alpha1.PluginInstallScopeNamespaces
	if pi.cfg.AllowClusterScope {
		scope = corev1alpha1.PluginInstallScopeCluster
	}
	ie := &corev1alpha1.InstalledExtension{
		ObjectMeta: metav1.ObjectMeta{Name: plugin.Name},
		Spec: corev1alpha1.InstalledExtensionSpec{
			Type: corev1alpha1.InstalledExtensionTypePlugin,
			Plugin: &corev1alpha1.PluginInstall{
				PluginCRName:      plugin.Name,
				Scope:             scope,
				AllowClusterScope: pi.cfg.AllowClusterScope,
			},
		},
	}
	if _, err := pi.kubeClient.CreateInstalledExtension(ctx, ie); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("cannot create InstalledExtension %q: %w", plugin.Name, err)
		}
	}

	fmt.Printf("Extension %q installed.\n", plugin.Name)
	return nil
}

// readPluginManifest reads a Plugin CR from a local file path or URL.
func readPluginManifest(source string) (*corev1alpha1.Plugin, error) {
	var data []byte
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		resp, httpErr := http.Get(source) //nolint:gosec,noctx
		if httpErr != nil {
			return nil, fmt.Errorf("failed to fetch %s: %w", source, httpErr)
		}
		defer resp.Body.Close() //nolint:errcheck
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch %s: HTTP %d", source, resp.StatusCode)
		}
		data, err = io.ReadAll(resp.Body)
	} else {
		data, err = os.ReadFile(source) //nolint:gosec
	}
	if err != nil {
		return nil, err
	}

	plugin := &corev1alpha1.Plugin{}
	if err := yaml.UnmarshalStrict(data, plugin); err != nil {
		return nil, fmt.Errorf("invalid plugin manifest: %w", err)
	}

	return plugin, nil
}
