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
	"os"
	"os/exec"

	"go.uber.org/zap"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	cliutils "github.com/openeverest/openeverest/v2/pkg/cli/utils"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// RunConfig holds configuration for the plugin run operation.
type RunConfig struct {
	KubeconfigPath string
	Pretty         bool
	PluginName     string
	Runtime        string
	ExtraArgs      []string
}

// PluginRunner executes a plugin's CLI container.
type PluginRunner struct {
	cfg        RunConfig
	kubeClient kubernetes.KubernetesConnector
	l          *zap.SugaredLogger
}

// NewPluginRunner creates a new PluginRunner.
func NewPluginRunner(cfg RunConfig, l *zap.SugaredLogger) (*PluginRunner, error) {
	r := &PluginRunner{
		cfg: cfg,
		l:   l.With("component", "plugin-runner"),
	}
	if cfg.Pretty {
		r.l = zap.NewNop().Sugar()
	}

	k, err := cliutils.NewKubeConnector(r.l, r.cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	r.kubeClient = k
	return r, nil
}

// Run looks up the Plugin CR, validates it has a CLI image, and execs the container.
func (r *PluginRunner) Run(ctx context.Context) error {
	plugin, err := r.kubeClient.GetPlugin(ctx, ctrlclient.ObjectKey{Name: r.cfg.PluginName})
	if err != nil {
		return fmt.Errorf("cannot find plugin %q: %w", r.cfg.PluginName, err)
	}

	if plugin.Spec.CLI == nil || plugin.Spec.CLI.Image == "" {
		return fmt.Errorf("plugin %q does not declare a CLI image (spec.cli.image)", r.cfg.PluginName)
	}

	image := plugin.Spec.CLI.Image
	runtime := r.cfg.Runtime
	if runtime == "" {
		runtime = "docker"
	}

	// Build the container run command.
	// Pass --rm so the container is cleaned up after exit.
	// Attach stdin/stdout/stderr for interactive use.
	args := []string{"run", "--rm", "-i"}

	// Pass EVEREST_HOST env var if set.
	if host := os.Getenv("EVEREST_HOST"); host != "" {
		args = append(args, "-e", "EVEREST_HOST="+host)
	}

	// Pass EVEREST_TOKEN env var if set.
	if token := os.Getenv("EVEREST_TOKEN"); token != "" {
		args = append(args, "-e", "EVEREST_TOKEN="+token)
	}

	args = append(args, image)
	args = append(args, r.cfg.ExtraArgs...)

	r.l.Debugf("executing: %s %v", runtime, args)

	cmd := exec.CommandContext(ctx, runtime, args...) //nolint:gosec
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("plugin CLI exited with error: %w", err)
	}
	return nil
}
