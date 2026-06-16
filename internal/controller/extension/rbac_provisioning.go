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

package extension

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

const (
	// pluginRolePrefix is prepended to the plugin name to form the Role and
	// ClusterRole names.
	pluginRolePrefix = "everest-plugin-"

	// managedByLabel marks resources created by the plugin system.
	managedByLabel = "app.kubernetes.io/managed-by"

	// pluginNameLabel records which Plugin CR a generated RBAC object belongs to.
	pluginNameLabel = "core.openeverest.io/plugin-name"

	// installedExtensionLabel records the owning InstalledExtension.
	installedExtensionLabel = "core.openeverest.io/installed-extension"

	// defaultServiceAccountNamespace is where the plugin's ServiceAccount lives
	// when no in-cluster backend service is declared.
	defaultServiceAccountNamespace = "everest-system"
)

func roleName(pluginName string) string { return pluginRolePrefix + pluginName }

// ensureRBAC dispatches to the cluster-scope or namespaces-scope provisioner
// based on spec.plugin.scope.
func (r *InstalledExtensionReconciler) ensureRBAC(
	ctx context.Context,
	ie *corev1alpha1.InstalledExtension,
	plugin *corev1alpha1.Plugin,
) error {
	if len(plugin.Spec.KubePermissions) == 0 {
		// Nothing to provision; remove any stale objects from prior generations.
		return r.cleanupRBAC(ctx, ie)
	}
	if violations := validateKubePermissions(plugin.Spec.KubePermissions); len(violations) > 0 {
		return fmt.Errorf("kubePermissions denied: %v", violations)
	}

	if ie.Spec.Plugin.Scope == corev1alpha1.PluginInstallScopeCluster {
		// Cleanup any per-namespace Roles from a previous scope=Namespaces
		// configuration before installing the ClusterRole.
		if err := r.cleanupNamespacedRBAC(ctx, ie); err != nil {
			return err
		}
		return r.ensureClusterRBAC(ctx, ie, plugin)
	}

	// scope=Namespaces — cleanup any ClusterRole/CRB from a previous
	// scope=Cluster configuration first.
	if err := r.cleanupClusterRBAC(ctx, ie); err != nil {
		return err
	}
	return r.ensureNamespacedRBAC(ctx, ie, plugin)
}

// ensureClusterRBAC creates or updates a ClusterRole + ClusterRoleBinding.
func (r *InstalledExtensionReconciler) ensureClusterRBAC(
	ctx context.Context,
	ie *corev1alpha1.InstalledExtension,
	plugin *corev1alpha1.Plugin,
) error {
	logger := log.FromContext(ctx)
	name := roleName(plugin.Name)
	rules := policyRules(plugin.Spec.KubePermissions)
	lbls := rbacLabels(ie, plugin)

	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: lbls},
		Rules:      rules,
	}
	if err := controllerutil.SetControllerReference(ie, cr, r.Scheme); err != nil {
		return fmt.Errorf("set owner on ClusterRole: %w", err)
	}
	existingCR := &rbacv1.ClusterRole{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: name}, existingCR)
	switch {
	case apierrors.IsNotFound(err):
		logger.Info("Creating plugin ClusterRole", "clusterrole", name)
		if err := r.Client.Create(ctx, cr); err != nil {
			return err
		}
	case err != nil:
		return fmt.Errorf("get ClusterRole: %w", err)
	default:
		existingCR.Rules = rules
		existingCR.Labels = lbls
		logger.Info("Updating plugin ClusterRole", "clusterrole", name)
		if err := r.Client.Update(ctx, existingCR); err != nil {
			return err
		}
	}

	saName := pluginRolePrefix + plugin.Name
	saNamespace := pluginServiceAccountNamespace(plugin)
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: lbls},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      saName,
			Namespace: saNamespace,
		}},
	}
	if err := controllerutil.SetControllerReference(ie, crb, r.Scheme); err != nil {
		return fmt.Errorf("set owner on ClusterRoleBinding: %w", err)
	}
	existingCRB := &rbacv1.ClusterRoleBinding{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: name}, existingCRB)
	switch {
	case apierrors.IsNotFound(err):
		logger.Info("Creating plugin ClusterRoleBinding", "crb", name)
		return r.Client.Create(ctx, crb)
	case err != nil:
		return fmt.Errorf("get ClusterRoleBinding: %w", err)
	default:
		existingCRB.RoleRef = crb.RoleRef
		existingCRB.Subjects = crb.Subjects
		existingCRB.Labels = lbls
		logger.Info("Updating plugin ClusterRoleBinding", "crb", name)
		return r.Client.Update(ctx, existingCRB)
	}
}

