// everest
// Copyright (C) 2023 Percona LLC
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
	"context"
	"sync"
	"testing"

	"github.com/rodaine/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/pkg/accounts"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// memoryAccounts is a small in-memory accounts.Interface for outcome-based tests.
type memoryAccounts struct {
	mu       sync.Mutex
	accounts map[string]*accounts.Account
	secure   map[string]bool
}

func newMemoryAccounts() *memoryAccounts {
	return &memoryAccounts{
		accounts: make(map[string]*accounts.Account),
		secure:   make(map[string]bool),
	}
}

func (m *memoryAccounts) Create(_ context.Context, username, password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.accounts[username]; exists {
		return accounts.ErrUserAlreadyExists
	}
	m.accounts[username] = &accounts.Account{
		Enabled:      true,
		Capabilities: []accounts.AccountCapability{accounts.AccountCapabilityLogin},
		PasswordHash: password,
	}
	m.secure[username] = true
	return nil
}

func (m *memoryAccounts) Get(_ context.Context, username string) (*accounts.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[username]
	if !ok {
		return nil, accounts.ErrAccountNotFound
	}
	cp := *acc
	return &cp, nil
}

func (m *memoryAccounts) List(_ context.Context) (map[string]*accounts.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]*accounts.Account, len(m.accounts))
	for user, acc := range m.accounts {
		cp := *acc
		out[user] = &cp
	}
	return out, nil
}

func (m *memoryAccounts) Delete(_ context.Context, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.accounts[username]; !ok {
		return accounts.ErrAccountNotFound
	}
	delete(m.accounts, username)
	delete(m.secure, username)
	return nil
}

func (m *memoryAccounts) SetPassword(_ context.Context, username, newPassword string, secure bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.accounts[username]
	if !ok {
		return accounts.ErrAccountNotFound
	}
	acc.PasswordHash = newPassword
	m.secure[username] = secure
	return nil
}

func (m *memoryAccounts) Verify(_ context.Context, username, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.accounts[username]; !ok {
		return accounts.ErrAccountNotFound
	}
	return nil
}

func (m *memoryAccounts) IsSecure(_ context.Context, username string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.accounts[username]; !ok {
		return false, accounts.ErrAccountNotFound
	}
	return m.secure[username], nil
}

// seed puts an account into the store without going through Create (for init-admin cases).
func (m *memoryAccounts) seed(username string, acc *accounts.Account, secure bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.accounts[username] = acc
	m.secure[username] = secure
}

// newTestAccounts bypasses NewAccounts (which needs a real k8s cluster).
func newTestAccounts(mgr accounts.Interface) *Accounts {
	return &Accounts{
		accountManager: mgr,
		l:              zap.NewNop().Sugar(),
		config:         Config{},
	}
}

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
				if tc.expectedErr != nil {
					require.ErrorIs(t, err, tc.expectedErr)
				} else {
					require.NoError(t, err)
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
				if tc.expectedErr != nil {
					require.ErrorIs(t, err, tc.expectedErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		t.Parallel()
		mgr := newMemoryAccounts()
		a := newTestAccounts(mgr)
		err := a.Create(ctx, CreateOptions{Username: "x", Password: "validpassword"})
		require.ErrorIs(t, err, ErrInvalidUsername)
		_, getErr := mgr.Get(ctx, "x")
		require.ErrorIs(t, getErr, accounts.ErrAccountNotFound)
	})

	t.Run("invalid_password", func(t *testing.T) {
		t.Parallel()
		mgr := newMemoryAccounts()
		a := newTestAccounts(mgr)
		err := a.Create(ctx, CreateOptions{Username: "validuser", Password: "bad"})
		require.ErrorIs(t, err, ErrInvalidNewPassword)
		_, getErr := mgr.Get(ctx, "validuser")
		require.ErrorIs(t, getErr, accounts.ErrAccountNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mgr := newMemoryAccounts()
		a := newTestAccounts(mgr)
		err := a.Create(ctx, CreateOptions{Username: "validuser", Password: "validpassword"})
		require.NoError(t, err)
		acc, getErr := mgr.Get(ctx, "validuser")
		require.NoError(t, getErr)
		assert.Equal(t, "validpassword", acc.PasswordHash)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		t.Parallel()
		a := newTestAccounts(newMemoryAccounts())
		err := a.Delete(ctx, "x")
		require.ErrorIs(t, err, ErrInvalidUsername)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mgr := newMemoryAccounts()
		a := newTestAccounts(mgr)
		require.NoError(t, a.Create(ctx, CreateOptions{Username: "validuser", Password: "validpassword"}))
		require.NoError(t, a.Delete(ctx, "validuser"))
		_, getErr := mgr.Get(ctx, "validuser")
		require.ErrorIs(t, getErr, accounts.ErrAccountNotFound)
	})
}

func TestSetPassword(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		t.Parallel()
		a := newTestAccounts(newMemoryAccounts())
		err := a.SetPassword(ctx, SetPasswordOptions{Username: "x", NewPassword: "validpassword"})
		require.ErrorIs(t, err, ErrInvalidUsername)
	})

	t.Run("invalid_password", func(t *testing.T) {
		t.Parallel()
		mgr := newMemoryAccounts()
		a := newTestAccounts(mgr)
		require.NoError(t, a.Create(ctx, CreateOptions{Username: "validuser", Password: "validpassword"}))
		err := a.SetPassword(ctx, SetPasswordOptions{Username: "validuser", NewPassword: "bad"})
		require.ErrorIs(t, err, ErrInvalidNewPassword)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mgr := newMemoryAccounts()
		// Seed as insecure so SetPassword(secure=true) has an observable effect.
		mgr.seed("validuser", &accounts.Account{
			Enabled:      true,
			Capabilities: []accounts.AccountCapability{accounts.AccountCapabilityLogin},
			PasswordHash: "oldpassword",
		}, false)
		a := newTestAccounts(mgr)
		require.NoError(t, a.SetPassword(ctx, SetPasswordOptions{Username: "validuser", NewPassword: "newpassword"}))
		secure, err := mgr.IsSecure(ctx, "validuser")
		require.NoError(t, err)
		assert.True(t, secure)
		acc, err := mgr.Get(ctx, "validuser")
		require.NoError(t, err)
		assert.Equal(t, "newpassword", acc.PasswordHash)
	})
}

