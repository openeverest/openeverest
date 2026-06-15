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

package tokenregistry

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

const testNamespace = "test-ns"

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	mockClient := fakeclient.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithObjects(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}).
		Build()
	k := kubernetes.NewEmpty(zap.NewNop().Sugar(), testNamespace).WithKubernetesClient(mockClient)

	r, err := New(context.Background(), zap.NewNop().Sugar(), k, testNamespace)
	require.NoError(t, err)
	return r
}

func TestRegistry_InitCreatesSecret(t *testing.T) {
	t.Parallel()
	r := newTestRegistry(t)

	secret, err := r.getSecret(context.Background())
	require.NoError(t, err)
	assert.Equal(t, common.EverestAuthTokensSecretName, secret.GetName())
}

func TestRegistry_MintAndValidate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := newTestRegistry(t)

	plaintext, rec, err := r.Mint(ctx, "alice", TypeRefresh, time.Hour)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(plaintext, "everest_rt_"))
	assert.Equal(t, "alice", rec.OwnerSubject)
	assert.Equal(t, TypeRefresh, rec.Type)
	require.NotNil(t, rec.ExpiresAt)
	assert.NotContains(t, rec.Hash, plaintext)

	got, err := r.Validate(ctx, plaintext)
	require.NoError(t, err)
	assert.Equal(t, rec.ID, got.ID)
	assert.Equal(t, "alice", got.OwnerSubject)
}

func TestRegistry_MintNonExpiring(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := newTestRegistry(t)

	plaintext, rec, err := r.Mint(ctx, "alice", TypeRefresh, 0)
	require.NoError(t, err)
	assert.Nil(t, rec.ExpiresAt)

	_, err = r.Validate(ctx, plaintext)
	require.NoError(t, err)
}

func TestRegistry_ValidateFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := newTestRegistry(t)

	plaintext, rec, err := r.Mint(ctx, "alice", TypeRefresh, time.Hour)
	require.NoError(t, err)

	t.Run("malformed token", func(t *testing.T) {
		_, err := r.Validate(ctx, "not-a-token")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("missing record", func(t *testing.T) {
		_, err := r.Validate(ctx, "everest_rt_0123456789abcdef0123456789abcdef_secret")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("tampered secret part", func(t *testing.T) {
		_, err := r.Validate(ctx, plaintext+"x")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("wrong type prefix", func(t *testing.T) {
		tampered := strings.Replace(plaintext, "everest_rt_", "everest_pat_", 1)
		_, err := r.Validate(ctx, tampered)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("expired token", func(t *testing.T) {
		r.now = func() time.Time { return rec.ExpiresAt.Add(time.Second) }
		defer func() { r.now = time.Now }()
		_, err := r.Validate(ctx, plaintext)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
}

func TestRegistry_Rotate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := newTestRegistry(t)

	oldPlaintext, oldRec, err := r.Mint(ctx, "alice", TypeRefresh, time.Hour)
	require.NoError(t, err)

	newPlaintext, newRec, err := r.Rotate(ctx, oldRec, time.Hour)
	require.NoError(t, err)
	assert.NotEqual(t, oldPlaintext, newPlaintext)
	assert.NotEqual(t, oldRec.ID, newRec.ID)
	assert.Equal(t, "alice", newRec.OwnerSubject)
	assert.Equal(t, TypeRefresh, newRec.Type)

	// New token is valid.
	_, err = r.Validate(ctx, newPlaintext)
	require.NoError(t, err)

	// Old token remains valid within the grace period...
	got, err := r.Validate(ctx, oldPlaintext)
	require.NoError(t, err)
	require.NotNil(t, got.ExpiresAt)
	assert.WithinDuration(t, time.Now().UTC().Add(RotationGracePeriod), *got.ExpiresAt, 5*time.Second)

	// ...but not after it.
	r.now = func() time.Time { return time.Now().UTC().Add(RotationGracePeriod + time.Second) }
	defer func() { r.now = time.Now }()
	_, err = r.Validate(ctx, oldPlaintext)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestRegistry_RotateDoesNotExtendExpiry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := newTestRegistry(t)

	// Token that expires before the grace period would.
	plaintext, rec, err := r.Mint(ctx, "alice", TypeRefresh, 10*time.Second)
	require.NoError(t, err)
	originalExpiry := *rec.ExpiresAt

	_, _, err = r.Rotate(ctx, rec, time.Hour)
	require.NoError(t, err)

	got, err := r.Validate(ctx, plaintext)
	require.NoError(t, err)
	assert.Equal(t, originalExpiry, *got.ExpiresAt)
}

func TestRegistry_Revoke(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := newTestRegistry(t)

	plaintext, rec, err := r.Mint(ctx, "alice", TypeRefresh, time.Hour)
	require.NoError(t, err)

	require.NoError(t, r.Revoke(ctx, rec.ID))

	_, err = r.Validate(ctx, plaintext)
	assert.ErrorIs(t, err, ErrInvalidToken)

	// Revoking again is not an error.
	require.NoError(t, r.Revoke(ctx, rec.ID))
}

func TestRegistry_PruneExpired(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := newTestRegistry(t)

	_, expiredRec, err := r.Mint(ctx, "alice", TypeRefresh, time.Second)
	require.NoError(t, err)
	_, liveRec, err := r.Mint(ctx, "bob", TypeRefresh, time.Hour)
	require.NoError(t, err)
	_, foreverRec, err := r.Mint(ctx, "carol", TypeRefresh, 0)
	require.NoError(t, err)

	r.now = func() time.Time { return time.Now().UTC().Add(time.Minute) }
	defer func() { r.now = time.Now }()
	require.NoError(t, r.PruneExpired(ctx))

	secret, err := r.getSecret(ctx)
	require.NoError(t, err)
	assert.NotContains(t, secret.Data, expiredRec.ID)
	assert.Contains(t, secret.Data, liveRec.ID)
	assert.Contains(t, secret.Data, foreverRec.ID)
}

func TestRegistry_ConcurrentMints(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := newTestRegistry(t)

	const n = 20
	done := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			_, _, err := r.Mint(ctx, "alice", TypeRefresh, time.Hour)
			done <- err
		}()
	}
	for i := 0; i < n; i++ {
		require.NoError(t, <-done)
	}

	secret, err := r.client.GetSecret(ctx, types.NamespacedName{Namespace: testNamespace, Name: common.EverestAuthTokensSecretName})
	require.NoError(t, err)
	assert.Len(t, secret.Data, n)
}

func TestParsePlaintext(t *testing.T) {
	t.Parallel()

	t.Run("refresh token", func(t *testing.T) {
		plaintext, id, err := newPlaintext(TypeRefresh)
		require.NoError(t, err)
		gotType, gotID, err := parsePlaintext(plaintext)
		require.NoError(t, err)
		assert.Equal(t, TypeRefresh, gotType)
		assert.Equal(t, id, gotID)
	})

	t.Run("pat token", func(t *testing.T) {
		plaintext, id, err := newPlaintext(TypePAT)
		require.NoError(t, err)
		gotType, gotID, err := parsePlaintext(plaintext)
		require.NoError(t, err)
		assert.Equal(t, TypePAT, gotType)
		assert.Equal(t, id, gotID)
	})

	t.Run("invalid", func(t *testing.T) {
		for _, tok := range []string{"", "everest_rt_", "everest_rt_no-secret", "bogus_prefix_id_secret"} {
			_, _, err := parsePlaintext(tok)
			assert.ErrorIs(t, err, ErrInvalidToken, "token: %q", tok)
		}
	})
}
