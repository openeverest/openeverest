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

package instance

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

// DeleteOptions configures `instance delete`.
type DeleteOptions struct {
	Name           string
	Namespace      string
	Cluster        string
	Context        string        // overrides the active context when set
	DeletionPolicy string        // "", "Cascade", or "Orphan"; "" leaves the Instance's own spec.deletionPolicy in effect
	Yes            bool          // skip the confirmation prompt
	JSON           bool          // --json was passed; forces non-interactive regardless of --verbose
	IgnoreNotFound bool          // treat "instance already gone" (404) as a successful delete
	Wait           bool          // block until the instance is fully deleted
	Timeout        time.Duration // bounds --wait; must be positive

	// IsTerminal overrides the TTY check for the prompt. Set in tests.
	IsTerminal func() bool
}

// Deleter implements `instance delete` business logic.
type Deleter struct {
	config Config
	l      *zap.SugaredLogger
}

// NewDeleter returns a new Deleter.
func NewDeleter(cfg Config, l *zap.SugaredLogger) *Deleter {
	id := &Deleter{config: cfg, l: l.With("component", "instance-deleter")}
	if cfg.Pretty {
		id.l = zap.NewNop().Sugar()
	}
	return id
}

// Run deletes the instance: confirms, deletes, then waits if asked. ctx
// stays plain here so Ctrl-C during confirm is a decline, not a cancelled
// wait, see waitForDeletion.
func (id *Deleter) Run(ctx context.Context, opts DeleteOptions, cfgPath string) error {
	if opts.DeletionPolicy != "" &&
		opts.DeletionPolicy != string(client.DeleteInstanceParamsDeletionPolicyCascade) &&
		opts.DeletionPolicy != string(client.DeleteInstanceParamsDeletionPolicyOrphan) {
		return fmt.Errorf("invalid --deletion-policy %q: must be Cascade or Orphan", opts.DeletionPolicy)
	}

	c, err := authcli.NewAPIClient(authcli.Config{Pretty: id.config.Pretty}, id.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	var preFetched *client.Instance
	if opts.IgnoreNotFound {
		inst, gone := id.checkInstanceExists(ctx, c, opts)
		if gone {
			return id.emitAlreadyGone(opts)
		}
		preFetched = inst
	}

	confirmOpts := confirm.Options{Yes: opts.Yes, JSON: opts.JSON, IsTerminal: opts.IsTerminal}
	if err := id.confirmDeletion(ctx, c, confirmOpts, opts, preFetched); err != nil {
		return err
	}

	var params client.DeleteInstanceParams
	if opts.DeletionPolicy != "" {
		policy := client.DeleteInstanceParamsDeletionPolicy(opts.DeletionPolicy)
		params.DeletionPolicy = &policy
	}

	resp, err := c.DeleteInstanceWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name, &params)
	if err != nil {
		return fmt.Errorf("delete instance request failed: %w", err)
	}

	alreadyGone, err := checkDeleteResponse(resp, opts)
	if err != nil {
		return err
	}

	if alreadyGone {
		return id.emitAlreadyGone(opts)
	}
	id.l.Infof("deleted instance %q in namespace %q", opts.Name, opts.Namespace)

	if !opts.Wait {
		return id.emitDeleted(opts)
	}
	return id.waitForDeletion(ctx, c, opts)
}

// checkInstanceExists is for --ignore-not-found: (nil, true) means the
// instance is gone. Otherwise it also returns the fetched instance, so
// confirmDeletion can reuse it instead of fetching again. Any unclear
// result (fetch error, non-404 status) returns false and lets the normal delete flow handle it.
func (id *Deleter) checkInstanceExists(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) (*client.Instance, bool) {
	resp, err := c.GetInstanceWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		id.l.Warnf("pre-delete check for instance %q failed: %v", opts.Name, err)
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
		id.l.Warnf("pre-delete check for instance %q returned unexpected status %s, proceeding without it", opts.Name, statusText)
		return nil, false
	}
}

