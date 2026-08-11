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

package restore

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

// DeleteOptions configures `restore delete`.
type DeleteOptions struct {
	Name           string
	Namespace      string
	Cluster        string
	Context        string        // overrides the active context when set
	Yes            bool          // skip the confirmation prompt
	JSON           bool          // --json was passed; forces non-interactive regardless of --verbose
	Force          bool          // override the in-flight (Pending/Running/no status yet) guard; orthogonal to Yes
	IgnoreNotFound bool          // treat "restore already gone" (404) as a successful delete
	Wait           bool          // block until the restore is fully deleted
	Timeout        time.Duration // bounds --wait; must be positive

	// IsTerminal overrides the TTY check for the prompt. Set in tests.
	IsTerminal func() bool
}

// Deleter implements `restore delete` business logic.
type Deleter struct {
	config Config
	l      *zap.SugaredLogger
}

// NewDeleter returns a new Deleter.
func NewDeleter(cfg Config, l *zap.SugaredLogger) *Deleter {
	rd := &Deleter{config: cfg, l: l.With("component", "restore-deleter")}
	if cfg.Pretty {
		rd.l = zap.NewNop().Sugar()
	}
	return rd
}

// Run guards against an in-flight restore, confirms, deletes, then waits.
// ctx stays plain so Ctrl-C during confirm declines, not cancels the wait.
func (rd *Deleter) Run(ctx context.Context, opts DeleteOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: rd.config.Pretty}, rd.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	restore, gone := rd.checkRestoreExists(ctx, c, opts)
	if gone {
		if !opts.IgnoreNotFound {
			return fmt.Errorf("restore %q not found in namespace %q", opts.Name, opts.Namespace)
		}
		return rd.emitAlreadyGone(opts)
	}

	// A nil restore (fetch failed/ambiguous) means nothing to guard: this is a
	// client-side accident guard, not an invariant, so it must not block delete.
	state, ok := restoreStateForGuard(restore)
	forcingInFlight := inFlight(state, ok) && opts.Force
	if inFlight(state, ok) && !opts.Force {
		if state == "" {
			return fmt.Errorf("restore %q hasn't reported its status yet; wait a moment and try again, or re-run with --force", opts.Name)
		}
		return fmt.Errorf("restore %q is still running; wait for it to finish or re-run with --force", opts.Name)
	}

	confirmOpts := confirm.Options{Yes: opts.Yes, JSON: opts.JSON, IsTerminal: opts.IsTerminal}
	msg := deleteConfirmMessage(opts.Name, opts.Namespace, state, forcingInFlight)
	if err := confirm.YesNo(ctx, confirmOpts, msg); err != nil {
		return err
	}

	resp, err := c.DeleteRestoreWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		return fmt.Errorf("delete restore request failed: %w", err)
	}

	alreadyGone, err := deletion.CheckDeleteResponse(resp.StatusCode(), resp.Status(), "restore", opts.Name, opts.Namespace, opts.IgnoreNotFound,
		func() (string, bool) { return clienterr.Message(resp.JSONDefault) })
	if err != nil {
		return err
	}

	if alreadyGone {
		return rd.emitAlreadyGone(opts)
	}
	rd.l.Infof("deleted restore %q in namespace %q", opts.Name, opts.Namespace)

	if !opts.Wait {
		return rd.emitDeleted(opts)
	}
	return rd.waitForDeletion(ctx, c, opts)
}

// checkRestoreExists is for --ignore-not-found and the in-flight guard:
// (nil, true) means gone; any ambiguous result returns (nil, false).
func (rd *Deleter) checkRestoreExists(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) (*client.Restore, bool) {
	resp, err := c.GetRestoreWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		rd.l.Warnf("pre-delete check for restore %q failed: %v", opts.Name, err)
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
		rd.l.Warnf("pre-delete check for restore %q returned unexpected status %s, proceeding without it", opts.Name, statusText)
		return nil, false
	}
}

// restoreStateForGuard reads state for the guard: ok is false only when the
// fetch found nothing or failed; ok true + state "" means read but no status yet.
func restoreStateForGuard(r *client.Restore) (string, bool) {
	if r == nil {
		return "", false
	}
	if r.Status != nil && r.Status.State != nil {
		return *r.Status.State, true
	}
	return "", true
}

// inFlight reports if the restore should block a delete: Pending/Running,
// or read successfully with no status yet; a failed/ambiguous fetch never is.
func inFlight(state string, ok bool) bool {
	if !ok {
		return false
	}
	return state == "" || state == restoreStatePending || state == restoreStateRunning
}

// deleteConfirmMessage is what the user reads before confirming.
// forcingInFlight adds a warning about interrupting an in-flight restore.
func deleteConfirmMessage(name, namespace, state string, forcingInFlight bool) string {
	if forcingInFlight {
		display := state
		if display == "" {
			display = "not yet reported"
		}
		return fmt.Sprintf(
			"Restore %q in namespace %q may still be in progress (state: %s). Interrupting it triggers the provider's cleanup of the engine restore and can leave the target instance in an inconsistent state. Delete anyway?",
			name, namespace, display,
		)
	}
	return fmt.Sprintf("Delete restore %q in namespace %q?", name, namespace)
}

// emitDeleted prints success: a line in pretty mode, JSON otherwise.
func (rd *Deleter) emitDeleted(opts DeleteOptions) error {
	return deletion.EmitDeleted(rd.config.Pretty, "restore", opts.Name, opts.Namespace)
}

// emitAlreadyGone reports the --ignore-not-found short-circuit: "deleted"
// stays false since this run didn't actually delete anything.
func (rd *Deleter) emitAlreadyGone(opts DeleteOptions) error {
	return deletion.EmitAlreadyGone(rd.config.Pretty, "restore", opts.Name, opts.Namespace)
}

// waitForDeletion blocks until the restore is gone. The cancellable context
// is set up here, not in Run, so Ctrl-C during confirm declines, not cancels.
func (rd *Deleter) waitForDeletion(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	poll := newRestoreDeletePoll(c, opts.Cluster, opts.Namespace, opts.Name)

	var onUpdate func(string)
	if rd.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Info("Restore %q deletion requested; waiting for it to be fully removed...", opts.Name))
		onUpdate = func(msg string) {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", msg)
		}
	}

	if err := wait.Until(ctx, poll, deleteCondition, wait.Options{
		Timeout:  opts.Timeout,
		OnUpdate: onUpdate,
		OnRetry:  func(err error) { rd.l.Warnf("%v — retrying", err) },
	}); err != nil {
		return err
	}

	return deletion.EmitWaitSucceeded(rd.config.Pretty, "restore", opts.Name, opts.Namespace)
}
