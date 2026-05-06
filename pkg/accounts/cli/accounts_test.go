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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/percona/everest/pkg/accounts"
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
			{"invalid_with_spaces", "b ob", accounts.ErrInvalidUsername},
			{"invalid_non_latin_chars", "аккаунт", accounts.ErrInvalidUsername},
			{"invalid_special_chars", "bob!!", accounts.ErrInvalidUsername},
			{"invalid_short", "f", accounts.ErrInvalidUsername},
			{"invalid_empty", "", accounts.ErrInvalidUsername},
			{"valid", "bob1", nil},
			{"valid_with_underscore", "bruce_wayne11", nil},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := accounts.ValidateUsername(tc.username)
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
			{"invalid_short", "pass", accounts.ErrInvalidNewPassword},
			{"invalid_with_spaces", "password with spaces", accounts.ErrInvalidNewPassword},
			{"invalid_non_latin_chars", "пароль", accounts.ErrInvalidNewPassword},
			{"invalid_empty", "", accounts.ErrInvalidNewPassword},
			{"valid_lower_case", "verysecurepassword", nil},
			{"valid_upper_case", "VERYSECUREPASSWORD", nil},
			{"valid_lower_case_with_special_chars", "^v#r4$ec*u%ep@s+sw_o&!d=-", nil},
			{"valid_upper_case_with_special_chars", "^V#R4$EC*U%EP@S+SW_O&!D=-", nil},
			{"valid_mixed_case_with_special_chars", "^V#R4$Ec*U%Ep@S+sW_o&!d=-", nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := accounts.ValidatePassword(tc.password)
				if tc.expectedErr == nil {
					require.ErrorIs(t, err, tc.expectedErr)
				}
			})
		}
	})
}
