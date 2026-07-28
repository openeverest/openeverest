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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/preset"
)

const (
	// defaultStorageClassAnnotation is the standard Kubernetes annotation for marking a StorageClass as default.
	defaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"
)

// ListInstancePresets returns list of instance presets, optionally filtered by provider.
func (h *k8sHandler) ListInstancePresets(ctx context.Context, cluster string, provider string) (*corev1alpha1.InstancePresetList, error) {
	list, err := h.kubeConnector.ListInstancePresets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list instance presets: %w", err)
	}

	if provider != "" {
		filtered := make([]corev1alpha1.InstancePreset, 0)
		for _, preset := range list.Items {
			if preset.Spec.ProviderRef.Name == provider {
				filtered = append(filtered, preset)
			}
		}
		list.Items = filtered
	}

	return list, nil
}

// GetInstancePreset returns an instance preset that matches the criteria.
func (h *k8sHandler) GetInstancePreset(ctx context.Context, cluster, name string) (*corev1alpha1.InstancePreset, error) {
	return h.kubeConnector.GetInstancePreset(ctx, types.NamespacedName{Name: name})
}

// ResolveInstancePreset returns an instance preset with namespace-specific default values populated.
func (h *k8sHandler) ResolveInstancePreset(ctx context.Context, cluster, name, namespace string) (*corev1alpha1.InstancePreset, error) {
	instancePreset, err := h.kubeConnector.GetInstancePreset(ctx, types.NamespacedName{Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to get instance preset: %w", err)
	}

	// Create a copy to avoid modifying the original
	resolved := instancePreset.DeepCopy()

	return h.resolveNamespaceDefaults(ctx, resolved, namespace)
}

// CreateInstancePreset creates an instance preset.
func (h *k8sHandler) CreateInstancePreset(ctx context.Context, cluster string, instancePreset *corev1alpha1.InstancePreset) (*corev1alpha1.InstancePreset, error) {
	if err := h.kubeConnector.CreateInstancePreset(ctx, instancePreset); err != nil {
		return nil, fmt.Errorf("failed to create instance preset: %w", err)
	}

	return h.kubeConnector.GetInstancePreset(ctx, types.NamespacedName{Name: instancePreset.Name})
}

// UpdateInstancePreset updates an instance preset.
func (h *k8sHandler) UpdateInstancePreset(ctx context.Context, cluster string, instancePreset *corev1alpha1.InstancePreset) (*corev1alpha1.InstancePreset, error) {
	if err := h.kubeConnector.UpdateInstancePreset(ctx, instancePreset); err != nil {
		return nil, fmt.Errorf("failed to update instance preset: %w", err)
	}

	return h.kubeConnector.GetInstancePreset(ctx, types.NamespacedName{Name: instancePreset.Name})
}

// DeleteInstancePreset deletes an instance preset.
func (h *k8sHandler) DeleteInstancePreset(ctx context.Context, cluster, name string) error {
	if err := h.kubeConnector.DeleteInstancePreset(ctx, types.NamespacedName{Name: name}); err != nil {
		if ctrlclient.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to delete instance preset: %w", err)
		}
	}

	return nil
}

// CreateInstancePresetFromInstance creates a new InstancePreset from an existing Instance.
func (h *k8sHandler) CreateInstancePresetFromInstance(ctx context.Context, cluster, namespace, instanceName, presetName string) (*corev1alpha1.InstancePreset, error) {
	// Get the instance
	instance, err := h.kubeConnector.GetInstance(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      instanceName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Build the preset from a copy of the instance spec so the fetched instance is
	// not mutated, then strip namespace-scoped references.
	newPreset := &corev1alpha1.InstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name: presetName,
		},
		Spec: corev1alpha1.InstancePresetSpec{
			InstanceSpec: *instance.Spec.DeepCopy(),
		},
	}

	if err := preset.ClearNamespaceRefs(&newPreset.Spec.InstanceSpec); err != nil {
		return nil, fmt.Errorf("failed to clear namespace-scoped fields: %w", err)
	}

	if err := h.kubeConnector.CreateInstancePreset(ctx, newPreset); err != nil {
		return nil, fmt.Errorf("failed to create instance preset: %w", err)
	}

	return h.kubeConnector.GetInstancePreset(ctx, types.NamespacedName{Name: presetName})
}

// resolveNamespaceDefaults fills empty namespace-scoped references (and empty
// StorageClass) in the preset with the defaults resolved for the namespace.
func (h *k8sHandler) resolveNamespaceDefaults(ctx context.Context, instancePreset *corev1alpha1.InstancePreset, namespace string) (*corev1alpha1.InstancePreset, error) {
	if err := preset.ResolveNamespaceRefs(ctx, &instancePreset.Spec.InstanceSpec, namespace, h); err != nil {
		return nil, fmt.Errorf("failed to resolve namespace defaults: %w", err)
	}
	return instancePreset, nil
}

// ResolveDefault implements preset.DefaultResolver. It returns the name of the
// most recently created resource of the given kind that carries the default
// annotation, or "" when none exists. Namespace-scoped kinds use the
// component-specific annotation "openeverest.io/is-default-components-<component>";
// StorageClass uses the standard Kubernetes default-class annotation and ignores
// the namespace.
func (h *k8sHandler) ResolveDefault(ctx context.Context, namespace string, kind preset.ResourceKind, component string) (string, error) {
	componentAnnotation := fmt.Sprintf("openeverest.io/is-default-components-%s", component)

	switch kind {
	case preset.KindSecret:
		list, err := h.kubeConnector.ListSecrets(ctx, ctrlclient.InNamespace(namespace))
		if err != nil {
			return "", err
		}
		return mostRecentDefault(toPtrs(list.Items), componentAnnotation), nil
	case preset.KindConfigMap:
		list, err := h.kubeConnector.ListConfigMaps(ctx, ctrlclient.InNamespace(namespace))
		if err != nil {
			return "", err
		}
		return mostRecentDefault(toPtrs(list.Items), componentAnnotation), nil
	case preset.KindMonitoringConfig:
		list, err := h.kubeConnector.ListMonitoringConfigsV2(ctx, ctrlclient.InNamespace(namespace))
		if err != nil {
			return "", err
		}
		return mostRecentDefault(toPtrs(list.Items), componentAnnotation), nil
	case preset.KindStorageClass:
		list, err := h.kubeConnector.ListStorageClasses(ctx)
		if err != nil {
			return "", err
		}
		return mostRecentDefault(toPtrs(list.Items), defaultStorageClassAnnotation), nil
	default:
		return "", nil
	}
}

// toPtrs returns a slice of pointers to the elements of items.
func toPtrs[T any](items []T) []*T {
	out := make([]*T, len(items))
	for i := range items {
		out[i] = &items[i]
	}
	return out
}

// mostRecentDefault returns the name of the most recently created object that
// carries annotationKey set to "true", or "" if none match.
func mostRecentDefault[T ctrlclient.Object](items []T, annotationKey string) string {
	var newest T
	found := false
	for _, item := range items {
		if item.GetAnnotations()[annotationKey] != "true" {
			continue
		}
		if !found || item.GetCreationTimestamp().After(newest.GetCreationTimestamp().Time) {
			newest = item
			found = true
		}
	}
	if !found {
		return ""
	}
	return newest.GetName()
}
