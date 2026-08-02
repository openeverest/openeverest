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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/confirm"
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

// Run deletes the instance: confirms, deletes, then waits if asked.
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

	confirmOpts := confirm.Options{Yes: opts.Yes, JSON: !id.config.Pretty, IsTerminal: opts.IsTerminal}
	if err := id.confirmDeletion(ctx, c, confirmOpts, opts); err != nil {
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

	if !alreadyGone {
		id.l.Infof("deleted instance %q in namespace %q", opts.Name, opts.Namespace)
	}

	if !opts.Wait || alreadyGone {
		return id.emitDeleted(opts)
	}
	return id.waitForDeletion(ctx, c, opts)
}

// checkDeleteResponse maps a DeleteInstance response to (alreadyGone, error).
func checkDeleteResponse(resp *client.DeleteInstanceResponse, opts DeleteOptions) (bool, error) {
	switch {
	case resp.StatusCode() == http.StatusNotFound:
		if !opts.IgnoreNotFound {
			return false, fmt.Errorf("instance %q not found in namespace %q", opts.Name, opts.Namespace)
		}
		return true, nil
	case resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent:
		if resp.JSONDefault != nil && resp.JSONDefault.Message != nil {
			return false, fmt.Errorf("server error: %s", *resp.JSONDefault.Message)
		}
		return false, fmt.Errorf("unexpected response deleting instance: %s", resp.Status())
	default:
		return false, nil
	}
}

// confirmDeletion shows the blast radius and asks the user to type the name back.
func (id *Deleter) confirmDeletion(ctx context.Context, c *client.ClientWithResponses, confirmOpts confirm.Options, opts DeleteOptions) error {
	policy := opts.DeletionPolicy
	if policy == "" && confirm.WillPrompt(confirmOpts) {
		policy = id.fetchEffectivePolicyForPrompt(ctx, c, opts)
	}
	if policy == "" {
		policy = string(client.DeleteInstanceParamsDeletionPolicyCascade)
	}

	msg := blastRadiusMessage(opts.Name, opts.Namespace, policy)
	return confirm.Name(ctx, confirmOpts, msg, opts.Name)
}

// fetchEffectivePolicyForPrompt looks up the instance's policy just to show
// it in the prompt. On failure it falls back to Cascade — the delete call
// below is what actually checks the instance exists.
func (id *Deleter) fetchEffectivePolicyForPrompt(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) string {
	resp, err := c.GetInstanceWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil || resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return ""
	}
	inst := resp.JSON200
	// DeletionPolicy is generated as interface{}, so read it as a plain string.
	policy, ok := inst.Spec.DeletionPolicy.(string)
	if !ok {
		return ""
	}
	return policy
}

func blastRadiusMessage(name, namespace, policy string) string {
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
	if id.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Instance %q deleted", opts.Name))
		return nil
	}
	return writeDeleteResultJSON(opts.Name, opts.Namespace)
}

// waitForDeletion blocks until the instance is gone, like instance create --wait.
func (id *Deleter) waitForDeletion(ctx context.Context, c *client.ClientWithResponses, opts DeleteOptions) error {
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

	if id.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Instance %q is deleted", opts.Name))
		return nil
	}
	return writeDeleteResultJSON(opts.Name, opts.Namespace)
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
