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

package tokenregistry

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	maxRetries      = 10
	backoffInterval = 500 * time.Millisecond

	// RotationGracePeriod is how long a rotated (consumed) refresh token remains
	// valid after rotation. This absorbs races where concurrent requests from the
	// same client refresh simultaneously.
	RotationGracePeriod = time.Minute
)

// ErrInvalidToken is returned for any token that cannot be validated
// (unknown, malformed, expired or hash mismatch). It is intentionally uniform
// to avoid leaking why validation failed.
var ErrInvalidToken = errors.New("invalid API token")

// Record is the at-rest representation of an opaque API token.
// The plaintext token is never persisted; only its SHA-256 hash is stored.
type Record struct {
	// ID is the stable identifier of the record. It is also embedded in the
	// plaintext token for O(1) lookup.
	ID string `json:"id"`
	// Name is a human label, unique per owner (used by PATs in a later phase).
	Name string `json:"name,omitempty"`
	// OwnerSubject is the owning account username (e.g. "alice").
	OwnerSubject string `json:"ownerSubject"`
	// Type is the token type.
	Type Type `json:"type"`
	// Hash is hex(SHA-256(plaintext)).
	Hash string `json:"hash"`
	// CreatedAt is the creation timestamp.
	CreatedAt time.Time `json:"createdAt"`
	// ExpiresAt is the expiry timestamp. Nil means the token never expires.
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	// LastUsedAt is a best-effort last-use timestamp (used by PATs in a later phase).
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

// SecretsClient contains the Kubernetes Secret operations needed by the registry.
type SecretsClient interface {
	// GetSecret returns a secret that matches the criteria.
	GetSecret(ctx context.Context, key ctrlclient.ObjectKey) (*corev1.Secret, error)
	// CreateSecret creates a secret.
	CreateSecret(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error)
	// PatchSecret patches a secret using the provided patch.
	PatchSecret(ctx context.Context, secret *corev1.Secret, patch ctrlclient.Patch) (*corev1.Secret, error)
}

// Registry stores opaque API tokens in a Kubernetes Secret, one key per record.
// Writes use JSON merge patches scoped to individual keys, so concurrent writes
// to different records do not conflict.
type Registry struct {
	client    SecretsClient
	l         *zap.SugaredLogger
	namespace string
	now       func() time.Time
}

// New creates a new token registry, ensuring the backing Secret exists.
func New(ctx context.Context, l *zap.SugaredLogger, client SecretsClient, namespace string) (*Registry, error) {
	r := &Registry{
		client:    client,
		l:         l,
		namespace: namespace,
		now:       time.Now,
	}
	if err := r.init(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) init(ctx context.Context) error {
	_, err := r.client.GetSecret(ctx, types.NamespacedName{Namespace: r.namespace, Name: common.EverestAuthTokensSecretName})
	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to get %s secret in the %s namespace: %w", common.EverestAuthTokensSecretName, r.namespace, err)
	}
	_, err = r.client.CreateSecret(ctx, &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.EverestAuthTokensSecretName,
			Namespace: r.namespace,
		},
	})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create %s secret in the %s namespace: %w", common.EverestAuthTokensSecretName, r.namespace, err)
	}
	return nil
}

// Mint creates a new token of the given type for the given owner and stores its record.
// A zero ttl creates a token that never expires.
// The returned plaintext is shown exactly once and never persisted.
func (r *Registry) Mint(ctx context.Context, owner string, t Type, ttl time.Duration) (string, Record, error) {
	plaintext, id, err := newPlaintext(t)
	if err != nil {
		return "", Record{}, err
	}

	now := r.now().UTC()
	rec := Record{
		ID:           id,
		OwnerSubject: owner,
		Type:         t,
		Hash:         hashPlaintext(plaintext),
		CreatedAt:    now,
	}
	if ttl > 0 {
		expiresAt := now.Add(ttl)
		rec.ExpiresAt = &expiresAt
	}

	if err := r.putRecord(ctx, rec); err != nil {
		return "", Record{}, fmt.Errorf("failed to store token record: %w", err)
	}
	return plaintext, rec, nil
}

