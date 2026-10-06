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
	"encoding/json"
	"fmt"

	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

const (
	// defaultStorageClassAnnotation is the standard Kubernetes annotation for marking a StorageClass as default
	defaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

	// namespaceDefaultsName is the singleton name of the NamespaceDefaults
	// object in each namespace.
	namespaceDefaultsName = "defaults"
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
	preset, err := h.kubeConnector.GetInstancePreset(ctx, types.NamespacedName{Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to get instance preset: %w", err)
	}

	// Create a copy to avoid modifying the original
	resolved := preset.DeepCopy()

	return h.resolveDefaults(ctx, resolved, namespace)
}

// resolveDefaults fills empty, namespace-scoped reference fields of
// the preset from the namespace's NamespaceDefaults object, and empty
// StorageClass fields from the cluster's default StorageClass.
//
// A NamespaceDefaults entry is matched to a reference by its path within the
// Instance spec: component parameter references under
// "components.<name>.parameters.<...>" and top-level references by their field
// name (e.g. "userSecretRef"). Only empty references are filled; StorageClass
// is cluster-scoped and keeps the standard Kubernetes "is-default-class"
// annotation.
//
// Missing defaults are not an error: an empty field is simply left empty.
func (h *k8sHandler) resolveDefaults(ctx context.Context, preset *corev1alpha1.InstancePreset, namespace string) (*corev1alpha1.InstancePreset, error) {
	defaults, err := h.kubeConnector.GetNamespaceDefaults(ctx, types.NamespacedName{Namespace: namespace, Name: namespaceDefaultsName})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get namespace defaults: %w", err)
	}

	defaultsByPath := defaultsByPath(defaults, preset.Spec.ProviderRef.Name)

	for componentName, component := range preset.Spec.Components {
		// Resolve Storage fields
		if component.Storage != nil {
			component, err = h.resolveStorageFields(ctx, component)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve component %s: %w", componentName, err)
			}
		}

		if len(defaultsByPath) == 0 {
			continue
		}

		// Resolve parameters fields
		if component.Parameters != nil && len(component.Parameters.Raw) > 0 {
			basePath := fmt.Sprintf("components.%s.parameters", componentName)
			component, err = resolveParametersFields(component, basePath, defaultsByPath)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve component %s: %w", componentName, err)
			}
		}

		preset.Spec.Components[componentName] = component
	}

	return preset, nil
}

// resolveStorageFields handles structured Storage.StorageClass.
func (h *k8sHandler) resolveStorageFields(ctx context.Context, component corev1alpha1.ComponentSpec) (corev1alpha1.ComponentSpec, error) {
	if component.Storage == nil {
		return component, nil
	}

	if isEmptyValue(component.Storage.StorageClass) {
		defaultStorageClass, err := h.findDefaultStorageClass(ctx)
		if err != nil {
			return component, err
		}
		if defaultStorageClass != nil {
			name := defaultStorageClass.GetName()
			component.Storage.StorageClass = &name
		}
	}

	return component, nil
}

// defaultsByPath resolves the namespace defaults into a path-keyed lookup for
// the given provider, applying provider precedence once.
func defaultsByPath(defaults *corev1alpha1.NamespaceDefaults, provider string) map[string]string {
	if defaults == nil {
		return nil
	}

	byPath := make(map[string]string, len(defaults.Spec.Defaults))
	for _, d := range defaults.Spec.Defaults {
		switch {
		case d.ProviderRef != nil && d.ProviderRef.Name == provider:
			// Provider match always wins, overwriting any agnostic entry.
			byPath[d.Path] = d.DefaultRef.Name
		case d.ProviderRef == nil:
			// Provider-agnostic default fills only where a provider-scoped
			// entry has not already filled.
			if _, ok := byPath[d.Path]; !ok {
				byPath[d.Path] = d.DefaultRef.Name
			}
		}
	}

	return byPath
}

// resolveParametersFields handles unstructured parameters fields recursively,
// building the spec path of each field so defaults can be matched by path.
func resolveParametersFields(
	component corev1alpha1.ComponentSpec,
	basePath string,
	defaultsByPath map[string]string,
) (corev1alpha1.ComponentSpec, error) {
	var data map[string]any
	if err := json.Unmarshal(component.Parameters.Raw, &data); err != nil {
		return component, err
	}

	if modified := resolveMapFieldsRecursive(data, basePath, defaultsByPath); modified {
		resolvedRaw, err := json.Marshal(data)
		if err != nil {
			return component, err
		}
		component.Parameters.Raw = resolvedRaw
	}

	return component, nil
}

// resolveMapFieldsRecursive walks parameters and fills empty reference fields.
// A reference object is filled if the object is {"name": ""} or an empty {}.
// A bare string ref is addressed directly. It returns whether anything changed.
func resolveMapFieldsRecursive(
	data map[string]any,
	basePath string,
	defaultsByPath map[string]string,
) bool {
	var modified bool

	for fieldName, value := range data {
		fieldPath := basePath + "." + fieldName

		if mapValue, ok := value.(map[string]any); ok {
			// A default at "<path>.name" fills the ref's name, creating the key
			// when the ref serialized empty ({}); otherwise recurse deeper.
			if defaultName := defaultsByPath[fieldPath+".name"]; defaultName != "" {
				if cur, exists := mapValue["name"]; !exists || isEmptyValue(cur) {
					mapValue["name"] = defaultName
					modified = true
				}
				continue
			}

			modified = resolveMapFieldsRecursive(mapValue, fieldPath, defaultsByPath) || modified
			continue
		}

		if defaultName := defaultsByPath[fieldPath]; defaultName != "" && isEmptyValue(value) {
			data[fieldName] = defaultName
			modified = true
		}
	}

	return modified
}

// isEmptyValue checks if value is empty/null
func isEmptyValue(value any) bool {
	switch v := value.(type) {
	case string:
		return v == ""
	case *string:
		return v == nil || *v == ""
	case map[string]any:
		// Empty object like {} or {"name": ""}
		if len(v) == 0 {
			return true
		}
		if len(v) == 1 {
			if nameVal, exists := v["name"]; exists {
				if name, ok := nameVal.(string); ok {
					return name == ""
				}
			}
		}
	}

	return false
}

// findDefaultStorageClass finds the most recent StorageClass using the same annotation
// as PVC finds the default StorageClass.
func (h *k8sHandler) findDefaultStorageClass(ctx context.Context) (*storagev1.StorageClass, error) {
	storageClasses, err := h.kubeConnector.ListStorageClasses(ctx)
	if err != nil {
		return nil, err
	}

	// Kubernetes API doesn't support annotation selectors, so we must list all StorageClasses
	// and filter client-side.
	filtered := make([]storagev1.StorageClass, 0)
	for _, sc := range storageClasses.Items {
		if annotations := sc.GetAnnotations(); annotations != nil {
			if annotations[defaultStorageClassAnnotation] == "true" {
				filtered = append(filtered, sc)
			}
		}
	}

	if len(filtered) == 0 {
		return nil, nil
	}

	mostRecent := filtered[0]
	for i := 1; i < len(filtered); i++ {
		if filtered[i].GetCreationTimestamp().After(mostRecent.GetCreationTimestamp().Time) {
			mostRecent = filtered[i]
		}
	}

	return &mostRecent, nil
}
