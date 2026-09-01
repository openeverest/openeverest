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

package session

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestShortenToken(t *testing.T) {
	t.Parallel()
	type tcase struct {
		name           string
		claims         jwt.MapClaims
		shortenedToken string
		error          error
	}
	//nolint:gosec // test JWT token data, not credentials
	tcases := []tcase{
		{
			name: "valid jti",
			claims: jwt.MapClaims{
				"jti": "9d1c1f98-a479-41e3-8939-c7cb3edefa",
				"exp": float64(331743679478),
			},
			shortenedToken: "9d1c1f98-a479-41e3-8939-c7cb3edefa331743679478",
			error:          nil,
		},
		{
			// "uti" is used by Microsoft Entra ID instead of "jti" claim
			name: "valid uti",
			claims: jwt.MapClaims{
				"uti": "9d1c1f98-a479-41e3-8939-c7cb3edefa",
				"exp": float64(331743679478),
			},
			shortenedToken: "9d1c1f98-a479-41e3-8939-c7cb3edefa331743679478",
			error:          nil,
		},
		{
			name: "no jti or uti",
			claims: jwt.MapClaims{
				"exp": float64(331743679478),
			},
			shortenedToken: "",
			error:          errExtractTokenID,
		},
		{
			name: "no exp",
			claims: jwt.MapClaims{
				"jti": "9d1c1f98-a479-41e3-8939-c7cb3e049a",
			},
			shortenedToken: "",
			error:          errExtractExp,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := shortenToken(jwt.NewWithClaims(jwt.SigningMethodHS256, tc.claims))
			assert.Equal(t, tc.error, err)
			assert.Equal(t, tc.shortenedToken, result)
		})
	}
}

func TestExtractContent(t *testing.T) {
	t.Parallel()
	type tcase struct {
		name   string
		token  *jwt.Token
		error  error
		result *JWTContent
	}
	tokenUnsupportedClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{})
	tcases := []tcase{
		{
			name:   "empty token",
			token:  nil,
			result: nil,
			error:  errEmptyToken,
		},
		{
			name:   "unsupported claims",
			token:  tokenUnsupportedClaims,
			result: nil,
			error:  errUnsupportedClaim(tokenUnsupportedClaims.Claims),
		},
		{
			name:  "valid empty payload",
			token: jwt.New(jwt.SigningMethodHS256),
			result: &JWTContent{
				Payload: make(map[string]any),
			},
			error: nil,
		},
		{
			name:  "valid with payload - jti claim",
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"jti": "9d1c1f98-a479-41e3-8939-c7cb3e049a", "exp": float64(331743679478)}),
			result: &JWTContent{
				Payload: map[string]any{"exp": float64(331743679478), "jti": "9d1c1f98-a479-41e3-8939-c7cb3e049a"},
			},
			error: nil,
		},
		{
			// "uti" is used by Microsoft Entra ID instead of "jti" claim
			name:  "valid with payload - uti claim",
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uti": "9d1c1f98-a479-41e3-8939-c7cb3e049a", "exp": float64(331743679478)}),
			result: &JWTContent{
				Payload: map[string]any{"exp": float64(331743679478), "uti": "9d1c1f98-a479-41e3-8939-c7cb3e049a"},
			},
			error: nil,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := extractContent(tc.token)
			assert.Equal(t, tc.error, err)
			assert.Equal(t, tc.result, result)
		})
	}
}

