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
	"fmt"
	"maps"
	"slices"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
)

// Field selectors of the BackupStorage selectable fields.
const (
	backupStorageClassField             = "spec.backupStorageClassRef.name"
	backupStorageCredentialsSecretField = "spec.authentication.credentialsSecretRef.name"
	backupStorageCASecretField          = "spec.tls.ca.secretKeyRef.name" //nolint:gosec // a field path, not a credential
)

// ListBackupStorages returns the backup storages in a namespace that match the
// filter. The backup storage class and Secret filters are evaluated by the API
// server through the CRD's selectable fields; the backup class filter is
// computed here from the class's spec.backupStorageClasses.
func (h *k8sHandler) ListBackupStorages(ctx context.Context, _, namespace string, filter handlers.BackupStorageFilter) (*backupv1alpha1.BackupStorageList, error) {
	fields := ctrlclient.MatchingFields{}
	if filter.BackupStorageClass != "" {
		fields[backupStorageClassField] = filter.BackupStorageClass
	}
	list, err := h.listBackupStorages(ctx, namespace, fields, filter.Secret)
	if err != nil || filter.BackupClass == "" {
		return list, err
	}
	class, err := h.kubeConnector.GetBackupClass(ctx, types.NamespacedName{Name: filter.BackupClass})
	if apierrors.IsNotFound(err) {
		list.Items = []backupv1alpha1.BackupStorage{}
		return list, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get BackupClass %q: %w", filter.BackupClass, err)
	}
	filtered := make([]backupv1alpha1.BackupStorage, 0, len(list.Items))
	for _, bs := range list.Items {
		if class.CanWriteTo(&bs).Compatible {
			filtered = append(filtered, bs)
		}
	}
	list.Items = filtered
	return list, nil
}

// listBackupStorages lists the storages matching fields and, when secret is
// set, referencing it. A field selector cannot express "or", so a storage that
// references the Secret as credentials and one that references it as CA bundle
// come from two lists, merged and deduplicated.
func (h *k8sHandler) listBackupStorages(ctx context.Context, namespace string, fields ctrlclient.MatchingFields, secret string) (*backupv1alpha1.BackupStorageList, error) {
	if secret == "" {
		opts := []ctrlclient.ListOption{ctrlclient.InNamespace(namespace)}
		if len(fields) > 0 {
			opts = append(opts, fields)
		}
		return h.kubeConnector.ListBackupStorages(ctx, opts...)
	}
	merged := &backupv1alpha1.BackupStorageList{Items: []backupv1alpha1.BackupStorage{}}
	seen := make(map[string]bool)
	for i, field := range []string{backupStorageCredentialsSecretField, backupStorageCASecretField} {
		selector := maps.Clone(fields)
		selector[field] = secret
		list, err := h.kubeConnector.ListBackupStorages(ctx, ctrlclient.InNamespace(namespace), selector)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			merged.TypeMeta, merged.ListMeta = list.TypeMeta, list.ListMeta
		}
		for _, bs := range list.Items {
			if !seen[bs.GetName()] {
				seen[bs.GetName()] = true
				merged.Items = append(merged.Items, bs)
			}
		}
	}
	slices.SortFunc(merged.Items, func(a, b backupv1alpha1.BackupStorage) int {
		return strings.Compare(a.GetName(), b.GetName())
	})
	return merged, nil
}

// GetBackupStorage returns a backup storage by name and namespace.
func (h *k8sHandler) GetBackupStorage(ctx context.Context, _, namespace, name string) (*backupv1alpha1.BackupStorage, error) {
	return h.kubeConnector.GetBackupStorage(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// CreateBackupStorage creates a backup storage. Its credentials Secret is
// created beforehand, through the Secrets API.
func (h *k8sHandler) CreateBackupStorage(ctx context.Context, _ string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	return h.kubeConnector.CreateBackupStorage(ctx, bs)
}

// UpdateBackupStorage updates a backup storage.
func (h *k8sHandler) UpdateBackupStorage(ctx context.Context, _ string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	return h.kubeConnector.UpdateBackupStorage(ctx, bs)
}

// PatchBackupStorage applies a merge patch to a backup storage, leaving the read-modify-write to the API server.
// Strict validation is not optional: without it a misspelt path is pruned and returns 200 having changed nothing.
func (h *k8sHandler) PatchBackupStorage(ctx context.Context, _, namespace, name string, patch []byte) (*backupv1alpha1.BackupStorage, error) {
	bs := &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
	return h.kubeConnector.PatchBackupStorage(ctx, bs,
		ctrlclient.RawPatch(types.MergePatchType, patch),
		ctrlclient.FieldValidation(metav1.FieldValidationStrict),
	)
}

// DeleteBackupStorage deletes a backup storage.
func (h *k8sHandler) DeleteBackupStorage(ctx context.Context, _, namespace, name string) error {
	bs, err := h.kubeConnector.GetBackupStorage(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if ctrlclient.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get backup storage: %w", err)
	}

	if bs == nil {
		// nothing to delete
		return nil
	}

	if err := h.kubeConnector.DeleteBackupStorage(ctx, bs); ctrlclient.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete backup storage: %w", err)
	}

	return nil
}

// ListBackupStorageClasses returns the backup storage classes.
func (h *k8sHandler) ListBackupStorageClasses(ctx context.Context, _ string) (*backupv1alpha1.BackupStorageClassList, error) {
	return h.kubeConnector.ListBackupStorageClasses(ctx)
}

// GetBackupStorageClass returns a backup storage class by name.
func (h *k8sHandler) GetBackupStorageClass(ctx context.Context, _, name string) (*backupv1alpha1.BackupStorageClass, error) {
	return h.kubeConnector.GetBackupStorageClass(ctx, types.NamespacedName{Name: name})
}
