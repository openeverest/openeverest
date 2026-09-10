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

package backup

import (
	"context"
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
	"github.com/openeverest/openeverest/v2/pkg/cli/deletion"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// backup deletion policy values (client.Backup.Spec.DeletionPolicy).
const (
	backupDeletionPolicyDelete = "Delete"
	backupDeletionPolicyRetain = "Retain"
)

// DeleteOptions configures `backup delete`.
type DeleteOptions struct {
	Name           string
	Namespace      string
	Cluster        string
	Context        string        // overrides the active context when set
	Yes            bool          // skip the confirmation prompt
	JSON           bool          // --json was passed; forces non-interactive regardless of --verbose
	Force          bool          // override the in-flight (Pending/Running/Error/no status yet) guard; orthogonal to Yes
	IgnoreNotFound bool          // treat "backup already gone" (404) as a successful delete
	Wait           bool          // block until the backup is fully deleted
	Timeout        time.Duration // bounds --wait; must be positive

	// IsTerminal overrides the TTY check for the prompt. Set in tests.
	IsTerminal func() bool
}

// Deleter implements `backup delete` business logic.
type Deleter struct {
	config Config
	l      *zap.SugaredLogger
}

// NewDeleter returns a new Deleter.
func NewDeleter(cfg Config, l *zap.SugaredLogger) *Deleter {
	bd := &Deleter{config: cfg, l: l.With("component", "backup-deleter")}
	if cfg.Pretty {
		bd.l = zap.NewNop().Sugar()
	}
	return bd
}

// Run guards against an in-flight backup, confirms (tier by policy),
// deletes, then waits. ctx stays plain so Ctrl-C during confirm declines.
func (bd *Deleter) Run(ctx context.Context, opts DeleteOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: bd.config.Pretty}, bd.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	backup, gone := bd.checkBackupExists(ctx, c, opts)
	if gone {
		if !opts.IgnoreNotFound {
			return fmt.Errorf("backup %q not found in namespace %q", opts.Name, opts.Namespace)
		}
		return bd.emitAlreadyGone(opts)
	}

	// A nil backup (fetch failed/ambiguous) means nothing to guard: this is a
	// client-side accident guard, not an invariant, so it must not block delete.
	state, ok := backupStateForGuard(backup)
	if inFlight(state, ok) && !opts.Force {
		switch state {
		case "":
			return fmt.Errorf("backup %q hasn't reported its status yet; wait a moment and try again, or re-run with --force", opts.Name)
		case client.BackupStatusStateError:
			return fmt.Errorf("backup %q is in the Error state and may still be retried by the controller; wait for it to settle or re-run with --force", opts.Name)
		default:
			return fmt.Errorf("backup %q is still running; wait for it to finish or re-run with --force", opts.Name)
		}
	}

	confirmOpts := confirm.Options{Yes: opts.Yes, JSON: opts.JSON, IsTerminal: opts.IsTerminal}
	if err := bd.confirmDeletion(ctx, confirmOpts, opts, backup); err != nil {
		return err
	}

	resp, err := c.DeleteBackupWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name, nil)
	if err != nil {
		return fmt.Errorf("delete backup request failed: %w", err)
	}

	alreadyGone, err := deletion.CheckDeleteResponse(resp.StatusCode(), resp.Status(), "backup", opts.Name, opts.Namespace, opts.IgnoreNotFound,
		func() (string, bool) { return clienterr.Message(resp.JSONDefault) })
	if err != nil {
		return err
	}

	if alreadyGone {
		return bd.emitAlreadyGone(opts)
	}
	bd.l.Infof("deleted backup %q in namespace %q", opts.Name, opts.Namespace)

	if !opts.Wait {
		return bd.emitDeleted(opts)
	}
	return bd.waitForDeletion(ctx, c, opts)
}

// checkBackupExists is for --ignore-not-found, the guard, and the confirm
// tier: (nil, true) means gone; any ambiguous result returns (nil, false).
func (bd *Deleter) checkBackupExists(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) (*client.Backup, bool) {
	resp, err := c.GetBackupWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		bd.l.Warnf("pre-delete check for backup %q failed: %v", opts.Name, err)
		return nil, false
	}
	switch resp.StatusCode() {
	case http.StatusNotFound:
		return nil, true
	case http.StatusOK:
		return resp.JSON200, false
	default:
		statusText := resp.Status()
		if msg, ok := clienterr.Message(resp.JSONDefault); ok {
			statusText = msg
		}
		bd.l.Warnf("pre-delete check for backup %q returned unexpected status %s, proceeding without it", opts.Name, statusText)
		return nil, false
	}
}

