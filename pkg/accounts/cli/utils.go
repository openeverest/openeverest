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

// Package cli holds commands for accounts command.
package cli

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/openeverest/openeverest/v2/pkg/cli/tui"
)

const (
	usernameCriteria = "Username may contain only letters, numbers, underscores, and must be at least 3 characters long"
	passwordCriteria = "Password may contain only letters, numbers and specific special characters (@*#$%^&+=!_-), and must be at least 6 characters long"
)

var (
	// Regular expression to validate username.
	// [a-zA-Z0-9_] - Allowed characters (letters, digits, underscore)
	// {3,} - Length of the username (minimum 3 characters)
	userNameValidateRegex = regexp.MustCompile("^[a-zA-Z0-9_]{3,}$")

	// ErrInvalidUsername is returned when the username doesn't match criteria.
	ErrInvalidUsername = errors.New(strings.ToLower(usernameCriteria))

	// Regular expression to validate password.
	// [a-zA-Z0-9@*#$%^&+=!_-] - Allowed characters (letters, digits, special characters)
	// {6,} - Length of the password (minimum 6 characters)
	passwordValidateRegex = regexp.MustCompile("^[a-zA-Z0-9@*#$%^&+=!_-]{6,}$")

	// ErrInvalidNewPassword is returned when the new password doesn't match criteria.
	ErrInvalidNewPassword = errors.New(strings.ToLower(passwordCriteria))
)

// ValidateUsername validates the provided username.
func ValidateUsername(username string) error {
	if !userNameValidateRegex.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}

// ValidatePassword validates the provided password.
func ValidatePassword(password string) error {
	if !passwordValidateRegex.MatchString(password) {
		return ErrInvalidNewPassword
	}
	return nil
}

// PopulateUsername function to fill the username.
// This function shall be called only in cases when there is no other way to obtain username value.
// User will be asked to provide the username in interactive mode.
func PopulateUsername(ctx context.Context) (string, error) {
	if username, err := tui.NewInput(ctx, "Provide username",
		tui.WithInputHint(usernameCriteria),
		tui.WithInputValidation(ValidateUsername),
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
		tui.WithPasswordHint(passwordCriteria),
		tui.WithPasswordValidation(ValidatePassword),
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
		tui.WithPasswordHint(passwordCriteria),
		tui.WithPasswordValidation(ValidatePassword),
	).Run(); err != nil {
		return "", err
	}

	if newConfPassword, err = tui.NewInputPassword(ctx, "Confirm a new password",
		tui.WithPasswordHint(passwordCriteria),
		tui.WithPasswordValidation(ValidatePassword),
	).Run(); err != nil {
		return "", err
	}

	if newPassword != newConfPassword {
		return "", errors.New("passwords do not match")
	}

	return newPassword, nil
}
