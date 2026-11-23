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

package backupstorage

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
)

// GetOptions holds the inputs for the get command.
type GetOptions struct {
	Name      string
	Namespace string
	Cluster   string
	Context   string
}

// GetRunner fetches a single backup storage via the Everest API.
type GetRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewGetRunner creates a new GetRunner.
func NewGetRunner(cfg Config, l *zap.SugaredLogger) *GetRunner {
	gr := &GetRunner{config: cfg, l: l.With("component", "backup-storage-get")}
	if cfg.Pretty {
		gr.l = zap.NewNop().Sugar()
	}
	return gr
}

// Run fetches one backup storage and prints it to stdout as JSON or YAML.
func (gr *GetRunner) Run(ctx context.Context, opts GetOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: gr.config.Pretty}, gr.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	bs, err := gr.fetch(ctx, c, opts)
	if err != nil {
		return err
	}
	return cli.Encode(os.Stdout, bs, gr.config.YAML)
}

func (gr *GetRunner) fetch(ctx context.Context, c *client.ClientWithResponses, opts GetOptions) (*client.BackupStorage, error) {
	resp, err := c.GetBackupStorageWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch backup storage %q: %w", opts.Name, err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("backup storage %q not found in namespace %q", opts.Name, opts.Namespace)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response fetching backup storage %q: %s", opts.Name, resp.Status())
	}
	return resp.JSON200, nil
}