// checkDeleteResponse maps a DeleteInstance response to (alreadyGone, error).
func checkDeleteResponse(resp *client.DeleteInstanceResponse, opts DeleteOptions) (bool, error) {
	return deletion.CheckDeleteResponse(resp.StatusCode(), resp.Status(), "instance", opts.Name, opts.Namespace, opts.IgnoreNotFound,
		func() (string, bool) { return clienterr.Message(resp.JSONDefault) })
}

// confirmDeletion shows the blast radius and asks the user to type the
// name back. Reuses preFetched if given, instead of fetching again.
func (id *Deleter) confirmDeletion(ctx context.Context, c *client.ClientWithResponses, confirmOpts confirm.Options, opts DeleteOptions, preFetched *client.Instance) error {
	policy := opts.DeletionPolicy
	verified := policy != ""
	if policy == "" && confirm.WillPrompt(confirmOpts) {
		inst := preFetched
		if inst == nil {
			inst = id.fetchInstanceForPrompt(ctx, c, opts)
		}
		if p := policyFromInstance(inst); p != "" {
			policy = p
			verified = true
		}
	}
	if policy == "" {
		policy = string(client.DeleteInstanceParamsDeletionPolicyCascade)
	}

	msg := blastRadiusMessage(opts.Name, opts.Namespace, policy, verified)
	return confirm.Name(ctx, confirmOpts, msg, opts.Name)
}

// fetchInstanceForPrompt looks up the instance just for the confirmation
// message. Returns nil on failure, the delete call below is the real check.
func (id *Deleter) fetchInstanceForPrompt(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) *client.Instance {
	resp, err := c.GetInstanceWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil || resp.StatusCode() != http.StatusOK {
		return nil
	}
	return resp.JSON200
}

// policyFromInstance reads Spec.DeletionPolicy as a string, or "" if unset.
func policyFromInstance(inst *client.Instance) string {
	if inst == nil || inst.Spec.DeletionPolicy == nil {
		return ""
	}
	return string(*inst.Spec.DeletionPolicy)
}

// blastRadiusMessage is what the user reads before confirming. If verified
// is false, the policy was never actually checked (lookup failed, or RBAC
// allows delete but not read), so it must not be named as fact.
func blastRadiusMessage(name, namespace, policy string, verified bool) string {
	if !verified {
		return fmt.Sprintf(
			"This permanently deletes instance %q in namespace %q and, per the instance's own deletion policy, may delete all its Backup and Restore objects. Type the instance name to confirm.",
			name, namespace,
		)
	}
	action := "deletes all its Backup and Restore objects"
	if policy == string(client.DeleteInstanceParamsDeletionPolicyOrphan) {
		action = "leaves its Backup and Restore objects in place"
	}
	return fmt.Sprintf(
		"This permanently deletes instance %q in namespace %q and, per policy %s, %s. Type the instance name to confirm.",
		name, namespace, policy, action,
	)
}

// emitDeleted prints success: a line in pretty mode, JSON otherwise.
func (id *Deleter) emitDeleted(opts DeleteOptions) error {
	return deletion.EmitDeleted(id.config.Pretty, "instance", opts.Name, opts.Namespace)
}

// emitAlreadyGone reports the --ignore-not-found short-circuit.
func (id *Deleter) emitAlreadyGone(opts DeleteOptions) error {
	return deletion.EmitAlreadyGone(id.config.Pretty, "instance", opts.Name, opts.Namespace)
}

// waitForDeletion blocks until the instance is gone, like instance create
// --wait. The cancellable context is set up here, not in Run, so Ctrl-C
// during confirm (which runs first) is a decline, not a cancelled wait.
func (id *Deleter) waitForDeletion(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	poll := newInstanceDeletePoll(c, opts.Cluster, opts.Namespace, opts.Name)

	var onUpdate func(string)
	if id.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Info("Instance %q deletion requested; waiting for it to be fully removed...", opts.Name))
		onUpdate = func(msg string) {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", msg)
		}
	}

	if err := wait.Until(ctx, poll, deleteCondition, wait.Options{
		Timeout:  opts.Timeout,
		OnUpdate: onUpdate,
		OnRetry:  func(err error) { id.l.Warnf("%v — retrying", err) },
	}); err != nil {
		return err
	}

	return deletion.EmitWaitSucceeded(id.config.Pretty, "instance", opts.Name, opts.Namespace)
}
