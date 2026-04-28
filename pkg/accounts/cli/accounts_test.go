// everest
// Copyright (C) 2023 Percona LLC
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
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/percona/everest/pkg/accounts"
	"github.com/percona/everest/pkg/common"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockAccountManager struct {
	mock.Mock
}

func (m *mockAccountManager) Create(ctx context.Context, username, password string) error {
	return m.Called(ctx, username, password).Error(0)
}

func (m *mockAccountManager) Get(ctx context.Context, username string) (*accounts.Account, error) {
	args := m.Called(ctx, username)
	acc, _ := args.Get(0).(*accounts.Account)
	return acc, args.Error(1)
}

func (m *mockAccountManager) List(ctx context.Context) (map[string]*accounts.Account, error) {
	args := m.Called(ctx)
	list, _ := args.Get(0).(map[string]*accounts.Account)
	return list, args.Error(1)
}

func (m *mockAccountManager) Delete(ctx context.Context, username string) error {
	return m.Called(ctx, username).Error(0)
}

func (m *mockAccountManager) SetPassword(ctx context.Context, username, newPassword string, secure bool) error {
	return m.Called(ctx, username, newPassword, secure).Error(0)
}

func (m *mockAccountManager) Verify(ctx context.Context, username, password string) error {
	return m.Called(ctx, username, password).Error(0)
}

func (m *mockAccountManager) IsSecure(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

type mockRSACreator struct {
	err error
}

func (m *mockRSACreator) CreateRSAKeyPair(_ context.Context) error {
	return m.err
}

// newTestAccounts bypasses NewAccounts (which needs a real k8s cluster).
func newTestAccounts(mgr accounts.Interface, kube rsaKeyCreator, pretty bool) *Accounts {
	return &Accounts{
		accountManager: mgr,
		kubeClient:     kube,
		l:              zap.NewNop().Sugar(),
		config:         Config{Pretty: pretty},
	}
}

// ---------------------------------------------------------------------------
// Existing validation tests (fixed: now asserts both valid AND invalid cases)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// WithAccountManager
// ---------------------------------------------------------------------------

func TestWithAccountManager(t *testing.T) {
	t.Parallel()
	a := newTestAccounts(nil, nil, false)
	mgr := &mockAccountManager{}
	a.WithAccountManager(mgr)
	require.Equal(t, mgr, a.accountManager)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		t.Parallel()
		a := newTestAccounts(nil, nil, false)
		err := a.Create(ctx, CreateOptions{Username: "x", Password: "validpassword"})
		require.ErrorIs(t, err, ErrInvalidUsername)
	})

	t.Run("invalid_password", func(t *testing.T) {
		t.Parallel()
		a := newTestAccounts(nil, nil, false)
		err := a.Create(ctx, CreateOptions{Username: "validuser", Password: "bad"})
		require.ErrorIs(t, err, ErrInvalidNewPassword)
	})

	t.Run("manager_error", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		managerErr := errors.New("create failed")
		mgr.On("Create", ctx, "validuser", "validpassword").Return(managerErr)
		a := newTestAccounts(mgr, nil, false)
		err := a.Create(ctx, CreateOptions{Username: "validuser", Password: "validpassword"})
		require.ErrorIs(t, err, managerErr)
		mgr.AssertExpectations(t)
	})

	t.Run("success_not_pretty", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("Create", ctx, "validuser", "validpassword").Return(nil)
		a := newTestAccounts(mgr, nil, false)
		err := a.Create(ctx, CreateOptions{Username: "validuser", Password: "validpassword"})
		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})

	t.Run("success_pretty", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("Create", ctx, "validuser", "validpassword").Return(nil)
		a := newTestAccounts(mgr, nil, true)
		err := a.Create(ctx, CreateOptions{Username: "validuser", Password: "validpassword"})
		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		t.Parallel()
		a := newTestAccounts(nil, nil, false)
		err := a.Delete(ctx, "x")
		require.ErrorIs(t, err, ErrInvalidUsername)
	})

	t.Run("manager_error", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		deleteErr := errors.New("delete failed")
		mgr.On("Delete", ctx, "validuser").Return(deleteErr)
		a := newTestAccounts(mgr, nil, false)
		err := a.Delete(ctx, "validuser")
		require.ErrorIs(t, err, deleteErr)
		mgr.AssertExpectations(t)
	})

	t.Run("success_not_pretty", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("Delete", ctx, "validuser").Return(nil)
		a := newTestAccounts(mgr, nil, false)
		err := a.Delete(ctx, "validuser")
		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})

	t.Run("success_pretty", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("Delete", ctx, "validuser").Return(nil)
		a := newTestAccounts(mgr, nil, true)
		err := a.Delete(ctx, "validuser")
		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})
}

// ---------------------------------------------------------------------------
// SetPassword
// ---------------------------------------------------------------------------