func TestBlocklist_Block(t *testing.T) {
	t.Parallel()
	type tcase struct {
		name    string
		token   *jwt.Token
		tokenID string
		error   error
	}

	tokenUnsupportedClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{})
	//nolint:gosec // test JWT token data, not credentials
	tcases := []tcase{
		{
			name:    "empty token",
			token:   nil,
			tokenID: "",
			error:   errEmptyToken,
		},
		{
			name:    "unsupported claims",
			token:   tokenUnsupportedClaims,
			tokenID: "",
			error:   errUnsupportedClaim(tokenUnsupportedClaims.Claims),
		},
		{
			name:    "valid empty payload",
			token:   jwt.New(jwt.SigningMethodHS256),
			tokenID: "",
			error:   errExtractTokenID,
		},
		{
			name:    "valid with payload - jti claim",
			token:   jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"jti": "9d1c1f98-a479-41e3-8939-c7cb3e049a", "exp": float64(331743679478)}),
			tokenID: "9d1c1f98-a479-41e3-8939-c7cb3e049a331743679478",
			error:   nil,
		},
		{
			// "uti" is used by Microsoft Entra ID instead of "jti" claim
			name:    "valid with payload - uti claim",
			token:   jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uti": "keQ6r7TMsEikaoCHni2bAA", "exp": float64(331743679478)}),
			tokenID: "keQ6r7TMsEikaoCHni2bAA331743679478",
			error:   nil,
		},
	}

	l := zap.NewNop().Sugar()
	ctx := context.Background()

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockClient := fakeclient.NewClientBuilder().WithScheme(kubernetes.CreateScheme())
			k := kubernetes.NewEmpty(l, "test-ns").WithKubernetesClient(mockClient.Build())

			// check there is no blocklist secret before the blocklist creation
			secret, err := k.GetSecret(ctx, ctrlclient.ObjectKey{
				Name:      common.EverestBlocklistSecretName,
				Namespace: "test-ns",
			})
			assert.True(t, k8serrors.IsNotFound(err))
			assert.Nil(t, secret)

			b, err := mockNewBlocklist(ctx, l, k)
			require.NoError(t, err)

			// blocklist secret appears after the blocklist creation
			secret, err = k.GetSecret(ctx, ctrlclient.ObjectKey{
				Name:      common.EverestBlocklistSecretName,
				Namespace: "test-ns",
			})
			require.NoError(t, err)
			assert.NotNil(t, secret)
			assert.Empty(t, secret.StringData[dataKey])

			// block the token from the context and check the secret has been changed accordingly
			err = b.Block(ctx, tc.token)
			if tc.error != nil {
				assert.Equal(t, tc.error, err)
				return
			}

			require.NoError(t, err)

			secret, err = k.GetSecret(ctx, ctrlclient.ObjectKey{
				Name:      common.EverestBlocklistSecretName,
				Namespace: "test-ns",
			})
			require.NoError(t, err)
			// the mocked client does not do this StringData -> Data transformation in Secrets which the actual k8a API do, so
			// we only check the StringData field
			assert.Equal(t, tc.tokenID, secret.StringData[dataKey])

			// deleting secret to test the backoff
			err = k.DeleteSecret(ctx, secret)
			require.NoError(t, err)

			// after deleting secret - try to block again, get the NotFound error
			err = b.Block(ctx, tc.token)
			assert.True(t, k8serrors.IsNotFound(err))
		})
	}
}

func TestBlocklist_IsBlocked(t *testing.T) {
	t.Parallel()
	type tcase struct {
		name    string
		objs    []ctrlclient.Object
		token   *jwt.Token
		tokenID string
		error   error
		blocked bool
	}

	tokenUnsupportedClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{})
	//nolint:gosec // test JWT token data, not credentials
	tcases := []tcase{
		{
			name:    "empty token",
			token:   nil,
			objs:    []ctrlclient.Object{},
			tokenID: "",
			error:   errors.Join(errEmptyToken, errShortenToken),
			blocked: false,
		},
		{
			name:    "unsupported claims",
			token:   tokenUnsupportedClaims,
			objs:    []ctrlclient.Object{},
			tokenID: "",
			error:   errors.Join(errUnsupportedClaim(tokenUnsupportedClaims.Claims), errShortenToken),
			blocked: false,
		},
		{
			name:    "valid empty payload",
			token:   jwt.New(jwt.SigningMethodHS256),
			objs:    []ctrlclient.Object{},
			tokenID: "",
			error:   errors.Join(errExtractTokenID, errShortenToken),
			blocked: false,
		},
		{
			name:    "empty list - not blocked valid with payload - jti claim",
			objs:    []ctrlclient.Object{},
			token:   jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"jti": "9d1c1f98-a479-41e3-8939-c7cb3e049a", "exp": float64(331743679478)}),
			tokenID: "9d1c1f98-a479-41e3-8939-c7cb3e049a331743679478",
			error:   nil,
			blocked: false,
		},
		{
			name: "non-empty list - not blocked valid with payload - jti claim",
			objs: []ctrlclient.Object{
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      common.EverestBlocklistSecretName,
						Namespace: "test-ns",
					},
					StringData: map[string]string{
						dataKey: "some-another-jti331743679478",
					},
					Data: map[string][]byte{
						dataKey: []byte("some-another-jti331743679478"),
					},
				},
			},
			token:   jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"jti": "9d1c1f98-a479-41e3-8939-c7cb3e049a", "exp": float64(331743679478)}),
			tokenID: "9d1c1f98-a479-41e3-8939-c7cb3e049a331743679478",
			error:   nil,
			blocked: false,
		},
		{
			name: "blocked valid with payload - jti claim",
			objs: []ctrlclient.Object{
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      common.EverestBlocklistSecretName,
						Namespace: "test-ns",
					},
					StringData: map[string]string{
						dataKey: "9d1c1f98-a479-41e3-8939-c7cb3e049a331743679478",
					},
					Data: map[string][]byte{
						dataKey: []byte("9d1c1f98-a479-41e3-8939-c7cb3e049a331743679478"),
					},
				},
			},
			token:   jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"jti": "9d1c1f98-a479-41e3-8939-c7cb3e049a", "exp": float64(331743679478)}),
			tokenID: "9d1c1f98-a479-41e3-8939-c7cb3e049a331743679478",
			error:   nil,
			blocked: true,
		},
		{
			// "uti" is used by Microsoft Entra ID instead of "jti" claim
			name:    "empty list - not blocked valid with payload - uti claim",
			objs:    []ctrlclient.Object{},
			token:   jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uti": "keQ6r7TMsEikaoCHni2bAA", "exp": float64(331743679478)}),
			tokenID: "keQ6r7TMsEikaoCHni2bAA331743679478",
			error:   nil,
			blocked: false,
		},
		{
			// "uti" is used by Microsoft Entra ID instead of "jti" claim
			name: "non-empty list - not blocked valid with payload - uti claim",
			objs: []ctrlclient.Object{
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      common.EverestBlocklistSecretName,
						Namespace: "test-ns",
					},
					StringData: map[string]string{
						dataKey: "some-another-jti331743679478",
					},
					Data: map[string][]byte{
						dataKey: []byte("some-another-jti331743679478"),
					},
				},
			},
			token:   jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uti": "keQ6r7TMsEikaoCHni2bAA", "exp": float64(331743679478)}),
			tokenID: "keQ6r7TMsEikaoCHni2bAA331743679478",
			error:   nil,
			blocked: false,
		},
		{
			// "uti" is used by Microsoft Entra ID instead of "jti" claim
			name: "blocked valid with payload - uti claim",
			objs: []ctrlclient.Object{
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      common.EverestBlocklistSecretName,
						Namespace: "test-ns",
					},
					StringData: map[string]string{
						dataKey: "keQ6r7TMsEikaoCHni2bAA331743679478",
					},
					Data: map[string][]byte{
						dataKey: []byte("keQ6r7TMsEikaoCHni2bAA331743679478"),
					},
				},
			},
			token:   jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uti": "keQ6r7TMsEikaoCHni2bAA", "exp": float64(331743679478)}),
			tokenID: "keQ6r7TMsEikaoCHni2bAA331743679478",
			error:   nil,
			blocked: true,
		},
	}

	l := zap.NewNop().Sugar()
	ctx := context.Background()

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockClient := fakeclient.NewClientBuilder().WithScheme(kubernetes.CreateScheme())
			mockClient.WithObjects(tc.objs...)
			k := kubernetes.NewEmpty(l, "test-ns").WithKubernetesClient(mockClient.Build())

			b, err := mockNewBlocklist(ctx, l, k)
			require.NoError(t, err)

			blocked, err := b.IsBlocked(ctx, tc.token)
			if tc.error != nil {
				assert.Equal(t, tc.error, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.blocked, blocked)
		})
	}
}

