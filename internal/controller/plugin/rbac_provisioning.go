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

package plugin

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/plugin/v1alpha1"
)

const (
	// pluginRolePrefix is prepended to the plugin name to form the Role name.
	pluginRolePrefix = "everest-plugin-"

	// managedByLabel marks resources created by the plugin system.
	managedByLabel = "app.kubernetes.io/managed-by"

	// pluginNameLabel records which plugin owns the resource.
	pluginNameLabel = "core.openeverest.io/plugin-name"
)

// roleName returns the deterministic Role name for a plugin.
func roleName(pluginName string) string {
	return pluginRolePrefix + pluginName
}

// ensurePluginRole creates or updates a namespace-scoped Role containing
// the kubePermissions declared by the plugin. The Role is owned by the
// PluginInstallation so it is garbage-collected on deletion.
func (r *PluginInstallationReconciler) ensurePluginRole(
	ctx context.Context,
	pi *pluginv1alpha1.PluginInstallation,
	plugin *pluginv1alpha1.Plugin,
) error {
	logger := log.FromContext(ctx)

	if len(plugin.Spec.KubePermissions) == 0 {
		// No kubePermissions declared — clean up any stale Role.
		return r.deletePluginRole(ctx, pi)
	}

	rules := make([]rbacv1.PolicyRule, 0, len(plugin.Spec.KubePermissions))
	for _, kp := range plugin.Spec.KubePermissions {
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: kp.APIGroups,
			Resources: kp.Resources,
			Verbs:     kp.Verbs,
		})
	}

	name := roleName(plugin.Name)
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pi.Namespace,
			Labels: map[string]string{
				managedByLabel:  pluginRolePrefix + "controller",
				pluginNameLabel: plugin.Name,
			},
		},
		Rules: rules,
	}

	// Set the PluginInstallation as owner so k8s GC cleans up on deletion.
	if err := controllerutil.SetOwnerReference(pi, role, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on Role: %w", err)
	}

	existing := &rbacv1.Role{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: pi.Namespace, Name: name}, existing)
	if apierrors.IsNotFound(err) {
		logger.Info("Creating plugin Role", "role", name, "namespace", pi.Namespace)
		return r.Client.Create(ctx, role)
	}
	if err != nil {
		return fmt.Errorf("failed to get existing Role: %w", err)
	}

	// Update rules if changed.
	existing.Rules = rules
	existing.Labels = role.Labels
	logger.Info("Updating plugin Role", "role", name, "namespace", pi.Namespace)
	return r.Client.Update(ctx, existing)
}

// ensurePluginRoleBinding creates or updates a RoleBinding that binds the
// plugin's Role to its ServiceAccount. The RoleBinding is owned by the
// PluginInstallation for garbage collection.
func (r *PluginInstallationReconciler) ensurePluginRoleBinding(
	ctx context.Context,
	pi *pluginv1alpha1.PluginInstallation,
	plugin *pluginv1alpha1.Plugin,
) error {
	logger := log.FromContext(ctx)

	if len(plugin.Spec.KubePermissions) == 0 {
		return r.deletePluginRoleBinding(ctx, pi)
	}

	name := roleName(plugin.Name)
	saName := pluginRolePrefix + plugin.Name // ServiceAccount name convention
	saNamespace := pluginServiceAccountNamespace(plugin)

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pi.Namespace,
			Labels: map[string]string{
				managedByLabel:  pluginRolePrefix + "controller",
				pluginNameLabel: plugin.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      saName,
				Namespace: saNamespace,
			},
		},
	}

	if err := controllerutil.SetOwnerReference(pi, rb, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on RoleBinding: %w", err)
	}

	existing := &rbacv1.RoleBinding{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: pi.Namespace, Name: name}, existing)
	if apierrors.IsNotFound(err) {
		logger.Info("Creating plugin RoleBinding", "rolebinding", name, "namespace", pi.Namespace)
		return r.Client.Create(ctx, rb)
	}
	if err != nil {
		return fmt.Errorf("failed to get existing RoleBinding: %w", err)
	}

	existing.RoleRef = rb.RoleRef
	existing.Subjects = rb.Subjects
	existing.Labels = rb.Labels
	logger.Info("Updating plugin RoleBinding", "rolebinding", name, "namespace", pi.Namespace)
	return r.Client.Update(ctx, existing)
}

// deletePluginRole removes the plugin Role if it exists.
func (r *PluginInstallationReconciler) deletePluginRole(
	ctx context.Context,
	pi *pluginv1alpha1.PluginInstallation,
) error {
	name := roleName(pi.Spec.PluginName)
	role := &rbacv1.Role{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: pi.Namespace, Name: name}, role)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.Client.Delete(ctx, role)
}

// deletePluginRoleBinding removes the plugin RoleBinding if it exists.
func (r *PluginInstallationReconciler) deletePluginRoleBinding(
	ctx context.Context,
	pi *pluginv1alpha1.PluginInstallation,
) error {
	name := roleName(pi.Spec.PluginName)
	rb := &rbacv1.RoleBinding{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: pi.Namespace, Name: name}, rb)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.Client.Delete(ctx, rb)
}

// pluginServiceAccountNamespace returns the namespace where the plugin's
// ServiceAccount lives. For in-cluster backends, this is the backend service
// namespace; otherwise defaults to "everest-system".
func pluginServiceAccountNamespace(plugin *pluginv1alpha1.Plugin) string {
	if plugin.Spec.Backend != nil && plugin.Spec.Backend.ServiceRef != nil {
		return plugin.Spec.Backend.ServiceRef.Namespace
	}
	return "everest-system"
}
