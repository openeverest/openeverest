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
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/plugintoken"
)

const (
	// ConditionTypeTokenIssued is True when the plugin's service token is
	// in its token Secret.
	ConditionTypeTokenIssued = "TokenIssued"

	reasonTokenIssued           = "TokenIssued"
	reasonSigningKeyUnavailable = "SigningKeyUnavailable"
	reasonSecretConflict        = "SecretConflict"

	// pluginTokenLabel marks the token Secrets the controller manages with
	// the name of their plugin.
	pluginTokenLabel = "extensions.openeverest.io/plugin-token"
)

// TokenSecretName is the name of the Secret holding a plugin's service token,
// in the namespace of its backend Service.
func TokenSecretName(pluginName string) string {
	return pluginName + "-token"
}

// wantsToken reports whether the plugin gets a service token: an enabled
// in-cluster backend that declares the permissions it needs.
func wantsToken(plugin *pluginv1alpha1.Plugin) bool {
	return plugin.Spec.Enabled &&
		plugin.Spec.Backend != nil && plugin.Spec.Backend.ServiceRef != nil &&
		len(plugin.Spec.Permissions) > 0
}

// reconcileToken keeps the plugin's service token Secret current: it mints a
// token when there is none, when the Plugin's permissions changed, and when
// a third of its lifetime is left. It returns how long until the token must be
// renewed, or zero when no renewal is scheduled.
func (r *PluginReconciler) reconcileToken(ctx context.Context, plugin *pluginv1alpha1.Plugin) (time.Duration, error) {
	if !wantsToken(plugin) {
		meta.RemoveStatusCondition(&plugin.Status.Conditions, ConditionTypeTokenIssued)
		return 0, r.deleteTokenSecrets(ctx, plugin, "")
	}
	if r.Tokens == nil {
		r.setTokenCondition(plugin, metav1.ConditionFalse, reasonSigningKeyUnavailable,
			"The controller cannot sign plugin service tokens: the JWT signing key is not mounted",
		)
		return 0, nil
	}

	ns := plugin.Spec.Backend.ServiceRef.Namespace
	// A token left in another namespace after the backend moved stops being
	// valid only when it expires, so remove it.
	if err := r.deleteTokenSecrets(ctx, plugin, ns); err != nil {
		return 0, err
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: ns, Name: TokenSecretName(plugin.GetName())}
	err := r.Get(ctx, key, secret)
	switch {
	case apierrors.IsNotFound(err):
		secret = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name}}
	case err != nil:
		return 0, fmt.Errorf("get token secret: %w", err)
	case !metav1.IsControlledBy(secret, plugin):
		r.setTokenCondition(plugin, metav1.ConditionFalse, reasonSecretConflict,
			fmt.Sprintf("Secret %s exists and is not managed by this plugin", key),
		)
		return 0, nil
	}

	now := r.now()
	renewAt, current := tokenRenewal(secret, plugin)
	if current && now.Before(renewAt) {
		r.setTokenCondition(plugin, metav1.ConditionTrue, reasonTokenIssued, "Service token is in Secret "+key.String())
		return renewAt.Sub(now), nil
	}

	token, err := r.Tokens.Mint(plugin, now, plugintoken.DefaultTTL)
	if err != nil {
		return 0, fmt.Errorf("mint service token: %w", err)
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if secret.Labels == nil {
			secret.Labels = make(map[string]string)
		}
		secret.Labels[pluginTokenLabel] = plugin.GetName()
		secret.Type = corev1.SecretTypeOpaque
		secret.Data = map[string][]byte{plugintoken.SecretKey: []byte(token)}
		return controllerutil.SetControllerReference(plugin, secret, r.Scheme)
	}); err != nil {
		return 0, fmt.Errorf("write token secret: %w", err)
	}
	r.setTokenCondition(plugin, metav1.ConditionTrue, reasonTokenIssued, "Service token is in Secret "+key.String())
	return tokenRenewIn(plugintoken.DefaultTTL), nil
}

// tokenRenewal returns when the token in secret must be renewed, and whether
// it still matches plugin.
func tokenRenewal(secret *corev1.Secret, plugin *pluginv1alpha1.Plugin) (time.Time, bool) {
	raw := secret.Data[plugintoken.SecretKey]
	if len(raw) == 0 {
		return time.Time{}, false
	}
	claims := &plugintoken.Claims{}
	// The controller signed the token itself; it only reads back the claims.
	if _, _, err := jwt.NewParser().ParseUnverified(string(raw), claims); err != nil {
		return time.Time{}, false
	}
	if claims.PluginUID != string(plugin.GetUID()) || claims.IssuedAt == nil || claims.ExpiresAt == nil ||
		!slices.Equal(claims.Permissions, plugin.Spec.Permissions) {
		return time.Time{}, false
	}
	ttl := claims.ExpiresAt.Sub(claims.IssuedAt.Time)
	return claims.IssuedAt.Add(tokenRenewIn(ttl)), true
}

// tokenRenewIn is how long after issuing a token of the given lifetime it is
// renewed, leaving the plugin a third of the lifetime to pick up the new one.
func tokenRenewIn(ttl time.Duration) time.Duration {
	return ttl * 2 / 3
}

// deleteTokenSecrets deletes the plugin's token Secrets outside keepNamespace.
func (r *PluginReconciler) deleteTokenSecrets(ctx context.Context, plugin *pluginv1alpha1.Plugin, keepNamespace string) error {
	secrets := &corev1.SecretList{}
	if err := r.List(ctx, secrets, client.MatchingLabels{pluginTokenLabel: plugin.GetName()}); err != nil {
		return fmt.Errorf("list token secrets: %w", err)
	}
	for i := range secrets.Items {
		s := &secrets.Items[i]
		if s.GetNamespace() == keepNamespace || !metav1.IsControlledBy(s, plugin) {
			continue
		}
		if err := r.Delete(ctx, s); client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("delete token secret %s/%s: %w", s.GetNamespace(), s.GetName(), err)
		}
	}
	return nil
}

func (r *PluginReconciler) setTokenCondition(plugin *pluginv1alpha1.Plugin, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&plugin.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeTokenIssued,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: plugin.Generation,
	})
}

func (r *PluginReconciler) now() time.Time {
	if r.Clock != nil {
		return r.Clock()
	}
	return time.Now()
}
