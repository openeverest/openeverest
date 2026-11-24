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

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/cli/confirm"
	"github.com/openeverest/openeverest/v2/pkg/cli/deletion"
)

// DeleteOptions configures `preset delete`.
type DeleteOptions struct {
	Name            string
	Namespace       string
	Cluster         string
	Context         string
	Yes             bool
	MachineReadable bool // -o json|yaml was passed; forces non-interactive regardless of --verbose
	IgnoreNotFound  bool

	// IsTerminal overrides the TTY check for the prompt. Set in tests.
	IsTerminal func() bool
}

// Deleter implements `preset delete` business logic.
type Deleter struct {
	config Config
	l      *zap.SugaredLogger
}

// NewDeleter returns a new Deleter.
func NewDeleter(cfg Config, l *zap.SugaredLogger) *Deleter {
	d := &Deleter{config: cfg, l: l.With("component", "preset-deleter")}
	if cfg.Pretty {
		d.l = zap.NewNop().Sugar()
	}
	return d
}

// Run deletes the instance preset after confirmation.
func (d *Deleter) Run(ctx context.Context, opts DeleteOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: d.config.Pretty}, d.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	if opts.IgnoreNotFound && d.presetAlreadyGone(ctx, c, opts) {
		return d.emitAlreadyGone(opts)
	}

	confirmOpts := confirm.Options{Yes: opts.Yes, JSON: opts.MachineReadable, IsTerminal: opts.IsTerminal}
	msg := fmt.Sprintf("Delete instance preset %q in namespace %q?", opts.Name, opts.Namespace)
	if err := confirm.YesNo(ctx, confirmOpts, msg); err != nil {
		return err
	}

	resp, err := c.DeleteInstancePresetWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		return fmt.Errorf("delete instance preset request failed: %w", err)
	}

	alreadyGone, err := checkDeleteResponse(resp, opts)
	if err != nil {
		return err
	}
	if alreadyGone {
		return d.emitAlreadyGone(opts)
	}

	d.l.Infof("deleted instance preset %q in namespace %q", opts.Name, opts.Namespace)
	return deletion.EmitDeleted(d.config.Pretty, d.config.YAML, "instance preset", opts.Name, opts.Namespace)
}

func (d *Deleter) presetAlreadyGone(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) bool {
	resp, err := c.GetInstancePresetWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		d.l.Warnf("pre-delete check for instance preset %q failed: %v", opts.Name, err)
		return false
	}
	if resp.StatusCode() == http.StatusNotFound {
		return true
	}
	if resp.StatusCode() != http.StatusOK {
		statusText := resp.Status()
		if msg, ok := clienterr.Message(resp.JSONDefault); ok {
			statusText = msg
		}
		d.l.Warnf("pre-delete check for instance preset %q returned unexpected status %s, proceeding without it", opts.Name, statusText)
	}
	return false
}

func checkDeleteResponse(resp *client.DeleteInstancePresetResponse, opts DeleteOptions) (bool, error) {
	return deletion.CheckDeleteResponse(resp.StatusCode(), resp.Status(), "instance preset", opts.Name, opts.Namespace, opts.IgnoreNotFound,
		func() (string, bool) {
			return clienterr.Message(clienterr.FromAPIError(resp.JSON409), resp.JSON403, resp.JSONDefault)
		})
}

func (d *Deleter) emitAlreadyGone(opts DeleteOptions) error {
	return deletion.EmitAlreadyGone(d.config.Pretty, d.config.YAML, "instance preset", opts.Name, opts.Namespace)
}
