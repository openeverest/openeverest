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
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

const otherFinalizer = "example.com/keep"

func reconcilePlugin(t *testing.T, plugin *pluginv1alpha1.Plugin) client.Client { //nolint:ireturn // the fake is used through the client interface
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, pluginv1alpha1.AddToScheme(scheme))
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(plugin).
		WithStatusSubresource(&pluginv1alpha1.Plugin{}).
		Build()

	r := &PluginReconciler{Client: c, Scheme: scheme}
	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: plugin.Name}})
	require.NoError(t, err)
	return c
}

func newPlugin(finalizers ...string) *pluginv1alpha1.Plugin {
	return &pluginv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: "plugin-hub", Finalizers: finalizers},
		Spec: pluginv1alpha1.PluginSpec{
			DisplayName: "Plugin Hub",
			Enabled:     true,
			Frontend:    &pluginv1alpha1.PluginFrontend{},
		},
	}
}

func TestPluginReconcileDoesNotAddFinalizer(t *testing.T) {
	t.Parallel()
	c := reconcilePlugin(t, newPlugin())

	got := &pluginv1alpha1.Plugin{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "plugin-hub"}, got))
	require.Empty(t, got.Finalizers)
	require.True(t, isConditionTrue(got.Status.Conditions, ConditionTypeReady))
}

func TestPluginReconcileRemovesLegacyFinalizer(t *testing.T) {
	t.Parallel()
	c := reconcilePlugin(t, newPlugin(legacyPluginFinalizer, otherFinalizer))

	got := &pluginv1alpha1.Plugin{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "plugin-hub"}, got))
	require.Equal(t, []string{otherFinalizer}, got.Finalizers)
	require.True(t, isConditionTrue(got.Status.Conditions, ConditionTypeReady))
}

func TestPluginReconcileReleasesDeletedPlugin(t *testing.T) {
	t.Parallel()
	plugin := newPlugin(legacyPluginFinalizer)
	now := metav1.Now()
	plugin.DeletionTimestamp = &now
	c := reconcilePlugin(t, plugin)

	err := c.Get(context.Background(), client.ObjectKey{Name: "plugin-hub"}, &pluginv1alpha1.Plugin{})
	require.True(t, apierrors.IsNotFound(err), "expected Plugin to be gone, got %v", err)
}
