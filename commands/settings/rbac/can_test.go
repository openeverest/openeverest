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

package rbac

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRenderCan_Pretty(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderCan(&buf, true, true)
	assert.Equal(t, "Yes\n", buf.String())

	buf.Reset()
	renderCan(&buf, false, true)
	assert.Equal(t, "No\n", buf.String())
}

func TestRenderCan_JSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderCan(&buf, true, false)
	assert.JSONEq(t, `{"can":true}`, buf.String())

	var res CanResult
	err := json.Unmarshal(buf.Bytes(), &res)
	require.NoError(t, err)
	assert.True(t, res.Can)

	buf.Reset()
	renderCan(&buf, false, false)
	assert.JSONEq(t, `{"can":false}`, buf.String())

	err = json.Unmarshal(buf.Bytes(), &res)
	require.NoError(t, err)
	assert.False(t, res.Can)
}

func TestSettingsRBACCanPreRunE_ActionValidation(t *testing.T) {
	t.Parallel()

	validActions := []string{"create", "read", "update", "delete", "*"}
	for _, action := range validActions {
		cmd := &cobra.Command{}
		err := settingsRBACCanPreRunE(cmd, []string{"alice", action, "database-clusters", "default/db1"})
		require.NoError(t, err, "expected action %q to be valid", action)
	}

	cmd := &cobra.Command{}
	err := settingsRBACCanPreRunE(cmd, []string{"alice", "invalid-action", "database-clusters", "default/db1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid action 'invalid-action'")
}

//nolint:paralleltest // mutates package-level rbacCanPretty; must run serially
func TestSettingsRBACCanPreRunE_Flags(t *testing.T) {
	tests := []struct {
		name           string
		verbose        bool
		json           bool
		expectedPretty bool
	}{
		{
			name:           "default flags result in pretty output",
			verbose:        false,
			json:           false,
			expectedPretty: true,
		},
		{
			name:           "--json flag disables pretty output",
			verbose:        false,
			json:           true,
			expectedPretty: false,
		},
		{
			name:           "--verbose flag disables pretty output",
			verbose:        true,
			json:           false,
			expectedPretty: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().BoolP(cli.FlagVerbose, "v", false, "")
			cmd.Flags().Bool(cli.FlagJSON, false, "")
			cmd.Flags().StringP(cli.FlagKubeconfig, "k", "", "")

			if tc.verbose {
				require.NoError(t, cmd.Flags().Set(cli.FlagVerbose, "true"))
			}
			if tc.json {
				require.NoError(t, cmd.Flags().Set(cli.FlagJSON, "true"))
			}

			err := settingsRBACCanPreRunE(cmd, []string{"alice", "read", "instances", "prod/ns1/db1"})
			require.NoError(t, err)
			assert.Equal(t, tc.expectedPretty, rbacCanPretty)
		})
	}
}

func TestSettingsRBACCan_EndToEnd_WithPolicyFile(t *testing.T) {
	t.Parallel()

	policyFile := filepath.Join("..", "..", "..", "pkg", "rbac", "testdata", "policy-1-good.csv")

	// alice has role:readonly and specific permission to create instances in prod/ns1/*
	canRead, err := rbac.Can(context.Background(), policyFile, nil, "alice", "read", "instances", "prod/ns1/db1")
	require.NoError(t, err)
	assert.True(t, canRead)

	var buf bytes.Buffer
	renderCan(&buf, canRead, false)
	assert.JSONEq(t, `{"can":true}`, buf.String())

	buf.Reset()
	renderCan(&buf, canRead, true)
	assert.Equal(t, "Yes\n", buf.String())

	// alice cannot delete instances
	canDelete, err := rbac.Can(context.Background(), policyFile, nil, "alice", "delete", "instances", "prod/ns1/db1")
	require.NoError(t, err)
	assert.False(t, canDelete)

	buf.Reset()
	renderCan(&buf, canDelete, false)
	assert.JSONEq(t, `{"can":false}`, buf.String())

	buf.Reset()
	renderCan(&buf, canDelete, true)
	assert.Equal(t, "No\n", buf.String())
}
