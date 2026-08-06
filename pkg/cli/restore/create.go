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

// Package restore provides CLI business logic for restore management.
package restore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// CreateOptions holds the inputs for the create command.
type CreateOptions struct {
	Instance  string
	Namespace string
	Backup    string
	Name      string
	Cluster   string
	Context   string
	Wait      bool
	Timeout   time.Duration
}

// CreateRunner creates a Restore via the Everest API.
type CreateRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewCreateRunner creates a new CreateRunner.
func NewCreateRunner(cfg Config, l *zap.SugaredLogger) *CreateRunner {
	cr := &CreateRunner{config: cfg, l: l.With("component", "restore-create")}
	if cfg.Pretty {
		cr.l = zap.NewNop().Sugar()
	}
	return cr
}

// Run creates a Restore and, with opts.Wait, blocks until it reaches a
// terminal state.
func (cr *CreateRunner) Run(ctx context.Context, opts CreateOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: cr.config.Pretty}, cr.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	md := metav1.ObjectMeta{Namespace: opts.Namespace}
	if opts.Name != "" {
		md.Name = opts.Name
	} else {
		md.GenerateName = opts.Instance + "-"
	}
	restore := client.Restore{Metadata: &md}
	restore.Spec.InstanceRef.Name = opts.Instance
	restore.Spec.DataSource.Type = client.RestoreSpecDataSourceTypeBackup
	restore.Spec.DataSource.Backup = &struct {
		BackupRef struct {
			Name string `json:"name"`
		} `json:"backupRef"`
		Pitr *struct {
			Date *time.Time `json:"date,omitempty"`
			Type any        `json:"type"`
		} `json:"pitr,omitempty"`
	}{}
	restore.Spec.DataSource.Backup.BackupRef.Name = opts.Backup

	resp, err := c.CreateRestoreWithResponse(ctx, opts.Cluster, opts.Namespace, restore)
	if err != nil {
		return fmt.Errorf("create restore request failed: %w", err)
	}

	if resp.StatusCode() != http.StatusCreated || resp.JSON201 == nil {
		if resp.StatusCode() == http.StatusConflict {
			return fmt.Errorf("restore %q already exists in namespace %q", opts.Name, opts.Namespace)
		}
		if msg, ok := clienterr.Message(resp.JSONDefault); ok {
			return fmt.Errorf("server error: %s", msg)
		}
		return fmt.Errorf("unexpected response creating restore: %s", resp.Status())
	}

	name := restoreName(resp.JSON201)
	cr.l.Infof("created restore %q for instance %q in namespace %q", name, opts.Instance, opts.Namespace)

	if !opts.Wait {
		return cr.emitCreated(resp.JSON201, name)
	}
	return cr.waitForRestore(ctx, c, resp.JSON201, opts, name)
}

// emitCreated reports a non-waiting create: a success line in pretty mode, or
// the created restore in JSON mode (so `create --json` is parseable either way).
func (cr *CreateRunner) emitCreated(created *client.Restore, name string) error {
	if cr.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Restore %q created", name))
		return nil
	}
	// A 201 with an unparseable body would otherwise emit empty stdout.
	if created == nil {
		return fmt.Errorf("restore %q was created but the server returned an unreadable response body", name)
	}
	return writeRestoreJSON(created)
}

// waitForRestore blocks until the restore reaches a terminal state, streaming
// progress in pretty mode. JSON mode emits one final object on success.
func (cr *CreateRunner) waitForRestore(ctx context.Context, c *client.ClientWithResponses, created *client.Restore, opts CreateOptions, name string) error {
	var latest *client.Restore
	basePoll := newRestorePoll(c, opts.Cluster, opts.Namespace, name)
	poll := func(ctx context.Context) (*client.Restore, error) {
		r, err := basePoll(ctx)
		if err == nil {
			latest = r
		}
		return r, err
	}

	var onUpdate func(string)
	if cr.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Info("Restore %q created; waiting for it to complete...", name))
		onUpdate = func(msg string) {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", msg)
		}
	}

	if err := wait.Until(ctx, poll, restoreCondition, wait.Options{
		Timeout:  opts.Timeout,
		OnUpdate: onUpdate,
		OnRetry:  func(err error) { cr.l.Warnf("%v — retrying", err) },
	}); err != nil {
		return err
	}

	final := latest
	if final == nil {
		final = created
	}

	if cr.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Restore %q completed", name))
		return nil
	}
	return writeRestoreJSON(final)
}

func writeRestoreJSON(r *client.Restore) error {
	if r == nil {
		return nil
	}
	if err := json.NewEncoder(os.Stdout).Encode(r); err != nil {
		return fmt.Errorf("failed to encode restore: %w", err)
	}
	return nil
}
