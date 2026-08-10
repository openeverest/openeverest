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

// Package backup provides CLI business logic for backup management.
package backup

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

// terminal backup states.
const (
	backupStateSucceeded = "Succeeded"
	backupStateFailed    = "Failed"
)

// CreateOptions holds the inputs for the create command.
type CreateOptions struct {
	Instance       string
	Namespace      string
	Class          string
	Storage        string
	Name           string
	DeletionPolicy string
	Cluster        string
	Context        string
	Wait           bool
	Timeout        time.Duration
}

// CreateRunner creates an on-demand Backup via the Everest API.
type CreateRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewCreateRunner creates a new CreateRunner.
func NewCreateRunner(cfg Config, l *zap.SugaredLogger) *CreateRunner {
	cr := &CreateRunner{config: cfg, l: l.With("component", "backup-create")}
	if cfg.Pretty {
		cr.l = zap.NewNop().Sugar()
	}
	return cr
}

// Run creates a Backup and, with opts.Wait, blocks until it reaches a
// terminal state, opts.Timeout elapses (wait.ErrTimeout), or ctx is
// cancelled (context.Canceled).
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
	backup := client.Backup{Metadata: &md}
	backup.Spec.InstanceRef.Name = opts.Instance
	backup.Spec.ClassRef.Name = opts.Class
	backup.Spec.StorageRef.Name = opts.Storage
	if opts.DeletionPolicy != "" {
		backup.Spec.DeletionPolicy = opts.DeletionPolicy
	}

	resp, err := c.CreateBackupWithResponse(ctx, opts.Cluster, opts.Namespace, backup)
	if err != nil {
		return fmt.Errorf("create backup request failed: %w", err)
	}

	if resp.StatusCode() != http.StatusCreated || resp.JSON201 == nil {
		if resp.StatusCode() == http.StatusConflict {
			return fmt.Errorf("backup %q already exists in namespace %q", opts.Name, opts.Namespace)
		}
		if msg, ok := clienterr.Message(resp.JSONDefault); ok {
			return fmt.Errorf("server error: %s", msg)
		}
		return fmt.Errorf("unexpected response creating backup: %s", resp.Status())
	}

	name := backupName(resp.JSON201)
	cr.l.Infof("created backup %q for instance %q in namespace %q", name, opts.Instance, opts.Namespace)

	if !opts.Wait {
		return cr.emitCreated(resp.JSON201, name)
	}
	return cr.waitForBackup(ctx, c, resp.JSON201, opts, name)
}

// emitCreated reports a non-waiting create: a success line in pretty mode, or
// the created backup in JSON mode (so `create --json` is parseable either way).
func (cr *CreateRunner) emitCreated(created *client.Backup, name string) error {
	if cr.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Backup %q created", name))
		return nil
	}
	return writeBackupJSON(created)
}

// waitForBackup blocks until the backup reaches a terminal state, streaming
// progress in pretty mode. JSON mode emits one final object on success.
func (cr *CreateRunner) waitForBackup(
	ctx context.Context,
	c *client.ClientWithResponses,
	created *client.Backup,
	opts CreateOptions,
	name string,
) error {
	var latest *client.Backup
	basePoll := newBackupPoll(c, opts.Cluster, opts.Namespace, name)
	poll := func(ctx context.Context) (*client.Backup, error) {
		b, err := basePoll(ctx)
		if err == nil {
			latest = b
		}
		return b, err
	}

	var onUpdate func(string)
	if cr.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Info("Backup %q created; waiting for it to complete...", name))
		onUpdate = func(msg string) {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", msg)
		}
	}

	if err := wait.Until(ctx, poll, backupCondition, wait.Options{
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
		_, _ = fmt.Fprint(os.Stdout, output.Success("Backup %q completed", name))
		return nil
	}
	return writeBackupJSON(final)
}

func writeBackupJSON(b *client.Backup) error {
	if err := json.NewEncoder(os.Stdout).Encode(b); err != nil {
		return fmt.Errorf("failed to encode backup: %w", err)
	}
	return nil
}
