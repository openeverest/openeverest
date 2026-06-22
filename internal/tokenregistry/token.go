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

// Package tokenregistry provides a registry for opaque, server-side API tokens
// (refresh tokens and, in a later phase, personal access tokens).
package tokenregistry

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
)

// Type is the type of an opaque API token.
type Type string

const (
	// TypeRefresh is a session refresh token.
	TypeRefresh Type = "refresh"
	// TypePAT is a personal access token (reserved for a later phase).
	TypePAT Type = "pat"

	refreshTokenPrefix = "everest_rt_"  //nolint:gosec
	patTokenPrefix     = "everest_pat_" //nolint:gosec

	// secretEntropyBytes is the amount of CSPRNG randomness in the secret part of a token (256 bits).
	secretEntropyBytes = 32
)

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// prefix returns the plaintext token prefix for the token type.
func (t Type) prefix() (string, error) {
	switch t {
	case TypeRefresh:
		return refreshTokenPrefix, nil
	case TypePAT:
		return patTokenPrefix, nil
	}
	return "", fmt.Errorf("unknown token type: %q", t)
}

// newPlaintext generates a new plaintext token of the given type along with its record ID.
// The format is <prefix><id>_<secret>, where the ID is embedded for O(1) registry lookup.
func newPlaintext(t Type) (plaintext, id string, err error) {
	prefix, err := t.prefix()
	if err != nil {
		return "", "", err
	}
	// uuid without dashes: a valid Kubernetes Secret data key and underscore-free.
	id = strings.ReplaceAll(uuid.New().String(), "-", "")

	buf := make([]byte, secretEntropyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("failed to generate token randomness: %w", err)
	}
	return prefix + id + "_" + encodeBase62(buf), id, nil
}

// parsePlaintext extracts the token type and record ID from a plaintext token.
func parsePlaintext(plaintext string) (t Type, id string, err error) {
	var rest string
	switch {
	case strings.HasPrefix(plaintext, refreshTokenPrefix):
		t, rest = TypeRefresh, strings.TrimPrefix(plaintext, refreshTokenPrefix)
	case strings.HasPrefix(plaintext, patTokenPrefix):
		t, rest = TypePAT, strings.TrimPrefix(plaintext, patTokenPrefix)
	default:
		return "", "", ErrInvalidToken
	}
	id, secret, found := strings.Cut(rest, "_")
	if !found || id == "" || secret == "" {
		return "", "", ErrInvalidToken
	}
	return t, id, nil
}

// hashPlaintext returns the at-rest representation of a plaintext token.
// A fast hash is acceptable because the secret part is high-entropy randomness,
// not a low-entropy password.
func hashPlaintext(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func encodeBase62(b []byte) string {
	num := (&big.Int{}).SetBytes(b)
	base := big.NewInt(int64(len(base62Alphabet)))
	rem := &big.Int{}
	out := make([]byte, 0, len(b)*2)
	for num.Sign() > 0 {
		num.DivMod(num, base, rem)
		out = append(out, base62Alphabet[rem.Int64()])
	}
	if len(out) == 0 {
		out = append(out, base62Alphabet[0])
	}
	// reverse
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}