// backupStateForGuard reads state for the guard: ok is false only when the
// fetch found nothing or failed; ok true + state "" means read but no status yet.
func backupStateForGuard(b *client.Backup) (client.BackupStatusState, bool) {
	if b == nil {
		return "", false
	}
	if b.Status != nil && b.Status.State != nil {
		return *b.Status.State, true
	}
	return "", true
}

// inFlight reports if the backup should block a delete: Pending/Running,
// Error (the controller may still retry it, per BackupStateError's own doc
// comment), or read successfully with no status yet; a failed/ambiguous
// fetch never is. Failed is excluded, that one really is terminal.
func inFlight(state client.BackupStatusState, ok bool) bool {
	if !ok {
		return false
	}
	return state == "" || state == client.BackupStatusStatePending || state == client.BackupStatusStateRunning || state == client.BackupStatusStateError
}

// isRetainTier is the only case that gets the y/N tier; Delete, or
// unverified (preFetched nil), gets the stricter type-the-name tier.
func isRetainTier(preFetched *client.Backup) bool {
	return preFetched != nil && policyFromBackup(preFetched) == backupDeletionPolicyRetain
}

// confirmDeletion picks the tier via isRetainTier and reuses preFetched
// instead of fetching again.
func (bd *Deleter) confirmDeletion(ctx context.Context, confirmOpts confirm.Options, opts DeleteOptions, preFetched *client.Backup) error {
	if isRetainTier(preFetched) {
		return confirm.YesNo(ctx, confirmOpts, retainConfirmMessage(opts.Name, opts.Namespace))
	}

	verified := preFetched != nil
	storage := ""
	if verified {
		storage = backupStorage(preFetched)
	}
	msg := deleteConfirmMessage(opts.Name, opts.Namespace, storage, verified)
	return confirm.Name(ctx, confirmOpts, msg, opts.Name)
}

// policyFromBackup reads Spec.DeletionPolicy as a string. Empty (unset)
// reads as Delete, the CRD's own default.
func policyFromBackup(b *client.Backup) string {
	if b == nil {
		return ""
	}
	if b.Spec.DeletionPolicy == nil || string(*b.Spec.DeletionPolicy) == "" {
		return backupDeletionPolicyDelete
	}
	return string(*b.Spec.DeletionPolicy)
}

// retainConfirmMessage is the y/N text for policy Retain: the underlying
// data stays in storage, this only removes the record.
func retainConfirmMessage(name, namespace string) string {
	return fmt.Sprintf(
		"Delete backup %q in namespace %q? Its data in storage stays in place (policy Retain).",
		name, namespace,
	)
}

// deleteConfirmMessage is the type-the-name text for policy Delete.
// If verified is false, neither policy nor storage is asserted as fact.
func deleteConfirmMessage(name, namespace, storage string, verified bool) string {
	if !verified {
		return fmt.Sprintf(
			"This permanently deletes backup %q in namespace %q and, per the backup's own deletion policy, may delete its data in storage. Type the backup name to confirm.",
			name, namespace,
		)
	}
	return fmt.Sprintf(
		"This permanently deletes backup %q in namespace %q and, per policy Delete, its data in storage %q. Type the backup name to confirm.",
		name, namespace, storage,
	)
}

// emitDeleted prints success: a line in pretty mode, JSON otherwise.
func (bd *Deleter) emitDeleted(opts DeleteOptions) error {
	return deletion.EmitDeleted(bd.config.Pretty, "backup", opts.Name, opts.Namespace)
}

// emitAlreadyGone reports the --ignore-not-found short-circuit: "deleted"
// stays false since this run didn't actually delete anything.
func (bd *Deleter) emitAlreadyGone(opts DeleteOptions) error {
	return deletion.EmitAlreadyGone(bd.config.Pretty, "backup", opts.Name, opts.Namespace)
}

// waitForDeletion blocks until the backup is gone. The cancellable context
// is set up here, not in Run, so Ctrl-C during confirm declines, not cancels.
func (bd *Deleter) waitForDeletion(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	poll := newBackupDeletePoll(c, opts.Cluster, opts.Namespace, opts.Name)

	var onUpdate func(string)
	if bd.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Info("Backup %q deletion requested; waiting for it to be fully removed...", opts.Name))
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

	return deletion.EmitWaitSucceeded(bd.config.Pretty, "backup", opts.Name, opts.Namespace)
}
