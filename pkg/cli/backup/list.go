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

// Package backup provides CLI business logic for backup management.
package backup

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

// Config holds the shared configuration for backup CLI runners.
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

// ListRunner lists backups via the Everest API.
type ListRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewListRunner creates a new ListRunner.
func NewListRunner(cfg Config, l *zap.SugaredLogger) *ListRunner {
	bl := &ListRunner{config: cfg, l: l.With("component", "backup-list")}
	if cfg.Pretty {
		bl.l = zap.NewNop().Sugar()
	}
	return bl
}

// Run fetches backups for the given instance and prints them to stdout, either
// as a table (pretty mode) or as raw JSON.
func (bl *ListRunner) Run(ctx context.Context, opts ListOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: bl.config.Pretty}, bl.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	backups, err := bl.listInstanceBackups(ctx, c, opts.Cluster, opts.Namespace, opts.Instance)
	if err != nil {
		return err
	}

	sortBackupsByRecency(backups)

	var buf bytes.Buffer
	bl.render(&buf, backups)
	fmt.Fprint(os.Stdout, buf.String())
	return nil
}

func (bl *ListRunner) listInstanceBackups(
	ctx context.Context, c *client.ClientWithResponses, cluster, namespace, instance string,
) ([]client.Backup, error) {
	resp, err := c.ListInstanceBackupsWithResponse(ctx, cluster, namespace, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups for instance %q in namespace %q: %w", instance, namespace, err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response listing backups for instance %q in namespace %q: %s", instance, namespace, resp.Status())
	}
	if resp.JSON200.Items == nil {
		return nil, nil
	}
	return *resp.JSON200.Items, nil
}

func (bl *ListRunner) render(w io.Writer, backups []client.Backup) {
	if !bl.config.Pretty {
		_ = json.NewEncoder(w).Encode(backups) //nolint:errchkjson
		return
	}
	printBackupTable(w, backups)
}

func printBackupTable(w io.Writer, backups []client.Backup) {
	tbl := table.New("NAME", "NAMESPACE", "INSTANCE", "STORAGE", "STATE", "SIZE", "AGE").WithWriter(w)
	for i := range backups {
		b := &backups[i]
		tbl.AddRow(
			backupName(b),
			backupNamespace(b),
			backupInstance(b),
			backupStorage(b),
			backupStateText(b),
			backupSize(b),
			backupAge(b),
		)
	}
	tbl.Print()
}

func backupName(b *client.Backup) string {
	if b.Metadata == nil || b.Metadata.Name == "" {
		return "-"
	}
	return b.Metadata.Name
}

func backupNamespace(b *client.Backup) string {
	if b.Metadata == nil || b.Metadata.Namespace == "" {
		return "-"
	}
	return b.Metadata.Namespace
}

func backupInstance(b *client.Backup) string {
	if b.Spec.InstanceRef.Name == "" {
		return "-"
	}
	return b.Spec.InstanceRef.Name
}

func backupStorage(b *client.Backup) string {
	if b.Spec.StorageRef.Name == "" {
		return "-"
	}
	return b.Spec.StorageRef.Name
}

func backupState(b *client.Backup) client.BackupStatusState {
	if b.Status == nil || b.Status.State == nil || *b.Status.State == "" {
		return ""
	}
	return *b.Status.State
}

func backupStateText(b *client.Backup) string {
	state := backupState(b)
	if state == "" {
		return "-"
	}
	return string(state)
}

func backupSize(b *client.Backup) string {
	if b.Status == nil || b.Status.Size == nil || *b.Status.Size == "" {
		return "-"
	}
	return *b.Status.Size
}

func backupAge(b *client.Backup) string {
	created, ok := backupCreationTime(b)
	if !ok {
		return "-"
	}
	return duration.ShortHumanDuration(time.Since(created))
}

// sortBackupsByRecency orders backups newest-first by creation time, falling
// back to name for stability when timestamps tie or are unavailable.
func sortBackupsByRecency(backups []client.Backup) {
	clientmeta.SortByRecency(backups, func(b *client.Backup) *metav1.ObjectMeta { return b.Metadata })
}

// backupCreationTime returns the backup's creation time and whether it could
// be determined. Backups without a timestamp sort last.
func backupCreationTime(b *client.Backup) (time.Time, bool) {
	if b.Metadata == nil || b.Metadata.CreationTimestamp.IsZero() {
		return time.Time{}, false
	}
	return b.Metadata.CreationTimestamp.Time, true
}