// TestTokenStore_Add_UpdateFailure asserts the invariant that Add reports
// success only when the token is genuinely blocklisted. When the underlying
// Secret update fails, Add must return the error rather than swallow it, and
// the token must not be persisted to the blocklist Secret.
//
// This is a regression test for the missing backport of #2086 (see #2766): on
// the unfixed code Add returns the nil `err` from the earlier GetSecret instead
// of the update error, so a failed write is reported as success and the token
// is never blocklisted.
func TestTokenStore_Add_UpdateFailure(t *testing.T) {
	t.Parallel()

	l := zap.NewNop().Sugar()
	ctx := context.Background()

	//nolint:gosec // test token ID, not a credential
	const shortenedToken = "9d1c1f98-a479-41e3-8939-c7cb3e049a331743679478"

	// Fail every Secret update to simulate a transient write error (e.g. a 409
	// conflict from concurrent logouts). Reads and the init-time create still
	// succeed, so Add reaches the update and fails there.
	mockClient := fakeclient.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(
				ctx context.Context,
				c ctrlclient.WithWatch,
				obj ctrlclient.Object,
				opts ...ctrlclient.UpdateOption,
			) error {
				if _, ok := obj.(*corev1.Secret); ok {
					return k8serrors.NewConflict(
						schema.GroupResource{Resource: "secrets"},
						obj.GetName(),
						errors.New("simulated conflict"),
					)
				}
				return c.Update(ctx, obj, opts...)
			},
		})
	k := kubernetes.NewEmpty(l, "test-ns").WithKubernetesClient(mockClient.Build())

	store, err := newTokenStore(ctx, k, l, "test-ns")
	require.NoError(t, err)

	// The Secret write fails, so Add must surface the error and not report
	// success. The defect returned the wrong error variable, so pin that the
	// surfaced error is the UpdateSecret conflict rather than merely "an error":
	// kubernetes.UpdateSecret returns the client error unwrapped.
	err = store.Add(ctx, shortenedToken)
	require.Error(t, err)
	assert.True(t, k8serrors.IsConflict(err), "Add must surface the UpdateSecret error, got %v", err)

	// Read-back invariant: because the write failed, the token must not have
	// reached the Secret. As in TestBlocklist_Block, the fake client does not do
	// the StringData -> Data transformation the real API server does, so the
	// assertion has to be on StringData.
	secret, err := k.GetSecret(ctx, ctrlclient.ObjectKey{
		Name:      common.EverestBlocklistSecretName,
		Namespace: "test-ns",
	})
	require.NoError(t, err)
	assert.NotContains(t, secret.StringData[dataKey], shortenedToken)
}

func mockNewBlocklist(ctx context.Context, logger *zap.SugaredLogger, mockClient TokenStoreClient) (Blocklist, error) {
	store, err := newTokenStore(ctx, mockClient, logger, "test-ns")
	if err != nil {
		return nil, err
	}
	return &blocklist{
		tokenStore: store,
		l:          logger,
	}, nil
}
