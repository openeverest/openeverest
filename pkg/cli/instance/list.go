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
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/rodaine/table"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
)

// ListOptions holds the inputs for the list command.
type ListOptions struct {
	Namespace     string
	AllNamespaces bool
	Cluster       string
	Context       string
}

// ListRunner lists instances via the Everest API.
type ListRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewListRunner creates a new ListRunner.
func NewListRunner(cfg Config, l *zap.SugaredLogger) *ListRunner {
	il := &ListRunner{config: cfg, l: l.With("component", "instance-list")}
	if cfg.Pretty {
		il.l = zap.NewNop().Sugar()
	}
	return il
}

// Run fetches instances and prints them to stdout, either as a table (pretty mode)
// or as raw JSON.
func (il *ListRunner) Run(ctx context.Context, opts ListOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: il.config.Pretty}, il.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	namespaces := []string{opts.Namespace}
	if opts.AllNamespaces {
		if namespaces, err = il.listNamespaces(ctx, c, opts.Cluster); err != nil {
			return err
		}
	}

	instances := make([]client.Instance, 0, len(namespaces))
	for _, ns := range namespaces {
		items, err := il.listInstances(ctx, c, opts.Cluster, ns)
		if err != nil {
			return err
		}
		instances = append(instances, items...)
	}

	sort.Slice(instances, func(i, j int) bool {
		ni, nj := instanceNamespace(&instances[i]), instanceNamespace(&instances[j])
		if ni != nj {
			return ni < nj
		}
		return instanceName(&instances[i]) < instanceName(&instances[j])
	})

	var buf bytes.Buffer
	il.render(&buf, instances)
	fmt.Fprint(os.Stdout, buf.String())
	return nil
}

func (il *ListRunner) listNamespaces(ctx context.Context, c *client.ClientWithResponses, cluster string) ([]string, error) {
	resp, err := c.ListNamespacesWithResponse(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response listing namespaces: %s", resp.Status())
	}
	return *resp.JSON200, nil
}

func (il *ListRunner) listInstances(ctx context.Context, c *client.ClientWithResponses, cluster, namespace string) ([]client.Instance, error) {
	resp, err := c.ListInstancesWithResponse(ctx, cluster, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances in namespace %q: %w", namespace, err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response listing instances in namespace %q: %s", namespace, resp.Status())
	}
	if resp.JSON200.Items == nil {
		return nil, nil
	}
	return *resp.JSON200.Items, nil
}

func (il *ListRunner) render(w io.Writer, instances []client.Instance) {
	if !il.config.Pretty {
		_ = json.NewEncoder(w).Encode(instances) //nolint:errchkjson
		return
	}
	printInstanceTable(w, instances)
}

func printInstanceTable(w io.Writer, instances []client.Instance) {
	tbl := table.New("NAME", "NAMESPACE", "PROVIDER", "VERSION", "PHASE", "AGE").WithWriter(w)
	for i := range instances {
		inst := &instances[i]
		tbl.AddRow(
			instanceName(inst),
			instanceNamespace(inst),
			instanceProvider(inst),
			instanceVersionValue(inst),
			instancePhaseValue(inst),
			instanceAge(inst),
		)
	}
	tbl.Print()
}

func instanceNamespace(inst *client.Instance) string {
	return metadataStringField(inst, "namespace")
}

func instanceProvider(inst *client.Instance) string {
	if inst.Spec.Provider == nil || *inst.Spec.Provider == "" {
		return "-"
	}
	return *inst.Spec.Provider
}

func instanceVersionValue(inst *client.Instance) string {
	if inst.Status == nil || inst.Status.Version == nil || *inst.Status.Version == "" {
		return "-"
	}
	return *inst.Status.Version
}

func instancePhaseValue(inst *client.Instance) string {
	if inst.Status == nil || inst.Status.Phase == nil {
		return "-"
	}
	return string(*inst.Status.Phase)
}

func instanceAge(inst *client.Instance) string {
	ts := metadataStringField(inst, "creationTimestamp")
	if ts == "-" {
		return "-"
	}
	created, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return "-"
	}
	return duration.ShortHumanDuration(time.Since(created))
}

func metadataStringField(inst *client.Instance, key string) string {
	if inst.Metadata == nil {
		return "-"
	}
	v, ok := (*inst.Metadata)[key]
	if !ok {
		return "-"
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "-"
	}
	return s
}