//nolint:paralleltest // swaps table.DefaultWriter
func TestListOutput(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	orig := table.DefaultWriter
	table.DefaultWriter = &buf
	t.Cleanup(func() { table.DefaultWriter = orig })

	mgr := newMemoryAccounts()
	mgr.seed("alice", &accounts.Account{
		Enabled:      true,
		Capabilities: []accounts.AccountCapability{accounts.AccountCapabilityLogin},
	}, true)
	a := newTestAccounts(mgr)

	t.Run("default_columns_with_headers", func(t *testing.T) {
		buf.Reset()
		require.NoError(t, a.List(ctx, ListOptions{}))
		out := buf.String()
		assert.Contains(t, out, "USER")
		assert.Contains(t, out, "CAPABILITIES")
		assert.Contains(t, out, "ENABLED")
		assert.Contains(t, out, "alice")
	})

	t.Run("no_headers", func(t *testing.T) {
		buf.Reset()
		require.NoError(t, a.List(ctx, ListOptions{NoHeaders: true}))
		out := buf.String()
		assert.NotContains(t, out, "USER")
		assert.Contains(t, out, "alice")
	})

	t.Run("custom_columns", func(t *testing.T) {
		buf.Reset()
		require.NoError(t, a.List(ctx, ListOptions{Columns: []string{ColumnUser, ColumnEnabled}}))
		out := buf.String()
		assert.Contains(t, out, "USER")
		assert.Contains(t, out, "ENABLED")
		assert.NotContains(t, out, "CAPABILITIES")
		assert.Contains(t, out, "alice")
	})
}

func TestGetInitAdminPassword(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("already_secure", func(t *testing.T) {
		t.Parallel()
		mgr := newMemoryAccounts()
		mgr.seed(common.EverestAdminUser, &accounts.Account{PasswordHash: "hash123"}, true)
		a := newTestAccounts(mgr)
		_, err := a.GetInitAdminPassword(ctx)
		require.ErrorContains(t, err, "cannot retrieve admin password")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mgr := newMemoryAccounts()
		mgr.seed(common.EverestAdminUser, &accounts.Account{PasswordHash: "hash123"}, false)
		a := newTestAccounts(mgr)
		hash, err := a.GetInitAdminPassword(ctx)
		require.NoError(t, err)
		require.Equal(t, "hash123", hash)
	})

	t.Run("refuses_after_password_secured", func(t *testing.T) {
		t.Parallel()
		mgr := newMemoryAccounts()
		mgr.seed(common.EverestAdminUser, &accounts.Account{PasswordHash: "hash123"}, false)
		a := newTestAccounts(mgr)

		hash, err := a.GetInitAdminPassword(ctx)
		require.NoError(t, err)
		require.Equal(t, "hash123", hash)

		require.NoError(t, a.SetPassword(ctx, SetPasswordOptions{
			Username:    common.EverestAdminUser,
			NewPassword: "newsecurepassword",
		}))
		_, err = a.GetInitAdminPassword(ctx)
		require.ErrorContains(t, err, "cannot retrieve admin password")
	})
}
