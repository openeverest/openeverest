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
	"errors"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// UpdateOptions holds the inputs for the update command. InsecureSkipTLSVerify
// is a pointer: nil means the flag was not passed, so the merge patch must not
// name that field at all and the stored value is left untouched.
type UpdateOptions struct {
	Name                  string
	Namespace             string
	Cluster               string
	Context               string
	Parameters            ParameterOptions
	AuthMode              string
	CredentialsSecret     string
	CASecret              string
	InsecureSkipTLSVerify *bool
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
	spec, err := buildSpecPatch(opts)
	if err != nil {
		return err
	}
	if len(spec) == 0 {
		return errors.New("nothing to update: pass at least one of --parameter, --region, --endpoint-url, --force-path-style, " +
			"--implementation, --auth-mode, --credentials-secret, --ca-secret or --insecure-skip-tls-verify")
	}

	c, err := authcli.NewAPIClient(authcli.Config{Pretty: u.config.Pretty}, u.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	patch := map[string]any{"spec": spec}
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
		if msg, ok := clienterr.Message(resp.JSON400, resp.JSON403, resp.JSON415, resp.JSONDefault); ok {
			return fmt.Errorf("server error: %s", msg)
		}
		return fmt.Errorf("unexpected response updating backup storage %q: %s", opts.Name, resp.Status())
	}
}

// buildSpecPatch names only the fields the caller actually set, so the merge
// patch leaves every other field untouched; parameters merge key by key.
func buildSpecPatch(opts UpdateOptions) (map[string]any, error) {
	spec := make(map[string]any)

	params, err := opts.Parameters.build()
	if err != nil {
		return nil, err
	}
	if len(params) > 0 {
		spec["parameters"] = params
	}

	auth := make(map[string]any)
	if opts.AuthMode != "" {
		auth["mode"] = opts.AuthMode
	}
	if opts.CredentialsSecret != "" {
		auth["credentialsSecretRef"] = map[string]any{"name": opts.CredentialsSecret}
	}
	if len(auth) > 0 {
		spec["authentication"] = auth
	}

	tls, err := buildTLSPatch(opts)
	if err != nil {
		return nil, err
	}
	if len(tls) > 0 {
		spec["tls"] = tls
	}
	return spec, nil
}

// buildTLSPatch sets a CA bundle or skipping verification, clearing the
// other, since a storage cannot have both.
func buildTLSPatch(opts UpdateOptions) (map[string]any, error) {
	insecure := opts.InsecureSkipTLSVerify != nil && *opts.InsecureSkipTLSVerify
	tls := make(map[string]any)
	if opts.CASecret != "" {
		if insecure {
			return nil, fmt.Errorf("--%s cannot be combined with --%s", cli.FlagBackupStorageInsecureSkipTLSVerify, cli.FlagBackupStorageCASecret)
		}
		name, key, err := parseCASecret(opts.CASecret)
		if err != nil {
			return nil, err
		}
		tls["ca"] = map[string]any{"secretKeyRef": map[string]any{"name": name, "key": key}}
		tls["insecureSkipVerify"] = nil
	}
	if opts.InsecureSkipTLSVerify != nil {
		tls["insecureSkipVerify"] = insecure
	}
	if insecure {
		tls["ca"] = nil
	}
	return tls, nil
}

// emitUpdated reports a successful update.
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
