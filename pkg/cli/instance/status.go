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
	"errors"
	"fmt"
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

	is.render(resp.JSON200, opts.Namespace)

	if opts.Watch {
		return is.watch(ctx, c, opts)
	}
	return nil
}

func (is *InstanceStatusRunner) watch(
	ctx context.Context,
	c *client.ClientWithResponses,
	opts StatusOptions,
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
				if is.config.Pretty {
					fmt.Fprintln(os.Stdout, "---")
				}
				is.render(resp.JSON200, opts.Namespace)
			case http.StatusNotFound:
				return fmt.Errorf("instance %q has been deleted", opts.Name)
			default:
				is.l.Warnf("unexpected response %s — retrying in %s", resp.Status(), interval)
			}
		}
	}
}

func (is *InstanceStatusRunner) render(inst *client.Instance, namespace string) {
	if !is.config.Pretty {
		_ = json.NewEncoder(os.Stdout).Encode(inst)
		return
	}
	printInstanceStatus(inst, namespace)
}

func printInstanceStatus(inst *client.Instance, namespace string) {
	name := "-"
	if inst.Metadata != nil {
		if n, ok := (*inst.Metadata)["name"]; ok {
			if s, ok := n.(string); ok {
				name = s
			}
		}
	}

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

	fmt.Fprintf(os.Stdout, "Instance:  %s\n", name)
	fmt.Fprintf(os.Stdout, "Namespace: %s\n", namespace)
	fmt.Fprintf(os.Stdout, "Phase:     %s\n", phase)
	fmt.Fprintf(os.Stdout, "Version:   %s\n", version)
	fmt.Fprintf(os.Stdout, "Message:   %s\n", message)

	if inst.Status != nil && inst.Status.Components != nil && len(*inst.Status.Components) > 0 {
		fmt.Fprintln(os.Stdout, "\nComponents:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  STATE\tREADY\tTOTAL")
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
			fmt.Fprintf(w, "  %s\t%d\t%d\n", state, ready, total)
		}
		w.Flush() //nolint:errcheck
	}

	if inst.Status != nil && inst.Status.Conditions != nil && len(*inst.Status.Conditions) > 0 {
		fmt.Fprintln(os.Stdout, "\nConditions:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  TYPE\tSTATUS\tREASON\tMESSAGE")
		for _, cond := range *inst.Status.Conditions {
			msg := cond.Message
			if msg == "" {
				msg = "-"
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", cond.Type, string(cond.Status), cond.Reason, msg)
		}
		w.Flush() //nolint:errcheck
	}
}