// ensureNamespacedRBAC creates per-namespace Role + RoleBinding entries.
// It also garbage-collects entries for namespaces no longer listed.
func (r *InstalledExtensionReconciler) ensureNamespacedRBAC(
	ctx context.Context,
	ie *corev1alpha1.InstalledExtension,
	plugin *corev1alpha1.Plugin,
) error {
	logger := log.FromContext(ctx)
	name := roleName(plugin.Name)
	rules := policyRules(plugin.Spec.KubePermissions)
	lbls := rbacLabels(ie, plugin)
	saName := pluginRolePrefix + plugin.Name
	saNamespace := pluginServiceAccountNamespace(plugin)

	desired := map[string]struct{}{}
	for _, nsCfg := range ie.Spec.Plugin.Namespaces {
		desired[nsCfg.Name] = struct{}{}

		role := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: nsCfg.Name, Labels: lbls},
			Rules:      rules,
		}
		if err := controllerutil.SetControllerReference(ie, role, r.Scheme); err != nil {
			return fmt.Errorf("set owner on Role: %w", err)
		}
		existingRole := &rbacv1.Role{}
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: nsCfg.Name, Name: name}, existingRole)
		switch {
		case apierrors.IsNotFound(err):
			logger.Info("Creating plugin Role", "role", name, "namespace", nsCfg.Name)
			if err := r.Client.Create(ctx, role); err != nil {
				return err
			}
		case err != nil:
			return fmt.Errorf("get Role: %w", err)
		default:
			existingRole.Rules = rules
			existingRole.Labels = lbls
			logger.Info("Updating plugin Role", "role", name, "namespace", nsCfg.Name)
			if err := r.Client.Update(ctx, existingRole); err != nil {
				return err
			}
		}

		rb := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: nsCfg.Name, Labels: lbls},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     name,
			},
			Subjects: []rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      saName,
				Namespace: saNamespace,
			}},
		}
		if err := controllerutil.SetControllerReference(ie, rb, r.Scheme); err != nil {
			return fmt.Errorf("set owner on RoleBinding: %w", err)
		}
		existingRB := &rbacv1.RoleBinding{}
		err = r.Client.Get(ctx, types.NamespacedName{Namespace: nsCfg.Name, Name: name}, existingRB)
		switch {
		case apierrors.IsNotFound(err):
			logger.Info("Creating plugin RoleBinding", "rolebinding", name, "namespace", nsCfg.Name)
			if err := r.Client.Create(ctx, rb); err != nil {
				return err
			}
		case err != nil:
			return fmt.Errorf("get RoleBinding: %w", err)
		default:
			existingRB.RoleRef = rb.RoleRef
			existingRB.Subjects = rb.Subjects
			existingRB.Labels = lbls
			logger.Info("Updating plugin RoleBinding", "rolebinding", name, "namespace", nsCfg.Name)
			if err := r.Client.Update(ctx, existingRB); err != nil {
				return err
			}
		}
	}

	return r.gcNamespacedRBAC(ctx, ie, plugin.Name, desired)
}

// cleanupRBAC removes any RBAC objects owned by ie, regardless of scope.
func (r *InstalledExtensionReconciler) cleanupRBAC(ctx context.Context, ie *corev1alpha1.InstalledExtension) error {
	if err := r.cleanupClusterRBAC(ctx, ie); err != nil {
		return err
	}
	return r.cleanupNamespacedRBAC(ctx, ie)
}

