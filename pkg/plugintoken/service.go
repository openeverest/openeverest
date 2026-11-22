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

// Package plugintoken mints and validates autonomous service tokens for
// daemon-mode plugins. Tokens use the same RS256 signing key and issuer
// as user session tokens, so the standard JWT middleware accepts them.
package plugintoken

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/casbin/casbin/v2/util"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/session"
)

const (
	// SubjectPrefix is prepended to the plugin name to form the token subject.
	SubjectPrefix = "plugin:"

	// DefaultTTL is the default token lifetime (24 h).
	DefaultTTL = 24 * time.Hour

	// SecretKey is the key of the token in the plugin's token Secret.
	SecretKey = "token"
)

// Claims are the claims of a plugin service token.
type Claims struct {
	jwt.RegisteredClaims

	// PluginUID binds the token to one Plugin object, so a token outlives
	// neither its Plugin nor a Plugin re-created under the same name.
	PluginUID   string                                `json:"plugin_uid"` //nolint:tagliatelle // JWT claim names are snake_case, like oidc_issuer.
	Permissions []extensionsv1alpha1.PluginPermission `json:"permissions,omitempty"`
}

// Service mints plugin service tokens.
type Service struct {
	signingKey *rsa.PrivateKey
}

// NewService creates a token service using the Everest JWT private key.
func NewService() (*Service, error) {
	pemBytes, err := os.ReadFile(common.EverestJWTPrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("read JWT private key: %w", err)
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block from private key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return NewServiceWithKey(key), nil
}

// NewServiceWithKey creates a token service signing with key.
func NewServiceWithKey(key *rsa.PrivateKey) *Service {
	return &Service{signingKey: key}
}

// Mint creates a new service token for plugin, carrying the permissions it
// declares.
func (s *Service) Mint(plugin *extensionsv1alpha1.Plugin, now time.Time, ttl time.Duration) (string, error) {
	if ttl == 0 {
		ttl = DefaultTTL
	}
	now = now.UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    session.SessionManagerClaimsIssuer,
			Subject:   SubjectPrefix + plugin.GetName(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		PluginUID:   string(plugin.GetUID()),
		Permissions: plugin.Spec.Permissions,
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.signingKey)
}

// IsSubjectPluginToken returns true if the subject string identifies a plugin daemon.
func IsSubjectPluginToken(subject string) bool {
	return strings.HasPrefix(subject, SubjectPrefix)
}

// ClaimsOf returns the plugin claims of a parsed token.
func ClaimsOf(token *jwt.Token) (*Claims, error) {
	if c, ok := token.Claims.(*Claims); ok {
		return c, nil
	}
	raw, err := json.Marshal(token.Claims)
	if err != nil {
		return nil, fmt.Errorf("marshal claims: %w", err)
	}
	c := &Claims{}
	if err := json.Unmarshal(raw, c); err != nil {
		return nil, fmt.Errorf("unmarshal plugin claims: %w", err)
	}
	if !IsSubjectPluginToken(c.Subject) {
		return nil, errors.New("not a plugin token")
	}
	return c, nil
}

// PluginName returns the name of the plugin the token was issued to.
func (c *Claims) PluginName() string {
	return strings.TrimPrefix(c.Subject, SubjectPrefix)
}

// ValidFor reports whether a token with these claims is still valid for
// plugin: the plugin must be the one it was issued to, enabled, not being
// deleted, and still declare every permission the token carries. This is how
// tokens are revoked before they expire.
func (c *Claims) ValidFor(plugin *extensionsv1alpha1.Plugin) bool {
	if plugin == nil || string(plugin.GetUID()) != c.PluginUID || c.PluginName() != plugin.GetName() {
		return false
	}
	if !plugin.Spec.Enabled || !plugin.GetDeletionTimestamp().IsZero() {
		return false
	}
	for _, p := range c.Permissions {
		if !slices.Contains(plugin.Spec.Permissions, p) {
			return false
		}
	}
	return true
}

// Allows reports whether permissions grant action on resource. Verbs and
// resources are glob patterns, as in RBAC policies.
func Allows(permissions []extensionsv1alpha1.PluginPermission, resource, action string) bool {
	for _, p := range permissions {
		resOK, err := util.GlobMatch(resource, p.Resource)
		if err != nil || !resOK {
			continue
		}
		if actOK, err := util.GlobMatch(action, p.Verb); err == nil && actOK {
			return true
		}
	}
	return false
}
