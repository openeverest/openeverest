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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// CreateOptions holds the inputs for the create command.
type CreateOptions struct {
	Name              string
	Namespace         string
	Type              string
	Bucket            string
	Region            string
	EndpointURL       string
	VerifyTLS         bool
	ForcePathStyle    bool
	CredentialsSecret string
	AccessKeyID       string
	SecretAccessKey   string
	Cluster           string
	Context           string
}

// CreateRunner creates a BackupStorage via the Everest API.
type CreateRunner struct {
	config Config
	l      *zap.SugaredLogger
}

// NewCreateRunner creates a new CreateRunner.
func NewCreateRunner(cfg Config, l *zap.SugaredLogger) *CreateRunner {
	cr := &CreateRunner{config: cfg, l: l.With("component", "backup-storage-create")}
	if cfg.Pretty {
		cr.l = zap.NewNop().Sugar()
	}
	return cr
}

// Run creates a BackupStorage via the Everest API.
func (cr *CreateRunner) Run(ctx context.Context, opts CreateOptions, cfgPath string) error {
	c, err := authcli.NewAPIClient(authcli.Config{Pretty: cr.config.Pretty}, cr.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	bs, err := newBackupStorage(opts)
	if err != nil {
		return err
	}

	resp, err := c.CreateBackupStorageWithResponse(ctx, opts.Cluster, opts.Namespace, *bs)
	if err != nil {
		return fmt.Errorf("create backup storage request failed: %w", err)
	}

	if resp.StatusCode() == http.StatusConflict {
		return fmt.Errorf("backup storage %q already exists in namespace %q", opts.Name, opts.Namespace)
	}
	if resp.StatusCode() != http.StatusCreated || resp.JSON201 == nil {
		return createStorageError(resp)
	}

	cr.l.Infof("created backup storage %q in namespace %q", opts.Name, opts.Namespace)
	return cr.emitCreated(resp.JSON201, opts.Name)
}

// ValidateType returns an error if t is not a supported storage type. Callers
// should check this before doing any work that shouldn't happen for a request
// that's already doomed, e.g. prompting for a credential. Delegates to the
// generated enum's Valid() so this self-updates with `make gen` instead of
// needing a hand-written value list kept in sync by hand.
func ValidateType(t string) error {
	if !client.BackupStorageSpecType(t).Valid() {
		return fmt.Errorf("unsupported storage type %q", t)
	}
	return nil
}

// newBackupStorage builds the request body for a create call.
func newBackupStorage(opts CreateOptions) (*client.BackupStorage, error) {
	if err := ValidateType(opts.Type); err != nil {
		return nil, err
	}

	bs := client.BackupStorage{
		Metadata: &metav1.ObjectMeta{Name: opts.Name, Namespace: opts.Namespace},
	}
	bs.Spec.Type = client.BackupStorageSpecType(opts.Type)

	switch opts.Type {
	case "s3":
		bs.Spec.S3 = &struct {
			AccessKeyId          *string `json:"accessKeyId,omitempty"` //nolint:revive // must match the generated field name exactly
			Bucket               string  `json:"bucket"`
			CredentialsSecretRef struct {
				Name string `json:"name"`
			} `json:"credentialsSecretRef"`
			EndpointURL     string  `json:"endpointURL"` //nolint:tagliatelle // must mirror the generated tag exactly
			ForcePathStyle  *bool   `json:"forcePathStyle,omitempty"`
			Region          string  `json:"region"`
			SecretAccessKey *string `json:"secretAccessKey,omitempty"`
			VerifyTLS       *bool   `json:"verifyTLS,omitempty"` //nolint:tagliatelle // must mirror the generated tag exactly
		}{
			Bucket:         opts.Bucket,
			Region:         opts.Region,
			EndpointURL:    opts.EndpointURL,
			VerifyTLS:      &opts.VerifyTLS,
			ForcePathStyle: &opts.ForcePathStyle,
		}

		switch {
		case opts.CredentialsSecret != "":
			bs.Spec.S3.CredentialsSecretRef.Name = opts.CredentialsSecret
		case opts.AccessKeyID != "":
			bs.Spec.S3.AccessKeyId = &opts.AccessKeyID
			bs.Spec.S3.SecretAccessKey = &opts.SecretAccessKey
			bs.Spec.S3.CredentialsSecretRef.Name = "backup-storage-" + opts.Name + "-credentials"
		}
	default:
		return nil, fmt.Errorf("no request builder for storage type %q", opts.Type)
	}

	return &bs, nil
}

// createStorageError maps a non-201 response to an error.
func createStorageError(resp *client.CreateBackupStorageResponse) error {
	if msg, ok := clienterr.Message(resp.JSON400, resp.JSON500, resp.JSONDefault); ok {
		return fmt.Errorf("server error: %s", msg)
	}
	return fmt.Errorf("unexpected response creating backup storage: %s", resp.Status())
}

// emitCreated reports a successful create: a success line in pretty mode, or the
// created backup storage in JSON mode (so `create --json` is parseable either way).
func (cr *CreateRunner) emitCreated(created *client.BackupStorage, name string) error {
	if cr.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Backup storage %q created", name))
		return nil
	}
	return writeBackupStorageJSON(created)
}

func writeBackupStorageJSON(bs *client.BackupStorage) error {
	if err := json.NewEncoder(os.Stdout).Encode(bs); err != nil {
		return fmt.Errorf("failed to encode backup storage: %w", err)
	}
	return nil
}
