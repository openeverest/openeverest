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

// Package backupclass provides CLI business logic for backup class management.
package backupclass

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rodaine/table"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
)

// Config holds the shared configuration for backup class CLI runners.
type Config struct {
	Pretty bool
}

// ListOptions holds the inputs for the list command.
type ListOptions struct {
	Cluster string
	Context string
}

// ListRunner lists backup classes via the Everest API.
type ListRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewListRunner creates a new ListRunner.
func NewListRunner(cfg Config, l *zap.SugaredLogger) *ListRunner {
	bl := &ListRunner{config: cfg, l: l.With("component", "backup-class-list")}
	if cfg.Pretty {
		bl.l = zap.NewNop().Sugar()
	}
	return bl
}

// Run fetches backup classes and prints them to stdout, either as a table (pretty mode)
// or as raw JSON.
func (bl *ListRunner) Run(ctx context.Context, opts ListOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: bl.config.Pretty}, bl.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	backupClasses, err := bl.listBackupClasses(ctx, c, opts.Cluster)
	if err != nil {
		return err
	}

	sort.Slice(backupClasses, func(i, j int) bool {
		return backupClassName(&backupClasses[i]) < backupClassName(&backupClasses[j])
	})

	var buf bytes.Buffer
	bl.render(&buf, backupClasses)
	fmt.Fprint(os.Stdout, buf.String())
	return nil
}

func (bl *ListRunner) listBackupClasses(ctx context.Context, c *client.ClientWithResponses, cluster string) ([]client.BackupClass, error) {
	resp, err := c.ListBackupClassesWithResponse(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list backup classes: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response listing backup classes: %s", resp.Status())
	}
	if resp.JSON200.Items == nil {
		return nil, nil
	}
	return *resp.JSON200.Items, nil
}

func (bl *ListRunner) render(w io.Writer, backupClasses []client.BackupClass) {
	if !bl.config.Pretty {
		_ = json.NewEncoder(w).Encode(backupClasses) //nolint:errchkjson
		return
	}
	printBackupClassTable(w, backupClasses)
}

func printBackupClassTable(w io.Writer, backupClasses []client.BackupClass) {
	tbl := table.New("NAME", "PROVIDER", "EXECUTION MODE", "AGE").WithWriter(w)
	for i := range backupClasses {
		bc := &backupClasses[i]
		tbl.AddRow(
			backupClassName(bc),
			backupClassProvider(bc),
			backupClassExecutionMode(bc),
			backupClassAge(bc),
		)
	}
	tbl.Print()
}

func backupClassName(bc *client.BackupClass) string {
	if bc.Metadata == nil || bc.Metadata.Name == "" {
		return "-"
	}
	return bc.Metadata.Name
}

func backupClassProvider(bc *client.BackupClass) string {
	if bc.Spec.SupportedProviders == nil || len(*bc.Spec.SupportedProviders) == 0 {
		return "-"
	}
	return strings.Join(*bc.Spec.SupportedProviders, ",")
}

func backupClassExecutionMode(bc *client.BackupClass) string {
	if bc.Spec.ExecutionMode == "" {
		return "-"
	}
	return string(bc.Spec.ExecutionMode)
}

func backupClassAge(bc *client.BackupClass) string {
	if bc.Metadata == nil || bc.Metadata.CreationTimestamp.IsZero() {
		return "-"
	}
	return duration.ShortHumanDuration(time.Since(bc.Metadata.CreationTimestamp.Time))
}