// Validate checks a plaintext token against the registry and returns its record.
// Any failure (unknown, malformed, expired, hash mismatch) returns ErrInvalidToken.
func (r *Registry) Validate(ctx context.Context, plaintext string) (Record, error) {
	t, id, err := parsePlaintext(plaintext)
	if err != nil {
		return Record{}, ErrInvalidToken
	}

	secret, err := r.getSecret(ctx)
	if err != nil {
		return Record{}, fmt.Errorf("failed to read token registry: %w", err)
	}

	raw, ok := secret.Data[id]
	if !ok {
		return Record{}, ErrInvalidToken
	}
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return Record{}, ErrInvalidToken
	}

	if rec.Type != t {
		return Record{}, ErrInvalidToken
	}
	if subtle.ConstantTimeCompare([]byte(rec.Hash), []byte(hashPlaintext(plaintext))) != 1 {
		return Record{}, ErrInvalidToken
	}
	if rec.ExpiresAt != nil && r.now().UTC().After(*rec.ExpiresAt) {
		return Record{}, ErrInvalidToken
	}
	return rec, nil
}

// Rotate consumes the given record and mints a replacement token of the same
// type for the same owner with the given ttl (sliding window).
// The consumed record is not deleted immediately; its expiry is shortened to
// RotationGracePeriod so that concurrent refreshes from the same client do not fail.
func (r *Registry) Rotate(ctx context.Context, old Record, ttl time.Duration) (string, Record, error) {
	// Shorten the old record's expiry to the grace period (never extend it).
	graceExpiry := r.now().UTC().Add(RotationGracePeriod)
	if old.ExpiresAt == nil || graceExpiry.Before(*old.ExpiresAt) {
		old.ExpiresAt = &graceExpiry
		if err := r.putRecord(ctx, old); err != nil {
			return "", Record{}, fmt.Errorf("failed to grace-expire rotated token: %w", err)
		}
	}

	return r.Mint(ctx, old.OwnerSubject, old.Type, ttl)
}

// Revoke deletes the record with the given ID, immediately invalidating its token.
// Revoking a non-existent record is not an error.
func (r *Registry) Revoke(ctx context.Context, id string) error {
	if err := r.patchRecords(ctx, map[string]*Record{id: nil}); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

// PruneExpired removes all expired records from the registry.
func (r *Registry) PruneExpired(ctx context.Context) error {
	secret, err := r.getSecret(ctx)
	if err != nil {
		return fmt.Errorf("failed to read token registry: %w", err)
	}

	now := r.now().UTC()
	expired := make(map[string]*Record)
	for id, raw := range secret.Data {
		var rec Record
		if err := json.Unmarshal(raw, &rec); err != nil {
			r.l.Warnf("token registry record %q contains invalid data, removing", id)
			expired[id] = nil
			continue
		}
		if rec.ExpiresAt != nil && now.After(*rec.ExpiresAt) {
			expired[id] = nil
		}
	}
	if len(expired) == 0 {
		return nil
	}
	if err := r.patchRecords(ctx, expired); err != nil {
		return fmt.Errorf("failed to prune expired tokens: %w", err)
	}
	r.l.Debugf("pruned %d expired API token record(s)", len(expired))
	return nil
}

func (r *Registry) getSecret(ctx context.Context) (*corev1.Secret, error) {
	return r.client.GetSecret(ctx, types.NamespacedName{Namespace: r.namespace, Name: common.EverestAuthTokensSecretName})
}

func (r *Registry) putRecord(ctx context.Context, rec Record) error {
	return r.patchRecords(ctx, map[string]*Record{rec.ID: &rec})
}

// patchRecords applies a JSON merge patch to the registry Secret that sets
// (non-nil) or removes (nil) the given records. Only the affected keys are
// touched, so concurrent writers do not conflict.
func (r *Registry) patchRecords(ctx context.Context, records map[string]*Record) error {
	data := make(map[string]*string, len(records))
	for id, rec := range records {
		if rec == nil {
			data[id] = nil // JSON merge patch: null removes the key.
			continue
		}
		raw, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("failed to marshal token record: %w", err)
		}
		encoded := base64.StdEncoding.EncodeToString(raw)
		data[id] = &encoded
	}
	patch, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		return fmt.Errorf("failed to marshal token registry patch: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.EverestAuthTokensSecretName,
			Namespace: r.namespace,
		},
	}

	var bOff backoff.BackOff
	bOff = backoff.NewConstantBackOff(backoffInterval)
	bOff = backoff.WithMaxRetries(bOff, maxRetries)
	bOff = backoff.WithContext(bOff, ctx)
	return backoff.Retry(
		func() error {
			_, err := r.client.PatchSecret(ctx, secret.DeepCopy(), ctrlclient.RawPatch(types.MergePatchType, patch))
			return err
		},
		bOff,
	)
}
