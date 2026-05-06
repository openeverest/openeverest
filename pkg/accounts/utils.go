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

package accounts

import (
	"errors"
	"regexp"
	"strings"
)

const (
	UsernameCriteria = "Username may contain only letters, numbers, underscores, and must be at least 3 characters long"
	PasswordCriteria = "Password may contain only letters, numbers and specific special characters (@*#$%^&+=!_-), and must be at least 6 characters long"
)

var (
	// Regular expression to validate username.
	// [a-zA-Z0-9_] - Allowed characters (letters, digits, underscore)
	// {3,} - Length of the username (minimum 3 characters)
	userNameValidateRegex = regexp.MustCompile("^[a-zA-Z0-9_]{3,}$")

	// ErrInvalidUsername is returned when the username doesn't match criteria.
	ErrInvalidUsername = errors.New(strings.ToLower(UsernameCriteria))

	// Regular expression to validate password.
	// [a-zA-Z0-9@*#$%^&+=!_-] - Allowed characters (letters, digits, special characters)
	// {6,} - Length of the password (minimum 6 characters)
	passwordValidateRegex = regexp.MustCompile("^[a-zA-Z0-9@*#$%^&+=!_-]{6,}$")

	// ErrInvalidNewPassword is returned when the new password doesn't match criteria.
	ErrInvalidNewPassword = errors.New(strings.ToLower(PasswordCriteria))
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
