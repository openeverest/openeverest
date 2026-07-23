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
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// terminal backup states. BackupStateError is deliberately excluded: it is a
// transient condition the controller may retry, not a final outcome.
const (
	backupStateSucceeded = "Succeeded"
	backupStateFailed    = "Failed"
)

const pollInterval = 2 * time.Second

// Config holds the shared configuration for backup CLI runners.
type Config struct {
	Pretty bool
}

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

// Run creates a Backup for the given instance and, with opts.Wait set, blocks
// until it reaches a terminal state (Succeeded or Failed) or opts.Timeout
// elapses.
func (cr *CreateRunner) Run(ctx context.Context, opts CreateOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: cr.config.Pretty}, cr.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	backup := client.Backup{
		Metadata: &map[string]interface{}{
			"namespace": opts.Namespace,
		},
	}
	if opts.Name != "" {
		(*backup.Metadata)["name"] = opts.Name
	} else {
		(*backup.Metadata)["generateName"] = opts.Instance + "-"
	}
	backup.Spec.InstanceName = opts.Instance
	backup.Spec.BackupClassName = opts.Class
	backup.Spec.StorageName = opts.Storage
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
		if resp.JSONDefault != nil && resp.JSONDefault.Message != nil {
			return fmt.Errorf("server error: %s", *resp.JSONDefault.Message)
		}
		return fmt.Errorf("unexpected response creating backup: %s", resp.Status())
	}

	name := metadataStringField(resp.JSON201, "name")
	cr.l.Infof("created backup %q for instance %q in namespace %q", name, opts.Instance, opts.Namespace)

	if !opts.Wait {
		cr.printCreated(name, resp.JSON201)
		return nil
	}

	return cr.wait(ctx, c, opts, name)
}

func (cr *CreateRunner) printCreated(name string, backup *client.Backup) {
	if cr.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Backup %q created", name))
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(backup) //nolint:errchkjson
}

// wait polls the backup until it reaches a terminal state or the timeout
// elapses. Ctrl-C (context cancellation) returns without cancelling the
// backup itself, which keeps running server-side.
func (cr *CreateRunner) wait(ctx context.Context, c *client.ClientWithResponses, opts CreateOptions, name string) error {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-waitCtx.Done():
			return fmt.Errorf("timed out after %s waiting for backup %q to complete", timeout, name)
		case <-ticker.C:
			resp, err := c.GetBackupWithResponse(waitCtx, opts.Cluster, opts.Namespace, name)
			if err != nil {
				if errors.Is(err, authcli.ErrTokenRefresh) {
					return err
				}
				cr.l.Warnf("fetch failed: %v — retrying in %s", err, pollInterval)
				continue
			}
			if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
				cr.l.Warnf("unexpected response %s — retrying in %s", resp.Status(), pollInterval)
				continue
			}

			state := backupState(resp.JSON200)
			switch state {
			case backupStateSucceeded:
				cr.printCreated(name, resp.JSON200)
				return nil
			case backupStateFailed:
				cr.printCreated(name, resp.JSON200)
				return fmt.Errorf("backup %q failed: %s", name, backupMessage(resp.JSON200))
			}
		}
	}
}

func backupState(b *client.Backup) string {
	if b.Status == nil || b.Status.State == nil {
		return ""
	}
	return *b.Status.State
}

func backupMessage(b *client.Backup) string {
	if b.Status == nil || b.Status.Message == nil || *b.Status.Message == "" {
		return "no message reported"
	}
	return *b.Status.Message
}

func metadataStringField(b *client.Backup, key string) string {
	if b.Metadata == nil {
		return "-"
	}
	v, ok := (*b.Metadata)[key]
	if !ok {
		return "-"
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "-"
	}
	return s
}
