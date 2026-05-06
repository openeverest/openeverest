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

// Package cli holds commands for accounts command.
package cli

import (
	"context"
	"errors"

	"github.com/percona/everest/pkg/accounts"
	"github.com/percona/everest/pkg/cli/tui"
)

// PopulateUsername function to fill the username.
// This function shall be called only in cases when there is no other way to obtain username value.
// User will be asked to provide the username in interactive mode.
func PopulateUsername(ctx context.Context) (string, error) {
	if username, err := tui.NewInput(ctx, "Provide username",
		tui.WithInputHint(accounts.UsernameCriteria),
		tui.WithInputValidation(accounts.ValidateUsername),
	).Run(); err != nil {
		return "", err
	} else {
		return username, nil
	}
}

// PopulatePassword function to fill the password.
// This function shall be called only in cases when there is no other way to obtain password value.
// User will be asked to provide the password in interactive mode.
func PopulatePassword(ctx context.Context) (string, error) {
	// ask user to provide password
	if password, err := tui.NewInputPassword(ctx, "Provide password",
		tui.WithPasswordHint(accounts.PasswordCriteria),
		tui.WithPasswordValidation(accounts.ValidatePassword),
	).Run(); err != nil {
		return "", err
	} else {
		return password, nil
	}
}

// PopulateNewPassword function to fill the new password.
// This function shall be called only in cases when there is no other way to obtain new password value.
// User will be asked to provide the new password and password confirmation in interactive mode.
func PopulateNewPassword(ctx context.Context) (string, error) {
	// ask user to provide new password
	var newPassword, newConfPassword string
	var err error
	if newPassword, err = tui.NewInputPassword(ctx, "Provide a new password",
		tui.WithPasswordHint(accounts.PasswordCriteria),
		tui.WithPasswordValidation(accounts.ValidatePassword),
	).Run(); err != nil {
		return "", err
	}

	if newConfPassword, err = tui.NewInputPassword(ctx, "Confirm a new password",
		tui.WithPasswordHint(accounts.PasswordCriteria),
		tui.WithPasswordValidation(accounts.ValidatePassword),
	).Run(); err != nil {
		return "", err
	}

	if newPassword != newConfPassword {
		return "", errors.New("passwords do not match")
	}

	return newPassword, nil
}
