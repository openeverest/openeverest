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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
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
			if preset.Spec.Provider == provider {
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

	return h.resolveNamespaceDefaults(ctx, resolved, namespace)
}

// resolveNamespaceDefaults scans components and resolves
// empty namespace reference fields and populates them.
// The fields that could have namespace references are in config and customSpec.
// It skips other fields like resources, image, etc. since they are not
// namespace-specific, and also skips fields with unknown type.
// Supported types are Secret and MonitoringConfig.
// The resolution is based on the most recently created resource with the
// annotation "openeverest.io/is-default-components-<componentName>" set
// to "true" in the specified namespace.
func (h *k8sHandler) resolveNamespaceDefaults(ctx context.Context, preset *corev1alpha1.InstancePreset, namespace string) (*corev1alpha1.InstancePreset, error) {
	for componentName, component := range preset.Spec.Components {
		var err error

		// Resolve Config fields
		if component.Config != nil {
			component, err = h.resolveConfigFields(ctx, component, componentName, namespace)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve component %s: %w", componentName, err)
			}
		}

		// Resolve customSpec fields
		if component.CustomSpec != nil && len(component.CustomSpec.Raw) > 0 {
			component, err = h.resolveCustomSpecFields(ctx, component, componentName, namespace)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve component %s: %w", componentName, err)
			}
		}

		preset.Spec.Components[componentName] = component
	}

	return preset, nil
}

// resolveConfigFields handles structured Config.SecretRef.Name.
// TODO: support Config.ConfigMapRef.Name.
func (h *k8sHandler) resolveConfigFields(ctx context.Context, component corev1alpha1.ComponentSpec, componentName, namespace string) (corev1alpha1.ComponentSpec, error) {
	if component.Config == nil {
		return component, nil
	}

	if isEmptyValue(component.Config.SecretRef) {
		defaultSecretName, err := h.findDefaultResource(ctx, namespace, "Secret", componentName)
		if err != nil {
			return component, err
		}
		component.Config.SecretRef.Name = defaultSecretName
	}

	return component, nil
}

// resolveCustomSpecFields handles unstructured customSpec fields recursively.
func (h *k8sHandler) resolveCustomSpecFields(ctx context.Context, component corev1alpha1.ComponentSpec, componentName, namespace string) (corev1alpha1.ComponentSpec, error) {
	var data map[string]any
	if err := json.Unmarshal(component.CustomSpec.Raw, &data); err != nil {
		return component, err
	}

	modified, err := h.resolveMapFieldsRecursive(ctx, data, componentName, namespace)
	if err != nil {
		return component, err
	}

	if modified {
		resolvedRaw, err := json.Marshal(data)
		if err != nil {
			return component, err
		}
		component.CustomSpec.Raw = resolvedRaw
	}

	return component, nil
}

// resolveMapFieldsRecursive walks customSpec and resolves empty fields matching patterns
func (h *k8sHandler) resolveMapFieldsRecursive(ctx context.Context, data map[string]any, componentName, namespace string) (bool, error) {
	var modified bool

	for fieldName, value := range data {
		resourceType := inferSupportedResourceType(fieldName)
		if resourceType == "" {
			if nested, ok := value.(map[string]any); ok {
				m, err := h.resolveMapFieldsRecursive(ctx, nested, componentName, namespace)
				if err != nil {
					return modified, err
				}

				modified = modified || m
			}
			continue
		}

		// Try to resolve if value is empty
		if !isEmptyValue(value) {
			continue
		}

		defaultName, err := h.findDefaultResource(ctx, namespace, resourceType, componentName)
		if err != nil {
			return modified, err
		}

		if defaultName == "" {
			continue
		}

		// Handle ref-like objects (e.g., {} or {"name": ""})
		if refMap, ok := value.(map[string]any); ok {
			refMap["name"] = defaultName
		} else {
			// Handle string values
			data[fieldName] = defaultName
		}

		modified = true
	}

	return modified, nil
}

