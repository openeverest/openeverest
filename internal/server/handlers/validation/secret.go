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

package validation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"

	"github.com/openeverest/openeverest/v2/internal/secretdefinition"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
)

// CreateSecret validates the secret request and proxies it to the next handler.
// The Secret must name its definition with the labels secretdefinition
// describes, and match it.
func (h *validateHandler) CreateSecret(ctx context.Context, cluster, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	if errs := k8svalidation.IsDNS1123Subdomain(secret.GetName()); len(errs) > 0 {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("invalid name %q: %s", secret.GetName(), strings.Join(errs, "; ")))
	}

	if err := secretdefinition.Validate(ctx, h.kubeConnector, secret); err != nil {
		if errors.Is(err, secretdefinition.ErrInvalidLabels) ||
			errors.Is(err, secretdefinition.ErrSourceNotFound) ||
			errors.Is(err, secretdefinition.ErrInvalidSecret) {
			return nil, errors.Join(ErrInvalidRequest, err)
		}
		return nil, err
	}

	return h.next.CreateSecret(ctx, cluster, namespace, secret)
}

// ListSecrets proxies the request to the next handler.
func (h *validateHandler) ListSecrets(ctx context.Context, cluster, namespace string, filter handlers.SecretFilter) (*corev1.SecretList, error) {
	return h.next.ListSecrets(ctx, cluster, namespace, filter)
}

// GetSecret proxies the request to the next handler.
func (h *validateHandler) GetSecret(ctx context.Context, cluster, namespace, name string) (*corev1.Secret, error) {
	return h.next.GetSecret(ctx, cluster, namespace, name)
}

// DeleteSecret proxies the request to the next handler.
func (h *validateHandler) DeleteSecret(ctx context.Context, cluster, namespace, name string) error {
	// TODO: prevent deletion of secrets that are still in use by an Instance (spec 011 §4.5).
	return h.next.DeleteSecret(ctx, cluster, namespace, name)
}
