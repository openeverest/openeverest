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

// Package session provides a session manager for creating and verifying JWT tokens.
package session

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/api"
	"github.com/openeverest/openeverest/v2/pkg/accounts"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	// SessionManagerClaimsIssuer fills the "iss" field of the token.
	SessionManagerClaimsIssuer = "everest"
)

var (
	errExtractSub       = errors.New("could not extract sub")
	errExtractIss       = errors.New("could not extract iss")
	errExtractIssueTime = errors.New("could not extract issue time")
)

// Manager provides functionality for creating and managing JWT tokens.
type Manager struct {
	accountManager accounts.Interface
	signingKey     *rsa.PrivateKey
	Blocklist
	l *zap.SugaredLogger
}

// Option is a function that modifies a SessionManager.
type Option func(*Manager)

func (mgr *Manager) IsBlocked(ctx context.Context, token *jwt.Token) (bool, error) {
	username, isBuiltInUser, err := extractUsername(token)
	if err != nil {
		return false, fmt.Errorf("failed to extract username: %w", err)
	}
	// for the built-in users check if the account exists
	if isBuiltInUser {
		user, err := mgr.accountManager.Get(ctx, username)
		if err != nil {
			if errors.Is(err, accounts.ErrAccountNotFound) {
				return true, nil
			}
			return false, err
		}
		// checking the time when the password was last updated
		if user.PasswordMtime != "" {
			passwordCreationTime, err := time.Parse(time.RFC3339, user.PasswordMtime)
			if err != nil {
				return false, err
			}
			tokenIssueTime, err := extractCreationTime(token)
			if err != nil {
				return false, err
			}
			// if the token was issued before the password was last updated for the user - consider the token as invalid.
			if tokenIssueTime.Before(passwordCreationTime) {
				return true, nil
			}
		}
	}
	return mgr.Blocklist.IsBlocked(ctx, token)
}

// New creates a new session manager with the given options.
func New(ctx context.Context, l *zap.SugaredLogger, options ...Option) (*Manager, error) {
	m := &Manager{}
	for _, opt := range options {
		opt(m)
	}
	privKey, err := getPrivateKey()
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to get private key"))
	}
	m.signingKey = privKey
	m.l = l

	blockList, err := NewBlocklist(ctx, l)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to configure tokens blocklist"))
	}

	m.Blocklist = blockList
	return m, nil
}

// WithAccountManager sets the account manager to use for verifying user credentials.
func WithAccountManager(i accounts.Interface) Option {
	return func(m *Manager) {
		m.accountManager = i
	}
}

// Create creates a new token for a given subject (user) and returns it as a string.
// Passing a value of `0` for secondsBeforeExpiry creates a token that never expires.
// The id parameter holds an optional unique JWT token identifier and stored as a standard claim "jti" in the JWT token.
func (mgr *Manager) Create(subject string, secondsBeforeExpiry int64, id string) (string, error) {
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    SessionManagerClaimsIssuer,
		NotBefore: jwt.NewNumericDate(now),
		Subject:   subject,
		ID:        id,
	}
	if secondsBeforeExpiry > 0 {
		expires := now.Add(time.Duration(secondsBeforeExpiry) * time.Second)
		claims.ExpiresAt = jwt.NewNumericDate(expires)
	}

	return mgr.signClaims(claims)
}

func (mgr *Manager) signClaims(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(mgr.signingKey)
}

// Authenticate verifies the given username and password.
func (mgr *Manager) Authenticate(ctx context.Context, username string, password string) error {
	if password == "" {
		return fmt.Errorf("blank passwords are not allowed")
	}

	if err := mgr.accountManager.Verify(ctx, username, password); err != nil {
		return err
	}

	account, err := mgr.accountManager.Get(ctx, username)
	if err != nil {
		return err
	}

	if !account.Enabled {
		return accounts.ErrAccountDisabled
	}

	if !account.HasCapability(accounts.AccountCapabilityLogin) {
		return errors.Join(accounts.ErrInsufficientCapabilities, errors.New("user does not have capability to login"))
	}
	return nil
}

func getPrivateKey() (*rsa.PrivateKey, error) {
	pemString, err := os.ReadFile(common.EverestJWTPrivateKeyFile)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to read JWT private key"))
	}
	block, _ := pem.Decode(pemString)
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// KeyFunc retruns a function for getting the public RSA keys used
// for verifying the JWT tokens signed by everest.
func (mgr *Manager) KeyFunc() jwt.Keyfunc {
	return func(_ *jwt.Token) (interface{}, error) {
		return mgr.signingKey.Public(), nil
	}
}

func (mgr *Manager) BlocklistMiddleWare(skipperFunc func() (echomiddleware.Skipper, error)) (echo.MiddlewareFunc, error) {
	skipper, err := skipperFunc()
	if err != nil {
		return nil, err
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if skipper(c) {
				return next(c)
			}
			ctx := c.Request().Context()
			token, tErr := common.ExtractToken(ctx)
			if tErr != nil {
				return tErr
			}
			if isBlocked, err := mgr.IsBlocked(ctx, token); err != nil {
				mgr.l.Error(err)
				return c.JSON(http.StatusInternalServerError, api.Error{
					Message: pointer.ToString("Internal error"),
				})
			} else if isBlocked {
				return c.JSON(http.StatusUnauthorized, api.Error{
					Message: pointer.ToString("Invalid token"),
				})
			}
			return next(c)
		}
	}, nil
}

// extractUsername returns
// - the current username from the token
// - a bool flag that indicates whereas it's a built-in auth user
// - an error
func extractUsername(token *jwt.Token) (string, bool, error) {
	content, err := extractContent(token)
	if err != nil {
		return "", false, err
	}
	sub, ok := content.Payload["sub"].(string)
	if !ok {
		return "", false, errExtractSub
	}
	iss, ok := content.Payload["iss"].(string)
	if !ok {
		return "", false, errExtractIss
	}
	username := strings.TrimSuffix(sub, ":login")
	return username, iss == SessionManagerClaimsIssuer, nil
}

func extractCreationTime(token *jwt.Token) (*time.Time, error) {
	content, err := extractContent(token)
	if err != nil {
		return nil, err
	}
	issTS, ok := content.Payload["iat"].(float64)
	if !ok {
		return nil, errExtractIssueTime
	}
	parsedTime := time.Unix(int64(issTS), 0).UTC()
	return &parsedTime, nil
}

// ClientCacheOptions returns the cache options for the session manager k8s client.
// To avoid overwhelming k8s API with requests, the client should cache the accounts secret,
// because every authenticated API request checks the secret.
// It also defines a rule for the system namespace which gets requested otherwise the ByObject won't allow to read the ns.
func ClientCacheOptions() *cache.Options {
	return &cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&corev1.Secret{}: {
				Field: fields.SelectorFromSet(fields.Set{"metadata.name": common.EverestAccountsSecretName}),
			},
			&corev1.Namespace{}: {
				Field: fields.SelectorFromSet(fields.Set{"metadata.name": common.SystemNamespace}),
			},
		},
	}
}