// isEmptyValue checks if value is empty/null
func isEmptyValue(value any) bool {
	switch v := value.(type) {
	case string:
		return v == ""
	case corev1.LocalObjectReference:
		return v.Name == ""
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

// findDefaultResource finds the default resource for a component field
func (h *k8sHandler) findDefaultResource(ctx context.Context, namespace, resourceType, componentName string) (string, error) {
	if resourceType == "" {
		return "", nil
	}

	// Build annotation key for this component
	annotationKey := fmt.Sprintf("openeverest.io/is-default-components-%s", componentName)

	// Query the appropriate resource type
	var mostRecent ctrlclient.Object
	var err error

	switch resourceType {
	case "Secret":
		mostRecent, err = h.findDefaultSecret(ctx, namespace, annotationKey)
	case "MonitoringConfig":
		mostRecent, err = h.findDefaultMonitoringConfig(ctx, namespace, annotationKey)
	default:
		return "", nil
	}

	if err != nil || mostRecent == nil {
		return "", err
	}

	return mostRecent.GetName(), nil
}

// inferSupportedResourceType derives resource type from field name.
// Returns empty string if field is not supported resource type.
// secretRef -> Secret
// monitoringConfigName -> MonitoringConfig
func inferSupportedResourceType(fieldName string) string {
	resolvableFields := map[string]string{
		"secret":               "Secret",
		"secretName":           "Secret",
		"secretRef":            "Secret",
		"monitoringConfig":     "MonitoringConfig",
		"monitoringConfigName": "MonitoringConfig",
		"monitoringConfigRef":  "MonitoringConfig",
	}

	if resourceType, ok := resolvableFields[fieldName]; ok {
		return resourceType
	}

	return ""
}

// findDefaultSecret finds the most recent Secret with the annotation
func (h *k8sHandler) findDefaultSecret(ctx context.Context, namespace, annotationKey string) (ctrlclient.Object, error) {
	secrets, err := h.kubeConnector.ListSecrets(ctx,
		ctrlclient.InNamespace(namespace),
	)
	if err != nil {
		return nil, err
	}

	// Filter by annotation. Note: Kubernetes API doesn't support annotation selectors,
	// so we must list all Secrets and filter client-side (same limitation as StorageClass).
	// If performance becomes an issue with many Secrets, consider adding labels instead.
	filtered := make([]corev1.Secret, 0)
	for _, secret := range secrets.Items {
		if annotations := secret.GetAnnotations(); annotations != nil {
			if annotations[annotationKey] == "true" {
				filtered = append(filtered, secret)
			}
		}
	}

	if len(filtered) == 0 {
		return nil, nil
	}

	return getMostRecentlyCreated(convertSecretsToObjects(filtered)), nil
}

// findDefaultMonitoringConfig finds the most recent MonitoringConfig with the annotation
func (h *k8sHandler) findDefaultMonitoringConfig(ctx context.Context, namespace, annotationKey string) (ctrlclient.Object, error) {
	configs, err := h.kubeConnector.ListMonitoringConfigsV2(ctx,
		ctrlclient.InNamespace(namespace),
	)
	if err != nil {
		return nil, err
	}

	// Filter by annotation. Note: Kubernetes API doesn't support annotation selectors,
	// so we must list all MonitoringConfigs and filter client-side (same limitation as StorageClass).
	// If performance becomes an issue with many configs, consider adding labels instead.
	filtered := make([]monitoringv1alpha1.MonitoringConfig, 0)
	for _, config := range configs.Items {
		if annotations := config.GetAnnotations(); annotations != nil {
			if annotations[annotationKey] == "true" {
				filtered = append(filtered, config)
			}
		}
	}

	if len(filtered) == 0 {
		return nil, nil
	}

	return getMostRecentlyCreated(convertMonitoringConfigsToObjects(filtered)), nil
}

// getMostRecentlyCreated returns the most recently created resource
func getMostRecentlyCreated(items []ctrlclient.Object) ctrlclient.Object {
	if len(items) == 0 {
		return nil
	}

	mostRecent := items[0]
	for i := 1; i < len(items); i++ {
		if items[i].GetCreationTimestamp().After(mostRecent.GetCreationTimestamp().Time) {
			mostRecent = items[i]
		}
	}

	return mostRecent
}

func convertSecretsToObjects(items []corev1.Secret) []ctrlclient.Object {
	result := make([]ctrlclient.Object, len(items))
	for i := range items {
		result[i] = &items[i]
	}
	return result
}

func convertMonitoringConfigsToObjects(items []monitoringv1alpha1.MonitoringConfig) []ctrlclient.Object {
	result := make([]ctrlclient.Object, len(items))
	for i := range items {
		result[i] = &items[i]
	}
	return result
}
