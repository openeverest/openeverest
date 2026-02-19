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
	"cmp"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

const (
	dataKey         = "list"
	maxRetries      = 10
	backoffInterval = 500 * time.Millisecond
)

var (
	errExtractTokenID   = errors.New("token doesn't contains jti or uti claim")
	errExtractExp       = errors.New("could not extract exp")
	errEmptyToken       = errors.New("token is empty")
	errUnsupportedClaim = func(claims any) error {
		return errors.New(fmt.Sprintf("unsupported claims type: %T", claims))
	}
	errShortenToken = errors.New("failed to shorten token")
)

// Blocklist represents interface to block JWT tokens and check if a token is blocked.
type Blocklist interface {
	// Block invalidates the token from the context by adding it to blocklist.
	Block(ctx context.Context, token *jwt.Token) error
	// IsBlocked checks if the token from the context is blocked.
	IsBlocked(ctx context.Context, token *jwt.Token) (bool, error)
}

type blocklist struct {
	tokenStore TokenStore
	l          *zap.SugaredLogger
}

// TokenStore represents an abstraction for storage, hiding details about how the data is actually stored.
type TokenStore interface {
	// Add adds the shortened token to the blocklist
	Add(ctx context.Context, shortenedToken string) error
	// Exists checks if the shortened token is in the blocklist
	Exists(ctx context.Context, shortenedToken string) (bool, error)
}

// NewBlocklist creates a new block list
func NewBlocklist(ctx context.Context, logger *zap.SugaredLogger) (Blocklist, error) {
	options := &cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&corev1.Secret{}: {
				Field: fields.SelectorFromSet(fields.Set{"metadata.name": common.EverestBlocklistSecretName}),
			},
		},
	}
	// A separate client is needed to apply the controller-runtime cache only to the related objects.
	// Using the controller-runtime client is also beneficial because it supports HA mode.
	tokenStoreClient, err := kubernetes.NewInCluster(logger, ctx, options)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed creating Kubernetes client for blockList"))
	}

	store, err := newTokenStore(ctx, tokenStoreClient, logger)
	if err != nil {
		return nil, err
	}
	return &blocklist{
		tokenStore: store,
		l:          logger,
	}, nil
}

// Block invalidates the token from the context by adding it to blocklist.
func (b *blocklist) Block(ctx context.Context, token *jwt.Token) error {
	shortenedToken, err := shortenToken(token)
	if err != nil {
		return err
	}

	var bOff backoff.BackOff
	bOff = backoff.NewConstantBackOff(backoffInterval)
	bOff = backoff.WithMaxRetries(bOff, maxRetries)
	bOff = backoff.WithContext(bOff, ctx)
	return backoff.Retry(
		func() error {
			return b.tokenStore.Add(ctx, shortenedToken)
		},
		bOff,
	)
}

// IsBlocked checks if the token from the context is blocked.
func (b *blocklist) IsBlocked(ctx context.Context, token *jwt.Token) (bool, error) {
	shortenedToken, err := shortenToken(token)
	if err != nil {
		return false, errors.Join(err, errShortenToken)
	}

	return b.tokenStore.Exists(ctx, shortenedToken)
}

// shortenToken contains only the "jti" and the "exp" claims from the token, so the format of shortened token is
// <jti><expiration_timestamp>, for example "9d1c1f98-a479-41e3-8939-c7cb3edefa331743679478",
// where last 10 digits represent the expiration timestamp.
func shortenToken(token *jwt.Token) (string, error) {
	content, err := extractContent(token)
	if err != nil {
		return "", err
	}
	tId := cmp.Or(
		content.getStringClaim("jti"),
		content.getStringClaim("uti"), // "uti" is used by Microsoft Entra ID instead of "jti" claim
	)
	if tId == "" {
		return "", errExtractTokenID
	}

	exp, ok := content.Payload["exp"].(float64)
	if !ok {
		return "", errExtractExp
	}
	return tId + strconv.FormatFloat(exp, 'f', 0, 64), nil
}

// JWTContent represents the JWT token structure that is used by blocklist.
type JWTContent struct {
	Payload map[string]interface{} `json:"payload"`
}

func (j JWTContent) getStringClaim(key string) string {
	value, ok := j.Payload[key]
	if !ok {
		return ""
	}
	return value.(string)
}

func extractContent(token *jwt.Token) (*JWTContent, error) {
	if token == nil {
		return nil, errEmptyToken
	}
	claimsMap := make(map[string]interface{})

	switch claims := token.Claims.(type) {
	case jwt.MapClaims:
		for key, val := range claims {
			claimsMap[key] = val
		}
	default:
		return nil, errUnsupportedClaim(claims)
	}

	return &JWTContent{
		Payload: claimsMap,
	}, nil
}
