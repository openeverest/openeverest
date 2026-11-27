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
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// CreateOptions holds the inputs for the create command.
type CreateOptions struct {
	Name      string // optional override for metadata.name
	Namespace string // optional override for metadata.namespace
	Cluster   string
	Context   string
	File      string // required path to an InstancePreset YAML/JSON manifest
}

// CreateRunner creates an InstancePreset via the Everest API.
type CreateRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewCreateRunner creates a new CreateRunner.
func NewCreateRunner(cfg Config, l *zap.SugaredLogger) *CreateRunner {
	cr := &CreateRunner{config: cfg, l: l.With("component", "preset-create")}
	if cfg.Pretty {
		cr.l = zap.NewNop().Sugar()
	}
	return cr
}

// Run creates an InstancePreset from -f and prints a success line or the
// created object.
func (cr *CreateRunner) Run(ctx context.Context, opts CreateOptions, cfgPath string) error {
	if opts.File == "" {
		return fmt.Errorf("--%s is required", cli.FlagPresetFile)
	}

	preset, err := loadPresetFile(opts.File)
	if err != nil {
		return err
	}
	stripServerFields(preset)
	name, namespace, err := applyIdentity(preset, opts.Name, opts.Namespace)
	if err != nil {
		return err
	}

	c, err := authcli.NewAPIClient(authcli.Config{Pretty: cr.config.Pretty}, cr.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	resp, err := c.CreateInstancePresetWithResponse(ctx, opts.Cluster, namespace, *preset)
	if err != nil {
		return fmt.Errorf("create instance preset request failed: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusCreated:
		cr.l.Infof("created instance preset %q in namespace %q", name, namespace)
		return cr.emitCreated(resp.JSON201, name)
	case http.StatusConflict:
		return fmt.Errorf("instance preset %q already exists in namespace %q", name, namespace)
	default:
		if msg, ok := clienterr.Message(resp.JSON400, resp.JSON403, resp.JSON404, clienterr.FromAPIError(resp.JSON409), clienterr.FromAPIError(resp.JSON422), resp.JSONDefault); ok {
			return fmt.Errorf("server error: %s", msg)
		}
		return fmt.Errorf("unexpected response creating instance preset %q: %s", name, resp.Status())
	}
}

func (cr *CreateRunner) emitCreated(created *client.InstancePreset, name string) error {
	if cr.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Instance preset %q created", name))
		return nil
	}
	if created == nil {
		return fmt.Errorf("instance preset %q was created but the server returned an unreadable response body", name)
	}
	return cli.Encode(os.Stdout, created, cr.config.YAML)
}
