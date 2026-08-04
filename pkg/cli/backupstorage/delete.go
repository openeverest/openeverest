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
	"os/signal"
	"syscall"
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
	Name           string
	Namespace      string
	Cluster        string
	Context        string        // overrides the active context when set
	Yes            bool          // skip the confirmation prompt
	JSON           bool          // --json was passed; forces non-interactive regardless of --verbose
	IgnoreNotFound bool          // treat "backup storage already gone" (404) as a successful delete
	Wait           bool          // block until the backup storage is fully deleted
	Timeout        time.Duration // bounds --wait; must be positive

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
// ctx stays plain here so Ctrl-C during confirm is a decline, not a
// cancelled wait, see waitForDeletion. The credentials Secret is not
// handled here — the BackupStorage controller adopts it via an owner
// reference, so Kubernetes garbage-collects it once the BackupStorage is
// actually gone.
func (bd *Deleter) Run(ctx context.Context, opts DeleteOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: bd.config.Pretty}, bd.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	if opts.IgnoreNotFound && bd.backupStorageAlreadyGone(ctx, c, opts) {
		return bd.emitAlreadyGone(opts)
	}

	confirmOpts := confirm.Options{Yes: opts.Yes, JSON: opts.JSON, IsTerminal: opts.IsTerminal}
	msg := fmt.Sprintf("Delete backup storage %q in namespace %q?", opts.Name, opts.Namespace)
	if err := confirm.YesNo(ctx, confirmOpts, msg); err != nil {
		return err
	}

	resp, err := c.DeleteBackupStorageWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		return fmt.Errorf("delete backup storage request failed: %w", err)
	}

	alreadyGone, err := checkDeleteResponse(resp, opts)
	if err != nil {
		return err
	}

	if !alreadyGone {
		bd.l.Infof("deleted backup storage %q in namespace %q", opts.Name, opts.Namespace)
	}

	if !opts.Wait || alreadyGone {
		return bd.emitDeleted(opts)
	}
	return bd.waitForDeletion(ctx, c, opts)
}

// backupStorageAlreadyGone checks, for --ignore-not-found, whether the backup
// storage is already gone so Run can skip confirmation and --wait entirely
// instead of asking the user to confirm deleting something that doesn't
// exist. Any ambiguous result (fetch error, non-404 status) returns false so
// the normal confirm+delete flow runs and surfaces the real error itself.
func (bd *Deleter) backupStorageAlreadyGone(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) bool {
	resp, err := c.GetBackupStorageWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		return false
	}
	return resp.StatusCode() == http.StatusNotFound
}

// checkDeleteResponse maps a DeleteBackupStorage response to (alreadyGone, error).
func checkDeleteResponse(resp *client.DeleteBackupStorageResponse, opts DeleteOptions) (bool, error) {
	switch {
	case resp.StatusCode() == http.StatusNotFound:
		if !opts.IgnoreNotFound {
			return false, fmt.Errorf("backup storage %q not found in namespace %q", opts.Name, opts.Namespace)
		}
		return true, nil
	case resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent:
		if msg, ok := clienterr.Message(resp.JSON400, resp.JSONDefault); ok {
			return false, fmt.Errorf("server error: %s", msg)
		}
		return false, fmt.Errorf("unexpected response deleting backup storage: %s", resp.Status())
	default:
		return false, nil
	}
}

// emitDeleted prints success: a line in pretty mode, JSON otherwise.
func (bd *Deleter) emitDeleted(opts DeleteOptions) error {
	if bd.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Backup storage %q deleted", opts.Name))
		return nil
	}
	return writeDeleteResultJSON(opts.Name, opts.Namespace, true)
}

// emitAlreadyGone reports the --ignore-not-found short-circuit: nothing was
// deleted, so "deleted" must stay false, it should only mean this run
// actually deleted something.
func (bd *Deleter) emitAlreadyGone(opts DeleteOptions) error {
	if bd.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Info("Backup storage %q not found in namespace %q; nothing to delete", opts.Name, opts.Namespace))
		return nil
	}
	return writeDeleteResultJSON(opts.Name, opts.Namespace, false)
}

// waitForDeletion blocks until the backup storage is gone, like instance
// delete --wait. The cancellable context is set up here, not in Run, so
// Ctrl-C during confirm (which runs first) is a decline, not a cancelled
// wait.
func (bd *Deleter) waitForDeletion(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	poll := newBackupStorageDeletePoll(c, opts.Cluster, opts.Namespace, opts.Name)

	var onUpdate func(string)
	if bd.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Info("Backup storage %q deletion requested; waiting for it to be fully removed...", opts.Name))
		onUpdate = func(msg string) {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", msg)
		}
	}

	if err := wait.Until(ctx, poll, deleteCondition, wait.Options{
		Timeout:  opts.Timeout,
		OnUpdate: onUpdate,
		OnRetry:  func(err error) { bd.l.Warnf("%v — retrying", err) },
	}); err != nil {
		return err
	}

	if bd.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Backup storage %q is deleted", opts.Name))
		return nil
	}
	return writeDeleteResultJSON(opts.Name, opts.Namespace, true)
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

func writeDeleteResultJSON(name, namespace string, deleted bool) error {
	result := struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Deleted   bool   `json:"deleted"`
	}{Name: name, Namespace: namespace, Deleted: deleted}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		return fmt.Errorf("failed to encode delete result: %w", err)
	}
	return nil
}
