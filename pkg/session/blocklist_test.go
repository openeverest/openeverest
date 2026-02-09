// everest
// Copyright (C) 2025 Percona LLC
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
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestShortenToken(t *testing.T) {
	type tcase struct {
		name           string
		claims         jwt.MapClaims
		shortenedToken string
		error          error
	}
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
				Payload: make(map[string]interface{}),
			},
			error: nil,
		},
		{
			name:  "valid with payload - jti claim",
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"jti": "9d1c1f98-a479-41e3-8939-c7cb3e049a", "exp": float64(331743679478)}),
			result: &JWTContent{
				Payload: map[string]interface{}{"exp": float64(331743679478), "jti": "9d1c1f98-a479-41e3-8939-c7cb3e049a"},
			},
			error: nil,
		},
		{
			// "uti" is used by Microsoft Entra ID instead of "jti" claim
			name:  "valid with payload - uti claim",
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uti": "9d1c1f98-a479-41e3-8939-c7cb3e049a", "exp": float64(331743679478)}),
			result: &JWTContent{
				Payload: map[string]interface{}{"exp": float64(331743679478), "uti": "9d1c1f98-a479-41e3-8939-c7cb3e049a"},
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
	type tcase struct {
		name    string
		token   *jwt.Token
		tokenID string
		error   error
	}

	tokenUnsupportedClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{})
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
			k := kubernetes.NewEmpty(l).WithKubernetesClient(mockClient.Build())

			// check there is no blocklist secret before the blocklist creation
			secret, err := k.GetSecret(ctx, ctrlclient.ObjectKey{
				Name:      common.EverestBlocklistSecretName,
				Namespace: common.SystemNamespace,
			})
			assert.True(t, k8serrors.IsNotFound(err))
			assert.Nil(t, secret)

			b, err := mockNewBlocklist(ctx, l, k)
			assert.NoError(t, err)

			// blocklist secret appears after the blocklist creation
			secret, err = k.GetSecret(ctx, ctrlclient.ObjectKey{
				Name:      common.EverestBlocklistSecretName,
				Namespace: common.SystemNamespace,
			})
			assert.NoError(t, err)
			assert.NotNil(t, secret)
			assert.Equal(t, "", secret.StringData[dataKey])

			// block the token from the context and check the secret has been changed accordingly
			err = b.Block(ctx, tc.token)
			if tc.error != nil {
				assert.Equal(t, tc.error, err)
				return
			}

			assert.NoError(t, err)

			secret, err = k.GetSecret(ctx, ctrlclient.ObjectKey{
				Name:      common.EverestBlocklistSecretName,
				Namespace: common.SystemNamespace,
			})
			assert.NoError(t, err)
			// the mocked client does not do this StringData -> Data transformation in Secrets which the actual k8a API do, so
			// we only check the StringData field
			assert.Equal(t, tc.tokenID, secret.StringData[dataKey])

			// deleting secret to test the backoff
			err = k.DeleteSecret(ctx, secret)
			assert.NoError(t, err)

			// after deleting secret - try to block again, get the NotFound error
			err = b.Block(ctx, tc.token)
			assert.Equal(t, true, k8serrors.IsNotFound(err))
		})
	}
}

func TestBlocklist_IsBlocked(t *testing.T) {
	type tcase struct {
		name    string
		objs    []ctrlclient.Object
		token   *jwt.Token
		tokenID string
		error   error
		blocked bool
	}

	tokenUnsupportedClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{})
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
						Namespace: common.SystemNamespace,
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
						Namespace: common.SystemNamespace,
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
						Namespace: common.SystemNamespace,
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
						Namespace: common.SystemNamespace,
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
			k := kubernetes.NewEmpty(l).WithKubernetesClient(mockClient.Build())

			b, err := mockNewBlocklist(ctx, l, k)
			assert.NoError(t, err)

			blocked, err := b.IsBlocked(ctx, tc.token)
			if tc.error != nil {
				assert.Equal(t, tc.error, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.blocked, blocked)
		})
	}
}

func mockNewBlocklist(ctx context.Context, logger *zap.SugaredLogger, mockClient TokenStoreClient) (Blocklist, error) {
	store, err := newTokenStore(ctx, mockClient, logger)
	if err != nil {
		return nil, err
	}
	return &blocklist{
		tokenStore: store,
		l:          logger,
	}, nil
}
