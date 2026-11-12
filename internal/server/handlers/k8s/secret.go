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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/internal/secretdefinition"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// CreateSecret creates a Secret and labels it openeverest.io/managed, which
// marks it as created through the API and lets the API patch it later.
// Definition labels supplied by the caller are preserved; owner references,
// finalizers and usage counts are not, since OpenEverest manages the
// Secret's lifecycle and core alone counts its users. The response is the
// Secret's secretView.
func (h *k8sHandler) CreateSecret(ctx context.Context, _ string, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	secret.Labels[common.OpenEverestManagedLabel] = "true" //nolint:goconst
	secret.Namespace = namespace
	secret.OwnerReferences = nil
	secret.Finalizers = nil
	delete(secret.Annotations, common.AttachedDestinationsAnnotation)
	delete(secret.Annotations, common.AttachedStoragesAnnotation)

	res, err := h.kubeConnector.CreateSecret(ctx, secret)
	if err != nil {
		return nil, err
	}

	return secretView(res), nil
}

// ListSecrets lists the labelled Secrets matching the filter, each as its
// secretView, whether or not they were created through the API.
func (h *k8sHandler) ListSecrets(ctx context.Context, _ string, namespace string, filter handlers.SecretFilter) (*corev1.SecretList, error) {
	selector := ctrlclient.MatchingLabels{}
	if filter.Provider != "" {
		selector[common.OpenEverestProviderLabel] = filter.Provider
	}
	if filter.MonitoringClass != "" {
		selector[common.OpenEverestMonitoringClassLabel] = filter.MonitoringClass
	}
	if filter.Definition != "" {
		selector[common.OpenEverestDefinitionLabel] = filter.Definition
	}

	list, err := h.kubeConnector.ListSecrets(ctx,
		ctrlclient.InNamespace(namespace),
		ctrlclient.HasLabels{common.OpenEverestDefinitionLabel},
		selector,
	)
	if err != nil {
		return nil, err
	}

	items := make([]corev1.Secret, 0, len(list.Items))
	for i := range list.Items {
		// A label selector can't express "exactly one source label".
		if secretdefinition.Labelled(list.Items[i].GetLabels()) {
			items = append(items, *secretView(&list.Items[i]))
		}
	}
	list.Items = items

	return list, nil
}

// GetSecret returns the secretView of the labelled secret specified by name.
func (h *k8sHandler) GetSecret(ctx context.Context, _ string, namespace, name string) (*corev1.Secret, error) {
	secret, err := h.kubeConnector.GetSecret(ctx, ctrlclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	if !secretdefinition.Labelled(secret.GetLabels()) {
		return nil, secretNotFound(name)
	}

	return secretView(secret), nil
}

// PatchSecret applies the data of a merge patch to the labelled Secret named
// by the path, on the condition that it is still at the resourceVersion the
// patch carries. The Secret must be writable: created through the API and
// owned by no controller. The patch sent to Kubernetes holds that
// precondition and the data keys, nothing else. The response is the patched
// Secret's secretView.
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

	if !secretdefinition.Labelled(stored.GetLabels()) {
		return nil, secretNotFound(name)
	}
	if err := secretdefinition.Writable(stored); err != nil {
		return nil, err
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

// DeleteSecret deletes the labelled secret specified by name in the
// namespace, unless a controller owns it and would recreate it. A Secret the
// API did not create can be deleted: leftovers of helm uninstall or of Git
// would otherwise stay forever.
func (h *k8sHandler) DeleteSecret(ctx context.Context, _ string, namespace, name string) error {
	secret, err := h.kubeConnector.GetSecret(ctx, ctrlclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	if !secretdefinition.Labelled(secret.GetLabels()) {
		return secretNotFound(name)
	}
	if err := secretdefinition.Writable(secret); err != nil && !errors.Is(err, secretdefinition.ErrManagedOutside) {
		return err
	}

	return h.kubeConnector.DeleteSecret(ctx, secret)
}

// secretNotFound answers for a Secret the API does not serve, one without
// definition labels, exactly as Kubernetes answers for a missing one, so the
// API reveals nothing about Secrets it does not serve.
func secretNotFound(name string) error {
	return k8serrors.NewNotFound(corev1.Resource("secrets"), name)
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
