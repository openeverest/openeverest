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

package common

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"

	"github.com/openeverest/openeverest/v2/pkg/accounts"
)

const (
	defaultPasswordLength = 32
)

func generateRandomPassword() (string, error) {
	b := make([]byte, defaultPasswordLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateInitialAdminAccount creates the initial admin account
// with a random plain-text password.
func CreateInitialAdminAccount(
	ctx context.Context,
	c accounts.Interface,
) error {
	pass, err := generateRandomPassword()
	if err != nil {
		return errors.Join(err, errors.New("could not generate random password"))
	}
	// Check if the admin account exists?
	if _, err := c.Get(ctx, EverestAdminUser); errors.Is(err, accounts.ErrAccountNotFound) {
		if createErr := c.Create(ctx, EverestAdminUser, pass); createErr != nil {
			return errors.Join(createErr, errors.New("could not create admin account"))
		}
	}
	// Create the admin account.
	return c.SetPassword(ctx, EverestAdminUser, pass, false)
}

// ExtractToken extracts token from context.
func ExtractToken(ctx context.Context) (*jwt.Token, error) {
	token, ok := ctx.Value(UserCtxKey).(*jwt.Token)
	if !ok {
		return nil, fmt.Errorf("failed to get token from context")
	}
	return token, nil
}