func (r *InstalledExtensionReconciler) cleanupClusterRBAC(ctx context.Context, ie *corev1alpha1.InstalledExtension) error {
	if ie.Spec.Plugin == nil {
		return nil
	}
	name := roleName(ie.Spec.Plugin.PluginCRName)
	if err := deleteIgnoreNotFound(ctx, r.Client, &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: name}}); err != nil {
		return fmt.Errorf("delete ClusterRoleBinding: %w", err)
	}
	if err := deleteIgnoreNotFound(ctx, r.Client, &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: name}}); err != nil {
		return fmt.Errorf("delete ClusterRole: %w", err)
	}
	return nil
}

func (r *InstalledExtensionReconciler) cleanupNamespacedRBAC(ctx context.Context, ie *corev1alpha1.InstalledExtension) error {
	if ie.Spec.Plugin == nil {
		return nil
	}
	return r.gcNamespacedRBAC(ctx, ie, ie.Spec.Plugin.PluginCRName, nil)
}

// gcNamespacedRBAC lists all Role/RoleBinding labeled with this
// InstalledExtension and deletes ones whose namespace is not in `keep`.
func (r *InstalledExtensionReconciler) gcNamespacedRBAC(
	ctx context.Context,
	ie *corev1alpha1.InstalledExtension,
	pluginName string,
	keep map[string]struct{},
) error {
	selector := client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(labels.Set{
		installedExtensionLabel: ie.Name,
	})}

	roles := &rbacv1.RoleList{}
	if err := r.Client.List(ctx, roles, selector); err != nil {
		return fmt.Errorf("list Roles: %w", err)
	}
	for i := range roles.Items {
		role := &roles.Items[i]
		if role.Name != roleName(pluginName) {
			continue
		}
		if _, want := keep[role.Namespace]; want {
			continue
		}
		if err := deleteIgnoreNotFound(ctx, r.Client, role); err != nil {
			return fmt.Errorf("delete Role: %w", err)
		}
	}
	rbs := &rbacv1.RoleBindingList{}
	if err := r.Client.List(ctx, rbs, selector); err != nil {
		return fmt.Errorf("list RoleBindings: %w", err)
	}
	for i := range rbs.Items {
		rb := &rbs.Items[i]
		if rb.Name != roleName(pluginName) {
			continue
		}
		if _, want := keep[rb.Namespace]; want {
			continue
		}
		if err := deleteIgnoreNotFound(ctx, r.Client, rb); err != nil {
			return fmt.Errorf("delete RoleBinding: %w", err)
		}
	}
	return nil
}

func deleteIgnoreNotFound(ctx context.Context, c client.Client, obj client.Object) error {
	err := c.Delete(ctx, obj)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func policyRules(kps []corev1alpha1.KubePermissionRule) []rbacv1.PolicyRule {
	rules := make([]rbacv1.PolicyRule, 0, len(kps))
	for _, kp := range kps {
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: kp.APIGroups,
			Resources: kp.Resources,
			Verbs:     kp.Verbs,
		})
	}
	return rules
}

func rbacLabels(ie *corev1alpha1.InstalledExtension, plugin *corev1alpha1.Plugin) map[string]string {
	return map[string]string{
		managedByLabel:          pluginRolePrefix + "controller",
		pluginNameLabel:         plugin.Name,
		installedExtensionLabel: ie.Name,
	}
}

// pluginServiceAccountNamespace returns the namespace where the plugin's
// ServiceAccount lives. For in-cluster backends, this is the backend service
// namespace; otherwise defaults to everest-system.
func pluginServiceAccountNamespace(plugin *corev1alpha1.Plugin) string {
	if plugin.Spec.Backend != nil && plugin.Spec.Backend.ServiceRef != nil {
		return plugin.Spec.Backend.ServiceRef.Namespace
	}
	return defaultServiceAccountNamespace
}
