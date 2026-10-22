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
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/output"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRenderValidate_Pretty(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderValidate(&buf, nil, true)
	assert.Equal(t, output.Success("Valid"), buf.String())

	buf.Reset()
	renderValidate(&buf, errors.New("policy syntax error\nline 2"), true)
	out := buf.String()
	assert.Contains(t, out, "Invalid")
	assert.Contains(t, out, "policy syntax error - line 2")
}

func TestRenderValidate_JSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderValidate(&buf, nil, false)
	assert.JSONEq(t, `{"valid":true}`, buf.String())

	var res ValidateResult
	err := json.Unmarshal(buf.Bytes(), &res)
	require.NoError(t, err)
	assert.True(t, res.Valid)
	assert.Empty(t, res.Error)

	buf.Reset()
	renderValidate(&buf, errors.New("policy syntax error"), false)
	assert.JSONEq(t, `{"valid":false,"error":"policy syntax error"}`, buf.String())

	err = json.Unmarshal(buf.Bytes(), &res)
	require.NoError(t, err)
	assert.False(t, res.Valid)
	assert.Equal(t, "policy syntax error", res.Error)
}

//nolint:paralleltest // mutates package-level rbacValidatePretty; must run serially
func TestSettingsRBACValidatePreRun_Flags(t *testing.T) {
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

			settingsRBACValidatePreRun(cmd, []string{})
			assert.Equal(t, tc.expectedPretty, rbacValidatePretty)
		})
	}
}

func TestSettingsRBACValidate_EndToEnd_WithPolicyFile(t *testing.T) {
	t.Parallel()

	goodFile := filepath.Join("..", "..", "..", "pkg", "rbac", "testdata", "policy-1-good.csv")
	err := rbac.ValidatePolicy(context.Background(), nil, goodFile)
	require.NoError(t, err)

	var buf bytes.Buffer
	renderValidate(&buf, err, false)
	assert.JSONEq(t, `{"valid":true}`, buf.String())

	buf.Reset()
	renderValidate(&buf, err, true)
	assert.Contains(t, buf.String(), "Valid")

	badFile := filepath.Join("..", "..", "..", "pkg", "rbac", "testdata", "policy-2-bad.csv")
	err = rbac.ValidatePolicy(context.Background(), nil, badFile)
	require.Error(t, err)

	buf.Reset()
	renderValidate(&buf, err, false)
	var res ValidateResult
	unmarshalErr := json.Unmarshal(buf.Bytes(), &res)
	require.NoError(t, unmarshalErr)
	assert.False(t, res.Valid)
	assert.NotEmpty(t, res.Error)

	buf.Reset()
	renderValidate(&buf, err, true)
	assert.Contains(t, buf.String(), "Invalid")
}
