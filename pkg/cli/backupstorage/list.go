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

// Package backupstorage provides CLI business logic for backup storage management.
package backupstorage

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

// Config holds the shared configuration for backup storage CLI runners.
type Config struct {
	Pretty bool
}

// ListOptions holds the inputs for the list command.
type ListOptions struct {
	Namespace     string
	AllNamespaces bool
	Cluster       string
	Context       string
}

// ListRunner lists backup storages via the Everest API.
type ListRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewListRunner creates a new ListRunner.
func NewListRunner(cfg Config, l *zap.SugaredLogger) *ListRunner {
	sl := &ListRunner{config: cfg, l: l.With("component", "backup-storage-list")}
	if cfg.Pretty {
		sl.l = zap.NewNop().Sugar()
	}
	return sl
}

// Run fetches backup storages and prints them to stdout, either as a table (pretty mode)
// or as raw JSON. Credentials are never part of the response: the API clears write-only
// credential fields before persisting a BackupStorage, so nothing needs to be redacted here.
func (sl *ListRunner) Run(ctx context.Context, opts ListOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: sl.config.Pretty}, sl.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	namespaces := []string{opts.Namespace}
	if opts.AllNamespaces {
		if namespaces, err = sl.listNamespaces(ctx, c, opts.Cluster); err != nil {
			return err
		}
	}

	backupStorages := make([]client.BackupStorage, 0, len(namespaces))
	for _, ns := range namespaces {
		items, err := sl.listBackupStorages(ctx, c, opts.Cluster, ns)
		if err != nil {
			return err
		}
		backupStorages = append(backupStorages, items...)
	}

	sort.Slice(backupStorages, func(i, j int) bool {
		ni, nj := backupStorageNamespace(&backupStorages[i]), backupStorageNamespace(&backupStorages[j])
		if ni != nj {
			return ni < nj
		}
		return backupStorageName(&backupStorages[i]) < backupStorageName(&backupStorages[j])
	})

	var buf bytes.Buffer
	sl.render(&buf, backupStorages)
	fmt.Fprint(os.Stdout, buf.String())
	return nil
}

func (sl *ListRunner) listNamespaces(ctx context.Context, c *client.ClientWithResponses, cluster string) ([]string, error) {
	resp, err := c.ListNamespacesWithResponse(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response listing namespaces: %s", resp.Status())
	}
	return *resp.JSON200, nil
}

func (sl *ListRunner) listBackupStorages(
	ctx context.Context, c *client.ClientWithResponses, cluster, namespace string,
) ([]client.BackupStorage, error) {
	resp, err := c.ListBackupStoragesWithResponse(ctx, cluster, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list backup storages in namespace %q: %w", namespace, err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response listing backup storages in namespace %q: %s", namespace, resp.Status())
	}
	if resp.JSON200.Items == nil {
		return nil, nil
	}
	return *resp.JSON200.Items, nil
}

func (sl *ListRunner) render(w io.Writer, backupStorages []client.BackupStorage) {
	if !sl.config.Pretty {
		_ = json.NewEncoder(w).Encode(backupStorages) //nolint:errchkjson
		return
	}
	printBackupStorageTable(w, backupStorages)
}

func printBackupStorageTable(w io.Writer, backupStorages []client.BackupStorage) {
	tbl := table.New("NAME", "NAMESPACE", "TYPE", "AGE").WithWriter(w)
	for i := range backupStorages {
		bs := &backupStorages[i]
		tbl.AddRow(
			backupStorageName(bs),
			backupStorageNamespace(bs),
			backupStorageType(bs),
			backupStorageAge(bs),
		)
	}
	tbl.Print()
}

func backupStorageName(bs *client.BackupStorage) string {
	return metadataStringField(bs, "name")
}

func backupStorageNamespace(bs *client.BackupStorage) string {
	return metadataStringField(bs, "namespace")
}

func backupStorageType(bs *client.BackupStorage) string {
	if bs.Spec.Type == "" {
		return "-"
	}
	return string(bs.Spec.Type)
}

func backupStorageAge(bs *client.BackupStorage) string {
	ts := metadataStringField(bs, "creationTimestamp")
	if ts == "-" {
		return "-"
	}
	created, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return "-"
	}
	return duration.ShortHumanDuration(time.Since(created))
}

func metadataStringField(bs *client.BackupStorage, key string) string {
	if bs.Metadata == nil {
		return "-"
	}
	v, ok := (*bs.Metadata)[key]
	if !ok {
		return "-"
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "-"
	}
	return s
}
