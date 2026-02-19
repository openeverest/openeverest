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

// Package kubernetes ...
package kubernetes

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/pbkdf2"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openeverest/openeverest/v2/pkg/accounts"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	// We set this annotation on the secret to indicate which passwords are stored in plain text.
	insecurePasswordAnnotation = "insecure-password/%s"
	insecurePasswordValueTrue  = "true"

	keyLength = 32
	iter      = 4096
)

type configMapsClient struct {
	k KubernetesConnector
}

// Accounts returns an implementation of the accounts interface that
// manages everest accounts directly via ConfigMaps.
func (k *Kubernetes) Accounts() accounts.Interface {
	return &configMapsClient{k: k}
}

// Get returns an account by username.
func (a *configMapsClient) Get(ctx context.Context, username string) (*accounts.Account, error) {
	users, err := a.listAllAccounts(ctx)
	if err != nil {
		return nil, err
	}
	user, found := users[username]
	if !found {
		return nil, accounts.ErrAccountNotFound
	}
	return user, nil
}

// List returns a list of all accounts.
func (a *configMapsClient) List(ctx context.Context) (map[string]*accounts.Account, error) {
	return a.listAllAccounts(ctx)
}

func (a *configMapsClient) listAllAccounts(ctx context.Context) (map[string]*accounts.Account, error) {
	result := make(map[string]*accounts.Account)
	secret, err := a.k.GetSecret(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestAccountsSecretName})
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(secret.Data[common.EverestAccountsFileName], result); err != nil {
		return nil, err
	}
	return result, nil
}

// Create a new user account.
func (a *configMapsClient) Create(ctx context.Context, username, password string) error {
	// Ensure that the user does not already exist.
	_, err := a.Get(ctx, username)
	if err != nil && !errors.Is(err, accounts.ErrAccountNotFound) {
		return errors.Join(err, errors.New("failed to check if account already exists"))
	} else if err == nil {
		return accounts.ErrUserAlreadyExists
	}

	if password == "" {
		return errors.New("password cannot be empty")
	}

	// Compute a hash for the password.
	hash, err := a.computePasswordHash(ctx, password)
	if err != nil {
		return errors.Join(err, errors.New("failed to compute hash"))
	}

	account := &accounts.Account{
		Enabled:       true,
		Capabilities:  []accounts.AccountCapability{accounts.AccountCapabilityLogin},
		PasswordMtime: time.Now().Format(time.RFC3339),
		PasswordHash:  hash,
	}
	return a.insertOrUpdateAccount(ctx, username, account, true)
}

// SetPassword sets a new password for an existing user account.
func (a *configMapsClient) SetPassword(ctx context.Context, username, newPassword string, secure bool) error {
	user, err := a.Get(ctx, username)
	if err != nil {
		return err
	}
	user.PasswordHash = newPassword
	if secure {
		pwHash, err := a.computePasswordHash(ctx, newPassword)
		if err != nil {
			return err
		}
		user.PasswordHash = pwHash
	}
	user.PasswordMtime = time.Now().Format(time.RFC3339)
	return a.insertOrUpdateAccount(ctx, username, user, secure)
}

func (a *configMapsClient) insertOrUpdateAccount(
	ctx context.Context,
	username string,
	account *accounts.Account,
	secure bool,
) error {
	secret, err := a.k.GetSecret(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestAccountsSecretName})
	if err != nil {
		return err
	}

	accounts := make(map[string]*accounts.Account)
	if err := yaml.Unmarshal(secret.Data[common.EverestAccountsFileName], &accounts); err != nil {
		return err
	}

	accounts[username] = account
	data, err := yaml.Marshal(accounts)
	if err != nil {
		return err
	}

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data[common.EverestAccountsFileName] = data

	annotations := secret.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	delete(annotations, fmt.Sprintf(insecurePasswordAnnotation, username))
	if !secure {
		annotations[fmt.Sprintf(insecurePasswordAnnotation, username)] = insecurePasswordValueTrue
	}
	secret.SetAnnotations(annotations)

	if _, err := a.k.UpdateSecret(ctx, secret); err != nil {
		return err
	}
	return nil
}

func (a *configMapsClient) salt(ctx context.Context) ([]byte, error) {
	ns, err := a.k.GetNamespace(ctx, types.NamespacedName{Name: common.SystemNamespace})
	if err != nil {
		return nil, err
	}
	return []byte(ns.UID), nil
}

// Delete an existing user account specified by username.
func (a *configMapsClient) Delete(ctx context.Context, username string) error {
	users, err := a.listAllAccounts(ctx)
	if err != nil {
		return err
	}

	if _, found := users[username]; !found {
		return accounts.ErrAccountNotFound
	}

	delete(users, username)
	secret, err := a.k.GetSecret(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestAccountsSecretName})
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(users)
	if err != nil {
		return err
	}
	secret.Data[common.EverestAccountsFileName] = data
	if _, err := a.k.UpdateSecret(ctx, secret); err != nil {
		return err
	}
	return nil
}

func (a *configMapsClient) Verify(ctx context.Context, username, password string) error {
	secret, err := a.k.GetSecret(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestAccountsSecretName})
	if err != nil {
		return err
	}

	users := make(map[string]*accounts.Account)
	if err := yaml.Unmarshal(secret.Data[common.EverestAccountsFileName], users); err != nil {
		return err
	}
	user, found := users[username]
	if !found {
		return accounts.ErrAccountNotFound
	}

	// helper to check if a password should be compared as a hash.
	shouldCompareAsHash := func() bool {
		annotations := secret.GetAnnotations()
		_, found := annotations[fmt.Sprintf(insecurePasswordAnnotation, username)]
		return !found
	}

	actual := user.PasswordHash
	provided := password

	if shouldCompareAsHash() {
		computedHash, err := a.computePasswordHash(ctx, password)
		if err != nil {
			return err
		}
		provided = computedHash
	}

	if subtle.ConstantTimeCompare([]byte(actual), []byte(provided)) == 0 {
		return accounts.ErrIncorrectPassword
	}
	return nil
}

// IsSecure returns true if the password for the given user is stored as a hash.
func (a *configMapsClient) IsSecure(ctx context.Context, username string) (bool, error) {
	secret, err := a.k.GetSecret(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestAccountsSecretName})
	if err != nil {
		return false, err
	}
	annotations := secret.GetAnnotations()
	isSecure, found := annotations[fmt.Sprintf(insecurePasswordAnnotation, username)]
	return !found || isSecure != insecurePasswordValueTrue, nil
}

func (a *configMapsClient) computePasswordHash(ctx context.Context, password string) (string, error) {
	salt, err := a.salt(ctx)
	if err != nil {
		return "", errors.Join(err, errors.New("failed to get salt"))
	}
	hash := pbkdf2.Key([]byte(password), salt, iter, keyLength, sha256.New)
	return string(hash), nil
}
