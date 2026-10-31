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

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
)

// Config holds the shared configuration for backup storage CLI runners.
type Config struct {
	Pretty bool
}

// ListOptions holds the inputs for the list command. BackupClass and Class are
// optional filters.
type ListOptions struct {
	Namespace     string
	AllNamespaces bool
	Cluster       string
	Context       string
	BackupClass   string
	Class         string
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
// or as raw JSON. Credentials are never part of the response: a BackupStorage only
// names the Secret that holds them.
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

	params := &client.ListBackupStoragesParams{}
	if opts.BackupClass != "" {
		params.BackupClass = &opts.BackupClass
	}
	if opts.Class != "" {
		params.BackupStorageClass = &opts.Class
	}

	backupStorages := make([]client.BackupStorage, 0, len(namespaces))
	for _, ns := range namespaces {
		items, err := sl.listBackupStorages(ctx, c, opts.Cluster, ns, params)
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
	ctx context.Context, c *client.ClientWithResponses, cluster, namespace string, params *client.ListBackupStoragesParams,
) ([]client.BackupStorage, error) {
	resp, err := c.ListBackupStoragesWithResponse(ctx, cluster, namespace, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list backup storages in namespace %q: %w", namespace, err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		if msg, ok := clienterr.Message(resp.JSON500, resp.JSONDefault); ok {
			return nil, fmt.Errorf("server error: %s", msg)
		}
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
	tbl := table.New("NAME", "NAMESPACE", "CLASS", "BUCKET", "AUTH", "ACCEPTED", "AGE").WithWriter(w)
	for i := range backupStorages {
		bs := &backupStorages[i]
		tbl.AddRow(
			backupStorageName(bs),
			backupStorageNamespace(bs),
			dashIfEmpty(bs.Spec.BackupStorageClassRef.Name),
			backupStorageBucket(bs),
			dashIfEmpty(bs.Spec.Authentication.Mode),
			backupStorageAccepted(bs),
			backupStorageAge(bs),
		)
	}
	tbl.Print()
}

func backupStorageName(bs *client.BackupStorage) string {
	if bs.Metadata == nil || bs.Metadata.Name == "" {
		return "-"
	}
	return bs.Metadata.Name
}

func backupStorageNamespace(bs *client.BackupStorage) string {
	if bs.Metadata == nil || bs.Metadata.Namespace == "" {
		return "-"
	}
	return bs.Metadata.Namespace
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func backupStorageBucket(bs *client.BackupStorage) string {
	loc := bs.Spec.Location
	if loc.Prefix != nil && *loc.Prefix != "" {
		return loc.Bucket + "/" + *loc.Prefix
	}
	return dashIfEmpty(loc.Bucket)
}

// backupStorageAccepted is the status of the Accepted condition, Unknown until
// the controller has set it.
func backupStorageAccepted(bs *client.BackupStorage) string {
	if bs.Status != nil && bs.Status.Conditions != nil {
		for _, cond := range *bs.Status.Conditions {
			if cond.Type == backupv1alpha1.BackupStorageConditionAccepted {
				return string(cond.Status)
			}
		}
	}
	return string(client.BackupStorageStatusConditionsStatusUnknown)
}

func backupStorageAge(bs *client.BackupStorage) string {
	if bs.Metadata == nil || bs.Metadata.CreationTimestamp.IsZero() {
		return "-"
	}
	return duration.ShortHumanDuration(time.Since(bs.Metadata.CreationTimestamp.Time))
}
