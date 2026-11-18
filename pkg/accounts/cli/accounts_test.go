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

package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/pkg/accounts"
)

func TestUsernamePasswordSanitation(t *testing.T) {
	t.Parallel()
	t.Run("Username", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name        string
			username    string
			expectedErr error
		}{
			{"invalid_with_spaces", "b ob", ErrInvalidUsername},
			{"invalid_non_latin_chars", "аккаунт", ErrInvalidUsername},
			{"invalid_special_chars", "bob!!", ErrInvalidUsername},
			{"invalid_short", "f", ErrInvalidUsername},
			{"invalid_empty", "", ErrInvalidUsername},
			{"valid", "bob1", nil},
			{"valid_with_underscore", "bruce_wayne11", nil},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := ValidateUsername(tc.username)
				if tc.expectedErr == nil {
					require.ErrorIs(t, err, tc.expectedErr)
				}
			})
		}
	})

	t.Run("Password validation", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name        string
			password    string
			expectedErr error
		}{
			{"invalid_short", "pass", ErrInvalidNewPassword},
			{"invalid_short_7_characters", "pass123", ErrInvalidNewPassword},
			{"invalid_with_spaces", "password with spaces", ErrInvalidNewPassword},
			{"invalid_non_latin_chars", "пароль", ErrInvalidNewPassword},
			{"invalid_empty", "", ErrInvalidNewPassword},
			{"valid_lower_case", "verysecurepassword", nil},
			{"valid_upper_case", "VERYSECUREPASSWORD", nil},
			{"valid_lower_case_with_special_chars", "^v#r4$ec*u%ep@s+sw_o&!d=-", nil},
			{"valid_upper_case_with_special_chars", "^V#R4$EC*U%EP@S+SW_O&!D=-", nil},
			{"valid_mixed_case_with_special_chars", "^V#R4$Ec*U%Ep@S+sW_o&!d=-", nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := ValidatePassword(tc.password)
				if tc.expectedErr == nil {
					require.ErrorIs(t, err, tc.expectedErr)
				}
			})
		}
	})
}

func TestAccounts_Render_JSON(t *testing.T) {
	t.Parallel()

	accList := []AccountInfo{
		{
			User:         "admin",
			Capabilities: []accounts.AccountCapability{accounts.AccountCapabilityLogin},
			Enabled:      true,
		},
		{
			User:         "operator",
			Capabilities: []accounts.AccountCapability{accounts.AccountCapabilityAPIKey},
			Enabled:      false,
		},
	}

	acc := &Accounts{
		config: Config{Pretty: false},
	}

	var buf bytes.Buffer
	acc.Render(&buf, accList, ListOptions{})

	assert.JSONEq(t, `[{"user":"admin","capabilities":["login"],"enabled":true},{"user":"operator","capabilities":["apiKey"],"enabled":false}]`, buf.String())

	var decoded []AccountInfo
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	require.Len(t, decoded, 2)
	assert.Equal(t, "admin", decoded[0].User)
	assert.Equal(t, []accounts.AccountCapability{accounts.AccountCapabilityLogin}, decoded[0].Capabilities)
	assert.True(t, decoded[0].Enabled)
	assert.Equal(t, "operator", decoded[1].User)
	assert.Equal(t, []accounts.AccountCapability{accounts.AccountCapabilityAPIKey}, decoded[1].Capabilities)
	assert.False(t, decoded[1].Enabled)
}

func TestAccounts_Render_JSON_Empty(t *testing.T) {
	t.Parallel()

	acc := &Accounts{
		config: Config{Pretty: false},
	}

	var buf bytes.Buffer
	acc.Render(&buf, []AccountInfo{}, ListOptions{})
	assert.Equal(t, "[]\n", buf.String())

	var decoded []AccountInfo
	err := json.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)
	assert.Empty(t, decoded)

	buf.Reset()
	acc.Render(&buf, nil, ListOptions{})
	assert.Equal(t, "[]\n", buf.String())
}

func TestAccounts_Render_Table(t *testing.T) {
	t.Parallel()

	accList := []AccountInfo{
		{
			User:         "admin",
			Capabilities: []accounts.AccountCapability{accounts.AccountCapabilityLogin},
			Enabled:      true,
		},
		{
			User:         "operator",
			Capabilities: []accounts.AccountCapability{accounts.AccountCapabilityAPIKey},
			Enabled:      false,
		},
	}

	acc := &Accounts{
		config: Config{Pretty: true},
	}

	var buf bytes.Buffer
	acc.Render(&buf, accList, ListOptions{})

	out := buf.String()
	assert.Contains(t, out, "USER")
	assert.Contains(t, out, "CAPABILITIES")
	assert.Contains(t, out, "ENABLED")
	assert.Contains(t, out, "admin")
	assert.Contains(t, out, "true")
	assert.Contains(t, out, "operator")
	assert.Contains(t, out, "apiKey")
	assert.Contains(t, out, "false")
}

func TestAccounts_Render_Table_NoHeaders(t *testing.T) {
	t.Parallel()

	accList := []AccountInfo{
		{
			User:         "admin",
			Capabilities: []accounts.AccountCapability{accounts.AccountCapabilityLogin},
			Enabled:      true,
		},
	}

	acc := &Accounts{
		config: Config{Pretty: true},
	}

	var buf bytes.Buffer
	acc.Render(&buf, accList, ListOptions{NoHeaders: true})

	out := buf.String()
	assert.NotContains(t, out, "USER")
	assert.NotContains(t, out, "CAPABILITIES")
	assert.Contains(t, out, "admin")
}

func TestAccounts_Render_Table_CustomColumns(t *testing.T) {
	t.Parallel()

	accList := []AccountInfo{
		{
			User:         "admin",
			Capabilities: []accounts.AccountCapability{accounts.AccountCapabilityLogin},
			Enabled:      true,
		},
	}

	acc := &Accounts{
		config: Config{Pretty: true},
	}

	var buf bytes.Buffer
	acc.Render(&buf, accList, ListOptions{Columns: []string{ColumnUser, ColumnEnabled}})

	out := buf.String()
	assert.Contains(t, out, "USER")
	assert.Contains(t, out, "ENABLED")
	assert.NotContains(t, out, "CAPABILITIES")
	assert.Contains(t, out, "admin")
}

func TestAccounts_Render_Table_Empty(t *testing.T) {
	t.Parallel()

	acc := &Accounts{
		config: Config{Pretty: true},
	}

	var buf bytes.Buffer
	acc.Render(&buf, []AccountInfo{}, ListOptions{})

	out := buf.String()
	assert.Contains(t, out, "USER")
	assert.Contains(t, out, "CAPABILITIES")
	assert.Contains(t, out, "ENABLED")

	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 1)
}