func TestSetPassword(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		t.Parallel()
		a := newTestAccounts(nil, nil, false)
		err := a.SetPassword(ctx, SetPasswordOptions{Username: "x", NewPassword: "validpassword"})
		require.ErrorIs(t, err, ErrInvalidUsername)
	})

	t.Run("invalid_password", func(t *testing.T) {
		t.Parallel()
		a := newTestAccounts(nil, nil, false)
		err := a.SetPassword(ctx, SetPasswordOptions{Username: "validuser", NewPassword: "bad"})
		require.ErrorIs(t, err, ErrInvalidNewPassword)
	})

	t.Run("manager_error", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		setErr := errors.New("set password failed")
		mgr.On("SetPassword", ctx, "validuser", "validpassword", true).Return(setErr)
		a := newTestAccounts(mgr, nil, false)
		err := a.SetPassword(ctx, SetPasswordOptions{Username: "validuser", NewPassword: "validpassword"})
		require.ErrorIs(t, err, setErr)
		mgr.AssertExpectations(t)
	})

	t.Run("success_not_pretty", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("SetPassword", ctx, "validuser", "validpassword", true).Return(nil)
		a := newTestAccounts(mgr, nil, false)
		err := a.SetPassword(ctx, SetPasswordOptions{Username: "validuser", NewPassword: "validpassword"})
		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})

	t.Run("success_pretty", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("SetPassword", ctx, "validuser", "validpassword", true).Return(nil)
		a := newTestAccounts(mgr, nil, true)
		err := a.SetPassword(ctx, SetPasswordOptions{Username: "validuser", NewPassword: "validpassword"})
		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	accountsList := map[string]*accounts.Account{
		"alice": {Enabled: true, Capabilities: []accounts.AccountCapability{accounts.AccountCapabilityLogin}},
	}

	t.Run("manager_error", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		listErr := errors.New("list failed")
		mgr.On("List", ctx).Return(nil, listErr)
		a := newTestAccounts(mgr, nil, false)
		err := a.List(ctx, ListOptions{})
		require.ErrorIs(t, err, listErr)
		mgr.AssertExpectations(t)
	})

	t.Run("default_columns_with_headers", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("List", ctx).Return(accountsList, nil)
		a := newTestAccounts(mgr, nil, false)
		err := a.List(ctx, ListOptions{})
		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})

	t.Run("no_headers", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("List", ctx).Return(accountsList, nil)
		a := newTestAccounts(mgr, nil, false)
		err := a.List(ctx, ListOptions{NoHeaders: true})
		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})

	t.Run("custom_columns", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("List", ctx).Return(accountsList, nil)
		a := newTestAccounts(mgr, nil, false)
		err := a.List(ctx, ListOptions{Columns: []string{ColumnUser, ColumnEnabled}})
		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})
}

// ---------------------------------------------------------------------------
// GetInitAdminPassword
// ---------------------------------------------------------------------------

func TestGetInitAdminPassword(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("is_secure_error", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		secureErr := errors.New("is-secure failed")
		mgr.On("IsSecure", ctx, common.EverestAdminUser).Return(false, secureErr)
		a := newTestAccounts(mgr, nil, false)
		_, err := a.GetInitAdminPassword(ctx)
		require.ErrorIs(t, err, secureErr)
		mgr.AssertExpectations(t)
	})

	t.Run("already_secure", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("IsSecure", ctx, common.EverestAdminUser).Return(true, nil)
		a := newTestAccounts(mgr, nil, false)
		_, err := a.GetInitAdminPassword(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot retrieve admin password")
		mgr.AssertExpectations(t)
	})

	t.Run("get_error", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		getErr := errors.New("get failed")
		mgr.On("IsSecure", ctx, common.EverestAdminUser).Return(false, nil)
		mgr.On("Get", ctx, common.EverestAdminUser).Return(nil, getErr)
		a := newTestAccounts(mgr, nil, false)
		_, err := a.GetInitAdminPassword(ctx)
		require.ErrorIs(t, err, getErr)
		mgr.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mgr := &mockAccountManager{}
		mgr.On("IsSecure", ctx, common.EverestAdminUser).Return(false, nil)
		mgr.On("Get", ctx, common.EverestAdminUser).Return(&accounts.Account{PasswordHash: "hash123"}, nil)
		a := newTestAccounts(mgr, nil, false)
		hash, err := a.GetInitAdminPassword(ctx)
		require.NoError(t, err)
		require.Equal(t, "hash123", hash)
		mgr.AssertExpectations(t)
	})
}

// ---------------------------------------------------------------------------
// CreateRSAKeyPair
// ---------------------------------------------------------------------------

func TestCreateRSAKeyPair(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("kube_error_calls_exit", func(t *testing.T) {
		t.Parallel()
		// Swap osExit so the test process doesn't actually terminate.
		origExit := osExit
		var gotCode int
		osExit = func(code int) { gotCode = code }
		t.Cleanup(func() { osExit = origExit })

		kube := &mockRSACreator{err: errors.New("rsa error")}
		a := newTestAccounts(nil, kube, false)
		_ = a.CreateRSAKeyPair(ctx)
		require.Equal(t, 1, gotCode)
	})

	t.Run("success_not_pretty", func(t *testing.T) {
		t.Parallel()
		kube := &mockRSACreator{err: nil}
		a := newTestAccounts(nil, kube, false)
		err := a.CreateRSAKeyPair(ctx)
		require.NoError(t, err)
	})

	t.Run("success_pretty", func(t *testing.T) {
		t.Parallel()
		// Redirect stdout to discard the pretty output.
		old := os.Stdout
		os.Stdout, _ = os.Open(os.DevNull)
		t.Cleanup(func() { os.Stdout = old })

		kube := &mockRSACreator{err: nil}
		a := newTestAccounts(nil, kube, true)
		err := a.CreateRSAKeyPair(ctx)
		require.NoError(t, err)
	})
}
