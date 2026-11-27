// everest
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

package preset

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
)

// GetOptions holds the inputs for the get command.
type GetOptions struct {
	Name      string
	Namespace string
	Cluster   string
	Context   string
}

// GetRunner fetches a single InstancePreset via the Everest API.
type GetRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewGetRunner creates a new GetRunner.
func NewGetRunner(cfg Config, l *zap.SugaredLogger) *GetRunner {
	gr := &GetRunner{config: cfg, l: l.With("component", "preset-get")}
	if cfg.Pretty {
		gr.l = zap.NewNop().Sugar()
	}
	return gr
}

// Run fetches one instance preset and prints it to stdout as JSON or YAML.
func (gr *GetRunner) Run(ctx context.Context, opts GetOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: gr.config.Pretty}, gr.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	preset, err := gr.fetch(ctx, c, opts)
	if err != nil {
		return err
	}
	return gr.render(os.Stdout, preset)
}

func (gr *GetRunner) render(w io.Writer, preset *client.InstancePreset) error {
	return cli.Encode(w, preset, gr.config.YAML)
}

func (gr *GetRunner) fetch(ctx context.Context, c *client.ClientWithResponses, opts GetOptions) (*client.InstancePreset, error) {
	resp, err := c.GetInstancePresetWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instance preset %q: %w", opts.Name, err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("instance preset %q not found in namespace %q", opts.Name, opts.Namespace)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		if msg, ok := clienterr.Message(resp.JSON403, resp.JSONDefault); ok {
			return nil, fmt.Errorf("server error: %s", msg)
		}
		return nil, fmt.Errorf("unexpected response fetching instance preset %q: %s", opts.Name, resp.Status())
	}
	return resp.JSON200, nil
}
