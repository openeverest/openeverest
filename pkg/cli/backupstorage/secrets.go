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
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// cleanupTimeout bounds deleting the Secrets of a create that failed.
const cleanupTimeout = 30 * time.Second

func credentialsSecretName(storage string) string {
	return "backup-storage-" + storage + "-credentials"
}

func caSecretName(storage string) string {
	return "backup-storage-" + storage + "-ca"
}

// readCredentialsFile reads a --credentials-file.
func readCredentialsFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}
	creds, err := parseCredentialsFile(data)
	if err != nil {
		return nil, fmt.Errorf("credentials file %s: %w", path, err)
	}
	return creds, nil
}

// parseCredentialsFile parses KEY=VALUE lines, skipping blank lines and #
// comments. Errors name the line, never its content, which may be a secret.
func parseCredentialsFile(data []byte) (map[string]string, error) {
	creds := make(map[string]string)
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", i+1)
		}
		if _, dup := creds[key]; dup {
			return nil, fmt.Errorf("line %d: %s is set more than once", i+1, key)
		}
		creds[key] = unquote(strings.TrimSpace(value))
	}
	if len(creds) == 0 {
		return nil, errors.New("no KEY=VALUE entries")
	}
	return creds, nil
}

func unquote(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	return v
}

// readCAFile reads a --ca-file, which must hold PEM data.
func readCAFile(path string) (string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("failed to read CA file: %w", err)
	}
	if block, _ := pem.Decode(data); block == nil {
		return "", fmt.Errorf("CA file %s holds no PEM data", path)
	}
	return string(data), nil
}

// createSecret creates an Opaque Secret through the Secrets API. refFlag is
// the flag that references an existing Secret instead, for the conflict hint.
func createSecret(
	ctx context.Context, c *client.ClientWithResponses, cluster, namespace, name, refFlag string, data map[string]string,
) error {
	secret := client.Secret{
		Metadata:   &client.ObjectMeta{Name: name, Namespace: namespace},
		Type:       new("Opaque"),
		StringData: &data,
	}
	resp, err := c.CreateSecretWithResponse(ctx, cluster, namespace, secret)
	if err != nil {
		return fmt.Errorf("create secret request failed: %w", err)
	}
	switch resp.StatusCode() {
	case http.StatusCreated:
		return nil
	case http.StatusConflict:
		return fmt.Errorf("secret %q already exists in namespace %q: reference it with --%s, or delete it first", name, namespace, refFlag)
	}
	if msg, ok := clienterr.Message(resp.JSON400, resp.JSON403, resp.JSON404, resp.JSON500); ok {
		return fmt.Errorf("server error creating secret %q: %s", name, msg)
	}
	return fmt.Errorf("unexpected response creating secret %q: %s", name, resp.Status())
}

// deleteSecrets removes the Secrets a failed create made, best effort: a
// failure is reported as a warning, and the cancellation that may have caused
// the failure does not stop the cleanup.
func (cr *CreateRunner) deleteSecrets(ctx context.Context, c *client.ClientWithResponses, opts CreateOptions, names []string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	for _, name := range names {
		resp, err := c.DeleteSecretWithResponse(ctx, opts.Cluster, opts.Namespace, name)
		if err == nil && (resp.StatusCode() == http.StatusNoContent || resp.StatusCode() == http.StatusNotFound) {
			cr.l.Infof("deleted secret %q in namespace %q", name, opts.Namespace)
			continue
		}
		reason := fmt.Sprint(err)
		if err == nil {
			reason = resp.Status()
			if msg, ok := clienterr.Message(resp.JSON403, resp.JSON500); ok {
				reason = msg
			}
		}
		output.PrintWarn(fmt.Sprintf("failed to delete secret %q in namespace %q, delete it by hand: %s", name, opts.Namespace, reason),
			cr.l, cr.config.Pretty,
		)
	}
}
