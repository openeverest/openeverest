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

// Package restore provides CLI business logic for restore management.
package restore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rodaine/table"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clientmeta"
)

// Config holds the shared configuration for restore CLI runners.
type Config struct {
	Pretty bool
}

// ListOptions holds the inputs for the list command.
type ListOptions struct {
	Namespace string
	Instance  string
	Cluster   string
	Context   string
}

// ListRunner lists restores via the Everest API.
type ListRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewListRunner creates a new ListRunner.
func NewListRunner(cfg Config, l *zap.SugaredLogger) *ListRunner {
	rl := &ListRunner{config: cfg, l: l.With("component", "restore-list")}
	if cfg.Pretty {
		rl.l = zap.NewNop().Sugar()
	}
	return rl
}

// Run fetches restores for the given instance and prints them to stdout, either
// as a table (pretty mode) or as raw JSON.
func (rl *ListRunner) Run(ctx context.Context, opts ListOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: rl.config.Pretty}, rl.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	restores, err := rl.listInstanceRestores(ctx, c, opts.Cluster, opts.Namespace, opts.Instance)
	if err != nil {
		return err
	}

	sortRestoresByRecency(restores)

	var buf bytes.Buffer
	rl.render(&buf, restores)
	fmt.Fprint(os.Stdout, buf.String())
	return nil
}

func (rl *ListRunner) listInstanceRestores(
	ctx context.Context, c *client.ClientWithResponses, cluster, namespace, instance string,
) ([]client.Restore, error) {
	resp, err := c.ListInstanceRestoresWithResponse(ctx, cluster, namespace, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to list restores for instance %q in namespace %q: %w", instance, namespace, err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response listing restores for instance %q in namespace %q: %s", instance, namespace, resp.Status())
	}
	if resp.JSON200.Items == nil {
		return nil, nil
	}
	return *resp.JSON200.Items, nil
}

func (rl *ListRunner) render(w io.Writer, restores []client.Restore) {
	if !rl.config.Pretty {
		_ = json.NewEncoder(w).Encode(restores) //nolint:errchkjson
		return
	}
	printRestoreTable(w, restores)
}

func printRestoreTable(w io.Writer, restores []client.Restore) {
	tbl := table.New("NAME", "NAMESPACE", "INSTANCE", "BACKUP", "STATE", "AGE").WithWriter(w)
	for i := range restores {
		r := &restores[i]
		tbl.AddRow(
			restoreName(r),
			restoreNamespace(r),
			restoreInstance(r),
			restoreBackup(r),
			restoreState(r),
			restoreAge(r),
		)
	}
	tbl.Print()
}

func restoreName(r *client.Restore) string {
	if r.Metadata == nil || r.Metadata.Name == "" {
		return "-"
	}
	return r.Metadata.Name
}

func restoreNamespace(r *client.Restore) string {
	if r.Metadata == nil || r.Metadata.Namespace == "" {
		return "-"
	}
	return r.Metadata.Namespace
}

func restoreInstance(r *client.Restore) string {
	if r.Spec.InstanceRef.Name == "" {
		return "-"
	}
	return r.Spec.InstanceRef.Name
}

func restoreBackup(r *client.Restore) string {
	// Point-in-time restores name a stream rather than a backup, so an empty
	// column here is expected rather than missing data.
	if r.Spec.DataSource.Backup == nil || r.Spec.DataSource.Backup.BackupRef.Name == "" {
		return "-"
	}
	return r.Spec.DataSource.Backup.BackupRef.Name
}

func restoreState(r *client.Restore) string {
	if r.Status == nil || r.Status.State == nil || *r.Status.State == "" {
		return "-"
	}
	return *r.Status.State
}

func restoreAge(r *client.Restore) string {
	created, ok := restoreCreationTime(r)
	if !ok {
		return "-"
	}
	return duration.ShortHumanDuration(time.Since(created))
}

// sortRestoresByRecency orders restores newest-first by creation time, falling
// back to name for stability when timestamps tie or are unavailable.
func sortRestoresByRecency(restores []client.Restore) {
	clientmeta.SortByRecency(restores, func(r *client.Restore) *metav1.ObjectMeta { return r.Metadata })
}

// restoreCreationTime returns the restore's creation time and whether it could
// be determined. Restores without a timestamp sort last.
func restoreCreationTime(r *client.Restore) (time.Time, bool) {
	if r.Metadata == nil || r.Metadata.CreationTimestamp.IsZero() {
		return time.Time{}, false
	}
	return r.Metadata.CreationTimestamp.Time, true
}
