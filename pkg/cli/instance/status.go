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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
)

// StatusOptions holds the inputs for the status command.
type StatusOptions struct {
	Name      string
	Namespace string
	Cluster   string
	Context   string
	Watch     bool
	Interval  time.Duration
}

// InstanceStatusRunner fetches and prints the status of an instance.
type InstanceStatusRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewInstanceStatusRunner creates a new InstanceStatusRunner.
func NewInstanceStatusRunner(cfg Config, l *zap.SugaredLogger) *InstanceStatusRunner {
	is := &InstanceStatusRunner{config: cfg, l: l.With("component", "instance-status")}
	if cfg.Pretty {
		is.l = zap.NewNop().Sugar()
	}
	return is
}

// Run fetches the current status of an instance, printing it to stdout.
// With opts.Watch set, it polls continuously until the context is cancelled,
// the instance is deleted, or token refresh fails.
func (is *InstanceStatusRunner) Run(ctx context.Context, opts StatusOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: is.config.Pretty}, is.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	resp, err := c.GetInstanceWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
	if err != nil {
		return fmt.Errorf("failed to fetch instance %q: %w", opts.Name, err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return fmt.Errorf("instance %q not found in namespace %q", opts.Name, opts.Namespace)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return fmt.Errorf("unexpected response fetching instance %q: %s", opts.Name, resp.Status())
	}

	var buf bytes.Buffer
	is.render(&buf, resp.JSON200, opts.Namespace)
	fmt.Fprint(os.Stdout, buf.String())

	if opts.Watch {
		return is.watch(ctx, c, opts, buf.String())
	}
	return nil
}

func (is *InstanceStatusRunner) watch(
	ctx context.Context,
	c *client.ClientWithResponses,
	opts StatusOptions,
	lastOutput string,
) error {
	interval := opts.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			resp, err := c.GetInstanceWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name)
			if err != nil {
				if errors.Is(err, authcli.ErrTokenRefresh) {
					return err
				}
				is.l.Warnf("fetch failed: %v — retrying in %s", err, interval)
				continue
			}

			switch resp.StatusCode() {
			case http.StatusOK:
				if resp.JSON200 == nil {
					is.l.Warnf("empty response body — retrying in %s", interval)
					continue
				}
				var buf bytes.Buffer
				is.render(&buf, resp.JSON200, opts.Namespace)
				if buf.String() == lastOutput {
					continue
				}
				lastOutput = buf.String()
				if is.config.Pretty {
					fmt.Fprintln(os.Stdout, "---")
				}
				fmt.Fprint(os.Stdout, lastOutput)
			case http.StatusNotFound:
				return fmt.Errorf("instance %q has been deleted", opts.Name)
			case http.StatusUnauthorized:
				return fmt.Errorf("server rejected credentials — run 'everestctl auth login' again")
			default:
				is.l.Warnf("unexpected response %s — retrying in %s", resp.Status(), interval)
			}
		}
	}
}

func (is *InstanceStatusRunner) render(w io.Writer, inst *client.Instance, namespace string) {
	if !is.config.Pretty {
		_ = json.NewEncoder(w).Encode(inst) //nolint:errchkjson
		return
	}
	printInstanceStatus(w, inst, namespace)
}

func printInstanceStatus(w io.Writer, inst *client.Instance, namespace string) {
	phase := "-"
	message := "-"
	version := "-"

	if inst.Status != nil {
		if inst.Status.Phase != nil {
			phase = string(*inst.Status.Phase)
		}
		if inst.Status.Message != nil && *inst.Status.Message != "" {
			message = *inst.Status.Message
		}
		if inst.Status.Version != nil && *inst.Status.Version != "" {
			version = *inst.Status.Version
		}
	}

	fmt.Fprintf(w, "Instance:  %s\n", instanceName(inst))
	fmt.Fprintf(w, "Namespace: %s\n", namespace)
	fmt.Fprintf(w, "Phase:     %s\n", phase)
	fmt.Fprintf(w, "Version:   %s\n", version)
	fmt.Fprintf(w, "Message:   %s\n", message)

	printComponentTable(w, inst)
	printConditionTable(w, inst)
}

func instanceName(inst *client.Instance) string {
	if inst.Metadata == nil || inst.Metadata.Name == "" {
		return "-"
	}
	return inst.Metadata.Name
}

func printComponentTable(w io.Writer, inst *client.Instance) {
	if inst.Status == nil || inst.Status.Components == nil || len(*inst.Status.Components) == 0 {
		return
	}
	fmt.Fprintln(w, "\nComponents:")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  STATE\tREADY\tTOTAL")
	for _, comp := range *inst.Status.Components {
		state := "-"
		if comp.State != nil {
			state = *comp.State
		}
		var ready, total int32
		if comp.Ready != nil {
			ready = *comp.Ready
		}
		if comp.Total != nil {
			total = *comp.Total
		}
		fmt.Fprintf(tw, "  %s\t%d\t%d\n", state, ready, total)
	}
	tw.Flush() //nolint:errcheck,gosec
}

func printConditionTable(w io.Writer, inst *client.Instance) {
	if inst.Status == nil || inst.Status.Conditions == nil || len(*inst.Status.Conditions) == 0 {
		return
	}
	fmt.Fprintln(w, "\nConditions:")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  TYPE\tSTATUS\tREASON\tMESSAGE")
	for _, cond := range *inst.Status.Conditions {
		msg := cond.Message
		if msg == "" {
			msg = "-"
		}
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", cond.Type, string(cond.Status), cond.Reason, msg)
	}
	tw.Flush() //nolint:errcheck,gosec
}
