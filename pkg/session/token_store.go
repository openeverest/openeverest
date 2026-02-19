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
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	sep          = ","
	timestampLen = 10
)

// TokenStoreClient contains the methods that are needed for the token store management.
type TokenStoreClient interface {
	// GetSecret returns a secret that matches the criteria.
	GetSecret(ctx context.Context, key client.ObjectKey) (*corev1.Secret, error)
	// CreateSecret creates a secret.
	CreateSecret(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error)
	// UpdateSecret updates a secret.
	UpdateSecret(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error)
}

func newTokenStore(ctx context.Context, client TokenStoreClient, logger *zap.SugaredLogger) (TokenStore, error) {
	s := &tokenStore{
		l:      logger,
		client: client,
	}
	err := s.init(ctx)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (ts *tokenStore) init(ctx context.Context) error {
	_, err := ts.client.GetSecret(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestBlocklistSecretName})
	if err == nil {
		return err
	}
	if !k8serrors.IsNotFound(err) {
		err = fmt.Errorf("failed to get %s secret in the %s namespace: %w", common.EverestBlocklistSecretName, common.SystemNamespace, err)
		ts.l.Error(err)
		return err
	}
	var createErr error
	secret := getBlockListSecretTemplate("")
	_, createErr = ts.client.CreateSecret(ctx, secret)
	if createErr != nil {
		err = fmt.Errorf("failed to create secret %s in namespace %s: %w", secret.Name, secret.Namespace, createErr)
		ts.l.Error(err)
		return err
	}
	return nil
}

type tokenStore struct {
	client TokenStoreClient
	l      *zap.SugaredLogger
}

// Add adds the shortened token to the blocklist
func (ts *tokenStore) Add(ctx context.Context, shortenedToken string) error {
	secret, err := ts.client.GetSecret(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestBlocklistSecretName})
	if err != nil {
		ts.l.Errorf("failed to get %s secret in the %s namespace: %v", common.EverestBlocklistSecretName, common.SystemNamespace, err)
		return err
	}

	secret = addDataToSecret(ts.l, secret, shortenedToken, time.Now().UTC())
	_, updateErr := ts.client.UpdateSecret(ctx, secret)
	if updateErr != nil {
		ts.l.Errorf("failed to update %s secret in the %s namespace with the %s shortened token, retrying: %v", secret.Name, secret.Namespace, shortenedToken, updateErr)
		return err
	}
	return nil
}

// Exists checks if the shortened token is in the blocklist
func (ts *tokenStore) Exists(ctx context.Context, shortenedToken string) (bool, error) {
	// no worries about overwhelming k8s API - the secret is cached
	secret, err := ts.client.GetSecret(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestBlocklistSecretName})
	if err != nil {
		ts.l.Errorf("failed to get %s secret in the %s namespace: %v", common.EverestBlocklistSecretName, common.SystemNamespace, err)
		return false, err
	}
	list, ok := secret.Data[dataKey]
	return ok && strings.Contains(string(list), shortenedToken), nil
}

func addDataToSecret(l *zap.SugaredLogger, secret *corev1.Secret, shortenedToken string, now time.Time) *corev1.Secret {
	existingList, ok := secret.Data[dataKey]
	if !ok {
		secret.StringData = map[string]string{
			dataKey: shortenedToken,
		}
		return secret
	}
	list := append(cleanupOld(l, string(existingList), now), shortenedToken)
	secret.StringData = map[string]string{
		dataKey: strings.Join(list, sep),
	}
	return secret
}

func cleanupOld(l *zap.SugaredLogger, list string, now time.Time) []string {
	tokens := strings.Split(list, sep)
	newList := make([]string, 0, len(tokens))
	for _, shortenedToken := range tokens {
		length := len(shortenedToken)
		if length < timestampLen {
			l.Warnf("blocklist token='%s' contains irregular data format", shortenedToken)
			continue
		}
		ts := shortenedToken[length-10 : length]
		tsInt, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			l.Warnf("failed to parse timestamp %v", tsInt)
			continue
		}
		timeObj := time.Unix(tsInt, 0)
		// only keep the tokens which natural expiration time is not over yet
		if timeObj.After(now) {
			newList = append(newList, shortenedToken)
		}
	}
	return newList
}

func getBlockListSecretTemplate(stringData string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.EverestBlocklistSecretName,
			Namespace: common.SystemNamespace,
		},
		StringData: map[string]string{
			dataKey: stringData,
		},
	}
}
