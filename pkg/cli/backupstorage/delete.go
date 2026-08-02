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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/cli/confirm"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// DeleteOptions configures `backup-storage delete`.
type DeleteOptions struct {
	Name      string
	Namespace string
	Cluster   string
	Context   string        // overrides the active context when set
	Yes       bool          // skip the confirmation prompt
	Wait      bool          // block until the backup storage is fully deleted
	Timeout   time.Duration // bounds --wait; must be positive

	// IsTerminal overrides the TTY check for the prompt. Set in tests.
	IsTerminal func() bool
}

// Deleter implements `backup-storage delete` business logic.
type Deleter struct {
	config Config
	l      *zap.SugaredLogger
}

// NewDeleter returns a new Deleter.
func NewDeleter(cfg Config, l *zap.SugaredLogger) *Deleter {
	bd := &Deleter{config: cfg, l: l.With("component", "backup-storage-deleter")}
	if cfg.Pretty {
		bd.l = zap.NewNop().Sugar()
	}
	return bd
}

// Run deletes the backup storage: confirms, deletes, then waits if asked.
// The credentials Secret is not handled here — the BackupStorage controller
// adopts it via an owner reference, so Kubernetes garbage-collects it once
// the BackupStorage is actually gone.
func (bd *Deleter) Run(ctx context.Context, opts DeleteOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: bd.config.Pretty}, bd.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	confirmOpts := confirm.Options{Yes: opts.Yes, JSON: !bd.config.Pretty, IsTerminal: opts.IsTerminal}
	msg := fmt.Sprintf("Delete backup storage %q in namespace %q?", opts.Name, opts.Namespace)
	if err := confirm.YesNo(ctx, confirmOpts, msg); err != nil {
		return err
	}

	resp, err := c.DeleteBackupStorageWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		return fmt.Errorf("delete backup storage request failed: %w", err)
	}
	if err := checkDeleteResponse(resp, opts); err != nil {
		return err
	}
	bd.l.Infof("deleted backup storage %q in namespace %q", opts.Name, opts.Namespace)

	if !opts.Wait {
		return bd.emitDeleted(opts)
	}
	return bd.waitForDeletion(ctx, c, opts)
}

// checkDeleteResponse maps a DeleteBackupStorage response to an error, or nil on success.
func checkDeleteResponse(resp *client.DeleteBackupStorageResponse, opts DeleteOptions) error {
	switch {
	case resp.StatusCode() == http.StatusNotFound:
		return fmt.Errorf("backup storage %q not found in namespace %q", opts.Name, opts.Namespace)
	case resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent:
		if msg, ok := clienterr.Message(resp.JSON400, resp.JSONDefault); ok {
			return fmt.Errorf("server error: %s", msg)
		}
		return fmt.Errorf("unexpected response deleting backup storage: %s", resp.Status())
	default:
		return nil
	}
}

// emitDeleted prints success: a line in pretty mode, JSON otherwise.
func (bd *Deleter) emitDeleted(opts DeleteOptions) error {
	if bd.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Backup storage %q deleted", opts.Name))
		return nil
	}
	return writeDeleteResultJSON(opts.Name, opts.Namespace)
}

// waitForDeletion blocks until the backup storage is gone, like instance delete --wait.
func (bd *Deleter) waitForDeletion(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) error {
	poll := newBackupStorageDeletePoll(c, opts.Cluster, opts.Namespace, opts.Name)

	var onUpdate func(string)
	if bd.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Info("Backup storage %q deletion requested; waiting for it to be fully removed...", opts.Name))
		onUpdate = func(msg string) {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", msg)
		}
	}

	err := wait.Until(ctx, poll, deleteCondition, wait.Options{
		Timeout:  opts.Timeout,
		OnUpdate: onUpdate,
		OnRetry:  func(err error) { bd.l.Warnf("%v — retrying", err) },
	})
	if errors.Is(err, wait.ErrTimeout) {
		return fmt.Errorf("timed out waiting for backup storage %q to be deleted — it may still be referenced by an Instance or Backup: %w", opts.Name, err)
	}
	if err != nil {
		return err
	}

	if bd.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Backup storage %q is deleted", opts.Name))
		return nil
	}
	return writeDeleteResultJSON(opts.Name, opts.Namespace)
}

// newBackupStorageDeletePoll checks if the backup storage still exists. A 404
// means it's gone, which is success here (unlike a create-side poll).
func newBackupStorageDeletePoll(
	c *client.ClientWithResponses,
	cluster, namespace, name string,
) wait.PollFunc[*client.BackupStorage] {
	return func(ctx context.Context) (*client.BackupStorage, error) {
		resp, err := c.GetBackupStorageWithResponse(ctx, cluster, namespace, name)
		if err != nil {
			if errors.Is(err, authcli.ErrTokenRefresh) {
				return nil, fmt.Errorf("failed to fetch backup storage %q: %w", name, err)
			}
			return nil, &wait.RetryableError{Err: fmt.Errorf("failed to fetch backup storage %q: %w", name, err)}
		}
		switch resp.StatusCode() {
		case http.StatusOK:
			if resp.JSON200 == nil {
				return nil, &wait.RetryableError{Err: fmt.Errorf("empty response body fetching backup storage %q", name)}
			}
			return resp.JSON200, nil
		case http.StatusNotFound:
			return nil, nil // gone
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("server rejected credentials — run 'everestctl auth login' again")
		default:
			return nil, &wait.RetryableError{Err: fmt.Errorf("unexpected response fetching backup storage %q: %s", name, resp.Status())}
		}
	}
}

func deleteCondition(bs *client.BackupStorage) (wait.Outcome, string) {
	if bs == nil {
		return wait.Succeeded, "backup storage deleted"
	}
	return wait.Pending, "backup storage still exists — likely referenced by an Instance or Backup"
}

func writeDeleteResultJSON(name, namespace string) error {
	result := struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Deleted   bool   `json:"deleted"`
	}{Name: name, Namespace: namespace, Deleted: true}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		return fmt.Errorf("failed to encode delete result: %w", err)
	}
	return nil
}
