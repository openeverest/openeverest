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
	"maps"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

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

	if err := h.validateSecretDefinition(ctx, secret); err != nil {
		return nil, err
	}

	return h.next.CreateSecret(ctx, cluster, namespace, secret)
}

// PatchSecret refuses a patch written against an older version of the Secret
// or one that would change its definition, checks the patched data against the
// definition, and proxies to the next handler.
func (h *validateHandler) PatchSecret(ctx context.Context, cluster, namespace, name string, patch []byte) (*corev1.Secret, error) {
	p, err := handlers.ParseSecretPatch(patch)
	if err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	stored, err := h.kubeConnector.GetSecret(ctx, ctrlclient.ObjectKey{Namespace: namespace, Name: name})
	if err != nil {
		return nil, err
	}
	// The next handler answers for a Secret the API does not show or may not
	// change, before anything about the patch is checked.
	if !secretdefinition.Labelled(stored.GetLabels()) || secretdefinition.Writable(stored) != nil {
		return h.next.PatchSecret(ctx, cluster, namespace, name, patch)
	}
	// Checked here, before the data, so a change made since reads as a
	// conflict rather than as whatever that change did to the data.
	if stored.GetResourceVersion() != p.ResourceVersion {
		return nil, k8serrors.NewConflict(corev1.Resource("secrets"), name,
			errors.New("the object has been modified; please apply your changes to the latest version and try again"),
		)
	}

	if err := checkSecretPatchKeepsDefinition(stored, p); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	candidate := stored.DeepCopy()
	candidate.Data = p.Apply(stored.Data)
	candidate.StringData = nil
	if err := h.validateSecretDefinition(ctx, candidate); err != nil {
		return nil, err
	}

	return h.next.PatchSecret(ctx, cluster, namespace, name, patch)
}

// checkSecretPatchKeepsDefinition refuses a patch that would change the
// Secret's type, labels or immutability. Sending them with their stored
// values is accepted, so a Secret returned by GET can be sent back as a patch.
func checkSecretPatchKeepsDefinition(stored *corev1.Secret, p *handlers.SecretPatch) error {
	if p.Type != nil && *p.Type != stored.Type {
		return fmt.Errorf("type can't change from %q to %q; create a new Secret instead", stored.Type, *p.Type)
	}
	for _, key := range slices.Sorted(maps.Keys(p.Labels)) {
		if value, found := stored.GetLabels()[key]; !found || p.Labels[key] == nil || value != *p.Labels[key] {
			return fmt.Errorf("label %q differs from the stored Secret; a patch keeps the labels, create a new Secret to change them", key)
		}
	}
	if p.Immutable != nil && *p.Immutable != (stored.Immutable != nil && *stored.Immutable) {
		return errors.New("immutable can't change through the API")
	}
	return nil
}

// validateSecretDefinition checks secret against the definition its labels
// name, as an invalid request when the labels or the Secret are at fault.
func (h *validateHandler) validateSecretDefinition(ctx context.Context, secret *corev1.Secret) error {
	if err := secretdefinition.Validate(ctx, h.kubeConnector, secret); err != nil {
		if errors.Is(err, secretdefinition.ErrInvalidLabels) ||
			errors.Is(err, secretdefinition.ErrSourceNotFound) ||
			errors.Is(err, secretdefinition.ErrInvalidSecret) {
			return errors.Join(ErrInvalidRequest, err)
		}
		return err
	}
	return nil
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
	// TODO: prevent deletion of secrets that are still in use by an Instance.
	return h.next.DeleteSecret(ctx, cluster, namespace, name)
}
