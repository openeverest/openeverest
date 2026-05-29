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
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/session"
)

const (
	// SubjectPrefix is prepended to the plugin name to form the token subject.
	SubjectPrefix = "plugin:"

	// DefaultTTL is the default token lifetime (24 h).
	DefaultTTL = 24 * time.Hour

	// PermissionsClaim is the custom claim key that carries allowed permissions.
	PermissionsClaim = "permissions"
)

// Permission describes a single verb+resource pair the daemon may exercise.
type Permission struct {
	Verb     string `json:"verb"`
	Resource string `json:"resource"`
}

// Service mints and inspects plugin service tokens.
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
	return &Service{signingKey: key}, nil
}

// Mint creates a new service token for the named plugin.
func (s *Service) Mint(pluginName string, permissions []Permission, ttl time.Duration) (string, error) {
	if ttl == 0 {
		ttl = DefaultTTL
	}
	now := time.Now().UTC()

	claims := pluginClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    session.SessionManagerClaimsIssuer,
			Subject:   SubjectPrefix + pluginName,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Permissions: permissions,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.signingKey)
}

// IsPluginToken returns true if the token subject starts with "plugin:".
func IsPluginToken(token *jwt.Token) bool {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}
	sub, _ := claims.GetSubject()
	return strings.HasPrefix(sub, SubjectPrefix)
}

// IsSubjectPluginToken returns true if the subject string identifies a plugin daemon.
func IsSubjectPluginToken(subject string) bool {
	return strings.HasPrefix(subject, SubjectPrefix)
}

// PluginName extracts the plugin name from a plugin service token.
func PluginName(token *jwt.Token) (string, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims type")
	}
	sub, err := claims.GetSubject()
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(sub, SubjectPrefix) {
		return "", errors.New("not a plugin token")
	}
	return strings.TrimPrefix(sub, SubjectPrefix), nil
}

// pluginClaims extends registered claims with a permissions list.
type pluginClaims struct {
	jwt.RegisteredClaims
	Permissions []Permission `json:"permissions,omitempty"`
}
