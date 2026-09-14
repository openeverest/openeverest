// everest
// Copyright (C) 2025 Percona LLC
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

package kubernetes

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/accounts"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

const testNamespace = "test-ns"

func TestAccounts(t *testing.T) {
	t.Parallel()

	objs := []ctrlclient.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.EverestAccountsSecretName,
				Namespace: "test-ns",
			},
		},
	}

	mockClient := fakeclient.NewClientBuilder().WithScheme(CreateScheme())
	mockClient.WithObjects(objs...)
	k := NewEmpty(zap.NewNop().Sugar(), "test-ns").WithKubernetesClient(mockClient.Build())
	accounts.Tests(t, k.Accounts())
}

// conflictClient wraps a ctrlclient.Client and injects a Conflict error
// on the first Update call, simulating a concurrent modification.
// On that first call it also mutates the underlying secret using the provided
// mutate function, so the retry must re-read fresh data.
type conflictClient struct {
	ctrlclient.Client

	attempt atomic.Int32
	mutate  func(*corev1.Secret)
}

func (c *conflictClient) Update(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.UpdateOption) error {
	if c.attempt.Add(1) == 1 {
		// Simulate a concurrent writer that adds a new user.
		sec := &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{
			Namespace: testNamespace,
			Name:      common.EverestAccountsSecretName,
		}, sec); err != nil {
			return err
		}

		c.mutate(sec)

		if err := c.Client.Update(ctx, sec); err != nil {
			return err
		}

		// Return a Conflict error so RetryOnConflict triggers a retry.
		return apierrors.NewConflict(
			schema.GroupResource{Resource: "secrets"},
			common.EverestAccountsSecretName,
			nil,
		)
	}

	return c.Client.Update(ctx, obj, opts...)
}

func TestAccounts_DeleteConflict(t *testing.T) {
	t.Parallel()

	objs := []ctrlclient.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:            common.EverestAccountsSecretName,
				Namespace:       testNamespace,
				ResourceVersion: "1",
			},
			Data: map[string][]byte{
				common.EverestAccountsFileName: []byte("admin:\n  enabled: true\n"),
			},
		},
	}

	mockClient := fakeclient.NewClientBuilder().
		WithScheme(CreateScheme()).
		WithObjects(objs...).
		Build()

	k := NewEmpty(zap.NewNop().Sugar(), testNamespace).WithKubernetesClient(&conflictClient{
		Client: mockClient,
		mutate: func(sec *corev1.Secret) {
			sec.Data[common.EverestAccountsFileName] = []byte("admin:\n  enabled: true\nconcurrent-user:\n  enabled: true\n")
		},
	})
	err := k.Accounts().Delete(context.Background(), "admin")
	require.NoError(t, err)

	accs, err := k.Accounts().List(context.Background())
	require.NoError(t, err)
	require.NotContains(t, accs, "admin")
	require.Contains(t, accs, "concurrent-user")
}

func TestAccounts_CreateConflict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mutateFn      func(sec *corev1.Secret)
		expectedError error
		expectedUsers []string
	}{
		{
			name: "concurrent writer adds an unrelated user",
			mutateFn: func(sec *corev1.Secret) {
				sec.Data[common.EverestAccountsFileName] = []byte("admin:\n  enabled: true\nunrelated-user:\n  enabled: true\n")
			},
			expectedError: nil,
			expectedUsers: []string{"newuser", "unrelated-user"},
		},
		{
			name: "concurrent writer adds the same user",
			mutateFn: func(sec *corev1.Secret) {
				sec.Data[common.EverestAccountsFileName] = []byte("admin:\n  enabled: true\nnewuser:\n  enabled: true\n")
			},
			expectedError: accounts.ErrUserAlreadyExists,
			expectedUsers: []string{"newuser"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			objs := []ctrlclient.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:            common.EverestAccountsSecretName,
						Namespace:       testNamespace,
						ResourceVersion: "1",
					},
					Data: map[string][]byte{
						common.EverestAccountsFileName: []byte("admin:\n  enabled: true\n"),
					},
				},
			}

			mockClient := fakeclient.NewClientBuilder().
				WithScheme(CreateScheme()).
				WithObjects(objs...).
				Build()

			k := NewEmpty(zap.NewNop().Sugar(), testNamespace).WithKubernetesClient(&conflictClient{
				Client: mockClient,
				mutate: tt.mutateFn,
			})
			err := k.Accounts().Create(context.Background(), "newuser", "password")

			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			accs, err := k.Accounts().List(context.Background())
			require.NoError(t, err)
			for _, u := range tt.expectedUsers {
				require.Contains(t, accs, u)
			}
		})
	}
}

func TestAccounts_Errors(t *testing.T) {
	t.Parallel()

	objs := []ctrlclient.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.EverestAccountsSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				common.EverestAccountsFileName: []byte("admin:\n  enabled: true\n"),
			},
		},
	}

	mockClient := fakeclient.NewClientBuilder().
		WithScheme(CreateScheme()).
		WithObjects(objs...).
		Build()

	k := NewEmpty(zap.NewNop().Sugar(), testNamespace).WithKubernetesClient(mockClient)

	err := k.Accounts().Create(context.Background(), "admin", "password")
	require.ErrorIs(t, err, accounts.ErrUserAlreadyExists)

	err = k.Accounts().SetPassword(context.Background(), "missing", "password", true)
	require.ErrorIs(t, err, accounts.ErrAccountNotFound)
}
