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

package backupstorage

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// UpdateOptions holds the inputs for the update command. VerifyTLS and ForcePathStyle are pointers: nil means the flag was not passed, so the
// merge patch must not name that field at all and the stored value is left untouched.
type UpdateOptions struct {
	Name              string
	Namespace         string
	Cluster           string
	Context           string
	CredentialsSecret string
	AccessKeyID       string
	SecretAccessKey   string
	VerifyTLS         *bool
	ForcePathStyle    *bool
}

// Updater implements `backupstorage update` business logic.
type Updater struct {
	config Config
	l      *zap.SugaredLogger
}

// NewUpdater returns a new Updater.
func NewUpdater(cfg Config, l *zap.SugaredLogger) *Updater {
	u := &Updater{config: cfg, l: l.With("component", "backup-storage-updater")}
	if cfg.Pretty {
		u.l = zap.NewNop().Sugar()
	}
	return u
}

// Run sends the changed fields as a single merge patch to the PatchBackupStorage endpoint where the server performs the read-merge-write.
func (u *Updater) Run(ctx context.Context, opts UpdateOptions, cfgPath string) error {
	s3 := buildS3Patch(opts)
	if len(s3) == 0 {
		return fmt.Errorf("at least one of --access-key-id, --credentials-secret, --verify-tls, or --force-path-style is required")
	}

	c, err := authcli.NewAPIClient(authcli.Config{Pretty: u.config.Pretty}, u.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	patch := map[string]any{"spec": map[string]any{"s3": s3}}
	resp, err := c.PatchBackupStorageWithApplicationMergePatchPlusJSONBodyWithResponse(ctx, opts.Cluster, opts.Namespace, opts.Name, patch)
	if err != nil {
		return fmt.Errorf("patch backup storage request failed: %w", err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		u.l.Infof("updated backup storage %q in namespace %q", opts.Name, opts.Namespace)
		return u.emitUpdated(resp.JSON200, opts)
	case http.StatusNotFound:
		return fmt.Errorf("backup storage %q not found in namespace %q", opts.Name, opts.Namespace)
	default:
		if msg, ok := clienterr.Message(resp.JSON400, resp.JSON415, resp.JSONDefault); ok {
			return fmt.Errorf("server error: %s", msg)
		}
		return fmt.Errorf("unexpected response updating backup storage %q: %s", opts.Name, resp.Status())
	}
}

// buildS3Patch names only the s3 fields the caller actually set, so the merge patch leaves every other field, 
// including the identity fields this command never exposes a flag for, untouched.
func buildS3Patch(opts UpdateOptions) map[string]any {
	s3 := map[string]any{}
	switch {
	case opts.CredentialsSecret != "":
		s3["credentialsSecretRef"] = map[string]any{"name": opts.CredentialsSecret}
	case opts.AccessKeyID != "":
		s3["accessKeyId"] = opts.AccessKeyID
		s3["secretAccessKey"] = opts.SecretAccessKey
	}
	if opts.VerifyTLS != nil {
		s3["verifyTLS"] = *opts.VerifyTLS
	}
	if opts.ForcePathStyle != nil {
		s3["forcePathStyle"] = *opts.ForcePathStyle
	}
	return s3
}

// emitUpdated reports a successful update
func (u *Updater) emitUpdated(updated *client.BackupStorage, opts UpdateOptions) error {
	if u.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Backup storage %q updated", opts.Name))
		return nil
	}
	if updated == nil {
		return fmt.Errorf("backup storage %q was updated but the server returned an unreadable response body", opts.Name)
	}
	return writeBackupStorageJSON(updated)
}
