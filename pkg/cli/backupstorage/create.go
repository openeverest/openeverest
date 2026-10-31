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
	"strings"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// CreateOptions holds the inputs for the create command. Exactly one of
// CredentialsSecret and CredentialsFile is set, and at most one of CASecret
// (NAME[:KEY]), CAFile and InsecureSkipTLSVerify.
type CreateOptions struct {
	Name                  string
	Namespace             string
	Cluster               string
	Context               string
	Class                 string
	Bucket                string
	Prefix                string
	Parameters            ParameterOptions
	AuthMode              string
	CredentialsSecret     string
	CredentialsFile       string
	CASecret              string
	CAFile                string
	InsecureSkipTLSVerify bool
}

// createRequest is a validated create: the BackupStorage to send and the
// content of the Secrets to create before it.
type createRequest struct {
	storage     client.BackupStorage
	credentials map[string]string // read from --credentials-file
	caBundle    string            // read from --ca-file
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

// Run creates the Secrets read from --credentials-file and --ca-file, then the
// BackupStorage referencing them. The Secrets are deleted again if the
// BackupStorage is not created.
func (cr *CreateRunner) Run(ctx context.Context, opts CreateOptions, cfgPath string) error {
	req, err := newCreateRequest(opts)
	if err != nil {
		return err
	}

	c, err := authcli.NewAPIClient(authcli.Config{Pretty: cr.config.Pretty}, cr.l.Desugar().Sugar(), cfgPath, opts.Context)
	if err != nil {
		return err
	}

	class, err := getStorageClass(ctx, c, opts.Cluster, opts.Class)
	if err != nil {
		return err
	}
	if err := checkAuthMode(opts.Class, class, opts.AuthMode, req.credentials); err != nil {
		return err
	}

	secrets, err := cr.createSecrets(ctx, c, opts, req)
	if err != nil {
		return err
	}

	created, err := cr.createBackupStorage(ctx, c, opts, req.storage, secrets)
	if err != nil {
		return err
	}

	cr.l.Infof("created backup storage %q in namespace %q", opts.Name, opts.Namespace)
	return cr.emitCreated(created, opts.Name, secrets)
}

// newCreateRequest validates opts and reads the files they name, before
// anything is sent to the server.
func newCreateRequest(opts CreateOptions) (createRequest, error) {
	var req createRequest
	if err := validateCreateSources(opts); err != nil {
		return req, err
	}
	params, err := opts.Parameters.build()
	if err != nil {
		return req, err
	}
	if opts.CredentialsFile != "" {
		if req.credentials, err = readCredentialsFile(opts.CredentialsFile); err != nil {
			return req, err
		}
	}
	if opts.CAFile != "" {
		if req.caBundle, err = readCAFile(opts.CAFile); err != nil {
			return req, err
		}
	}
	req.storage, err = newBackupStorage(opts, params)
	return req, err
}

func validateCreateSources(opts CreateOptions) error {
	switch {
	case opts.CredentialsSecret == "" && opts.CredentialsFile == "":
		return fmt.Errorf("credentials are required: pass --%s NAME to reference an existing Secret, or --%s PATH to create one",
			cli.FlagBackupStorageCredentialsSecret, cli.FlagBackupStorageCredentialsFile,
		)
	case opts.CredentialsSecret != "" && opts.CredentialsFile != "":
		return fmt.Errorf("--%s and --%s are mutually exclusive", cli.FlagBackupStorageCredentialsSecret, cli.FlagBackupStorageCredentialsFile)
	case opts.CASecret != "" && opts.CAFile != "":
		return fmt.Errorf("--%s and --%s are mutually exclusive", cli.FlagBackupStorageCASecret, cli.FlagBackupStorageCAFile)
	case opts.InsecureSkipTLSVerify && (opts.CASecret != "" || opts.CAFile != ""):
		return fmt.Errorf("--%s cannot be combined with --%s or --%s",
			cli.FlagBackupStorageInsecureSkipTLSVerify, cli.FlagBackupStorageCASecret, cli.FlagBackupStorageCAFile,
		)
	}
	return nil
}

// newBackupStorage builds the request body for a create call.
func newBackupStorage(opts CreateOptions, params map[string]any) (client.BackupStorage, error) {
	bs := client.BackupStorage{
		Metadata: &client.ObjectMeta{Name: opts.Name, Namespace: opts.Namespace},
	}
	bs.Spec.BackupStorageClassRef.Name = opts.Class
	bs.Spec.Location.Bucket = opts.Bucket
	if opts.Prefix != "" {
		bs.Spec.Location.Prefix = new(opts.Prefix)
	}
	if len(params) > 0 {
		bs.Spec.Parameters = &params
	}

	credentialsSecret := opts.CredentialsSecret
	if opts.CredentialsFile != "" {
		credentialsSecret = credentialsSecretName(opts.Name)
	}
	bs.Spec.Authentication.Mode = opts.AuthMode
	bs.Spec.Authentication.CredentialsSecretRef = &struct {
		Name string `json:"name"`
	}{Name: credentialsSecret}

	var caName, caKey string
	switch {
	case opts.CAFile != "":
		caName, caKey = caSecretName(opts.Name), defaultCAKey
	case opts.CASecret != "":
		var err error
		if caName, caKey, err = parseCASecret(opts.CASecret); err != nil {
			return client.BackupStorage{}, err
		}
	}

	if caName == "" && !opts.InsecureSkipTLSVerify {
		return bs, nil
	}
	bs.Spec.Tls = &struct {
		Ca *struct {
			SecretKeyRef *struct {
				Key  string `json:"key"`
				Name string `json:"name"`
			} `json:"secretKeyRef,omitempty"`
		} `json:"ca,omitempty"`
		InsecureSkipVerify *bool `json:"insecureSkipVerify,omitempty"`
	}{}
	if opts.InsecureSkipTLSVerify {
		bs.Spec.Tls.InsecureSkipVerify = new(true)
		return bs, nil
	}
	bs.Spec.Tls.Ca = &struct {
		SecretKeyRef *struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"secretKeyRef,omitempty"`
	}{SecretKeyRef: &struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}{Key: caKey, Name: caName}}
	return bs, nil
}

// getStorageClass fetches the BackupStorageClass a create names, so an unknown
// class fails before any Secret is created.
func getStorageClass(ctx context.Context, c *client.ClientWithResponses, cluster, name string) (*client.BackupStorageClass, error) {
	resp, err := c.GetBackupStorageClassWithResponse(ctx, cluster, name)
	if err != nil {
		return nil, fmt.Errorf("get backup storage class request failed: %w", err)
	}
	switch {
	case resp.StatusCode() == http.StatusNotFound:
		return nil, fmt.Errorf("backup storage class %q not found", name)
	case resp.StatusCode() != http.StatusOK || resp.JSON200 == nil:
		if msg, ok := clienterr.Message(resp.JSON403, resp.JSONDefault); ok {
			return nil, fmt.Errorf("server error: %s", msg)
		}
		return nil, fmt.Errorf("unexpected response getting backup storage class %q: %s", name, resp.Status())
	}
	return resp.JSON200, nil
}

// checkAuthMode fails unless the class lists mode and, when the credentials
// come from a file, the file holds every key the mode requires.
func checkAuthMode(className string, class *client.BackupStorageClass, mode string, credentials map[string]string) error {
	modes := make([]string, 0, len(class.Spec.AuthModes))
	for _, m := range class.Spec.AuthModes {
		if m.Name != mode {
			modes = append(modes, m.Name)
			continue
		}
		if m.Credentials == nil || credentials == nil {
			return nil
		}
		var missing []string
		for _, key := range m.Credentials.Required {
			if credentials[key] == "" {
				missing = append(missing, key)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("credentials file is missing %s, required by authentication mode %s", strings.Join(missing, ", "), mode)
		}
		return nil
	}
	return fmt.Errorf("backup storage class %q has no authentication mode %q (supported: %s)", className, mode, strings.Join(modes, ", "))
}

// createSecrets creates the Secrets read from --credentials-file and --ca-file
// and returns their names. If the second fails, the first is deleted again.
func (cr *CreateRunner) createSecrets(ctx context.Context, c *client.ClientWithResponses, opts CreateOptions, req createRequest) ([]string, error) {
	var created []string
	if req.credentials != nil {
		name := credentialsSecretName(opts.Name)
		if err := createSecret(ctx, c, opts.Cluster, opts.Namespace, name, cli.FlagBackupStorageCredentialsSecret, req.credentials); err != nil {
			return nil, err
		}
		cr.l.Infof("created secret %q in namespace %q", name, opts.Namespace)
		created = append(created, name)
	}
	if req.caBundle != "" {
		name := caSecretName(opts.Name)
		data := map[string]string{defaultCAKey: req.caBundle}
		if err := createSecret(ctx, c, opts.Cluster, opts.Namespace, name, cli.FlagBackupStorageCASecret, data); err != nil {
			cr.deleteSecrets(ctx, c, opts, created)
			return nil, err
		}
		cr.l.Infof("created secret %q in namespace %q", name, opts.Namespace)
		created = append(created, name)
	}
	return created, nil
}

// createBackupStorage sends the BackupStorage. Unless the server answers 201,
// the Secrets this run created for it are deleted again.
func (cr *CreateRunner) createBackupStorage(
	ctx context.Context, c *client.ClientWithResponses, opts CreateOptions, bs client.BackupStorage, secrets []string,
) (*client.BackupStorage, error) {
	resp, err := c.CreateBackupStorageWithResponse(ctx, opts.Cluster, opts.Namespace, bs)
	if err != nil || resp.StatusCode() != http.StatusCreated {
		cr.deleteSecrets(ctx, c, opts, secrets)
	}
	if err != nil {
		return nil, fmt.Errorf("create backup storage request failed: %w", err)
	}
	if resp.StatusCode() == http.StatusConflict {
		return nil, fmt.Errorf("backup storage %q already exists in namespace %q", opts.Name, opts.Namespace)
	}
	if resp.StatusCode() != http.StatusCreated || resp.JSON201 == nil {
		return nil, createStorageError(resp)
	}
	return resp.JSON201, nil
}

// createStorageError maps a non-201 response to an error.
func createStorageError(resp *client.CreateBackupStorageResponse) error {
	if msg, ok := clienterr.Message(resp.JSON400, resp.JSON403, resp.JSON404, resp.JSON500, resp.JSONDefault); ok {
		return fmt.Errorf("server error: %s", msg)
	}
	return fmt.Errorf("unexpected response creating backup storage: %s", resp.Status())
}

// emitCreated reports a successful create: success lines in pretty mode, or the
// created backup storage in JSON mode (so `create --json` is parseable either way).
func (cr *CreateRunner) emitCreated(created *client.BackupStorage, name string, secrets []string) error {
	if cr.config.Pretty {
		for _, secret := range secrets {
			_, _ = fmt.Fprint(os.Stdout, output.Success("Secret %q created", secret))
		}
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
