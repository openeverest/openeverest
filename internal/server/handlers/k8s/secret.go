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

package k8s

import (
	"context"
	"errors"
	"maps"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// CreateSecret creates a Secret and labels it as managed by OpenEverest.
// Definition labels supplied by the caller are preserved; owner references
// and finalizers are not, since OpenEverest manages the Secret's lifecycle.
// The response is the Secret's secretView.
func (h *k8sHandler) CreateSecret(ctx context.Context, _ string, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	secret.Labels[common.OpenEverestManagedLabel] = "true" //nolint:goconst
	secret.Namespace = namespace
	secret.OwnerReferences = nil
	secret.Finalizers = nil

	res, err := h.kubeConnector.CreateSecret(ctx, secret)
	if err != nil {
		return nil, err
	}

	return secretView(res), nil
}

// ListSecrets lists OpenEverest-managed secrets matching the filter, each as
// its secretView.
func (h *k8sHandler) ListSecrets(ctx context.Context, _ string, namespace string, filter handlers.SecretFilter) (*corev1.SecretList, error) {
	selector := ctrlclient.MatchingLabels{common.OpenEverestManagedLabel: "true"}
	if filter.Provider != "" {
		selector[common.OpenEverestProviderLabel] = filter.Provider
	}
	if filter.MonitoringClass != "" {
		selector[common.OpenEverestMonitoringClassLabel] = filter.MonitoringClass
	}
	if filter.Definition != "" {
		selector[common.OpenEverestDefinitionLabel] = filter.Definition
	}

	list, err := h.kubeConnector.ListSecrets(ctx, ctrlclient.InNamespace(namespace), selector)
	if err != nil {
		return nil, err
	}

	for i := range list.Items {
		list.Items[i] = *secretView(&list.Items[i])
	}

	return list, nil
}

// GetSecret returns the secretView of the secret specified by name.
func (h *k8sHandler) GetSecret(ctx context.Context, _ string, namespace, name string) (*corev1.Secret, error) {
	secret, err := h.kubeConnector.GetSecret(ctx, ctrlclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	if !isManagedSecret(secret) {
		return nil, ErrNotFound
	}

	return secretView(secret), nil
}

// PatchSecret applies the data of a merge patch to the managed Secret named by
// the path, on the condition that it is still at the resourceVersion the patch
// carries. The patch sent to Kubernetes holds that precondition and the data
// keys, nothing else. The response is the patched Secret's secretView.
func (h *k8sHandler) PatchSecret(ctx context.Context, _ string, namespace, name string, patch []byte) (*corev1.Secret, error) {
	p, err := handlers.ParseSecretPatch(patch)
	if err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	stored, err := h.kubeConnector.GetSecret(ctx, ctrlclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	if !isManagedSecret(stored) {
		return nil, ErrNotFound
	}

	dataPatch, err := p.DataMergePatch()
	if err != nil {
		return nil, err
	}
	res, err := h.kubeConnector.PatchSecret(ctx,
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}},
		ctrlclient.RawPatch(types.MergePatchType, dataPatch),
	)
	if err != nil {
		return nil, err
	}

	return secretView(res), nil
}

// DeleteSecret deletes the secret specified by name in the namespace.
func (h *k8sHandler) DeleteSecret(ctx context.Context, _ string, namespace, name string) error {
	secret, err := h.kubeConnector.GetSecret(ctx, ctrlclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	if !isManagedSecret(secret) {
		return ErrNotFound
	}

	return h.kubeConnector.DeleteSecret(ctx, secret)
}

// isManagedSecret reports whether the HTTP API manages secret, i.e. it carries
// the label CreateSecret sets.
func isManagedSecret(secret *corev1.Secret) bool {
	return secret.Labels[common.OpenEverestManagedLabel] == "true"
}

// secretView returns what the HTTP API shows of a Secret: its identity, labels,
// owners, type and the annotations OpenEverest owns. It is an allowlist because
// the data does not only live in data and stringData: client-side kubectl apply
// copies the whole Secret into the last-applied-configuration annotation,
// managedFields names every key, and other tools keep hashes of the data in
// their own annotations.
func secretView(secret *corev1.Secret) *corev1.Secret {
	view := &corev1.Secret{
		TypeMeta: secret.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:              secret.Name,
			Namespace:         secret.Namespace,
			UID:               secret.UID,
			ResourceVersion:   secret.ResourceVersion,
			CreationTimestamp: secret.CreationTimestamp,
			DeletionTimestamp: secret.DeletionTimestamp,
			Labels:            maps.Clone(secret.Labels),
		},
		Type:      secret.Type,
		Immutable: secret.Immutable,
	}
	for _, ref := range secret.OwnerReferences {
		view.OwnerReferences = append(view.OwnerReferences, metav1.OwnerReference{
			APIVersion: ref.APIVersion,
			Kind:       ref.Kind,
			Name:       ref.Name,
			UID:        ref.UID,
			Controller: ref.Controller,
		})
	}
	for key, value := range secret.Annotations {
		if isOpenEverestKey(key) {
			metav1.SetMetaDataAnnotation(&view.ObjectMeta, key, value)
		}
	}
	return view
}

// isOpenEverestKey reports whether a label or annotation key is in the
// openeverest.io domain or one of its subdomains.
func isOpenEverestKey(key string) bool {
	prefix, _, found := strings.Cut(key, "/")
	return found && (prefix == openEverestDomain || strings.HasSuffix(prefix, "."+openEverestDomain))
}

const openEverestDomain = "openeverest.io"
