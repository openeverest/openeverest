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
	"encoding/json"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/kube-openapi/pkg/validation/spec"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/config/crd/bases"
	"github.com/openeverest/openeverest/v2/internal/controller/ssa"
)

// otherFieldOwner stands in for another writer of Plugin conditions.
const otherFieldOwner = "openeverest-test-compatibility"

// pluginTypeConverter types server-side apply on Plugins with their CRD
// schema, as the API server does: status.conditions is a list map keyed by
// type, so each field manager owns its own conditions.
func pluginTypeConverter(t *testing.T) managedfields.TypeConverter { //nolint:ireturn // the fake client takes the interface
	t.Helper()
	raw, err := fs.ReadFile(bases.CRDs, "extensions.openeverest.io_plugins.yaml")
	require.NoError(t, err)
	crd := &apiextensionsv1.CustomResourceDefinition{}
	require.NoError(t, yaml.Unmarshal(raw, crd))
	models := make(map[string]*spec.Schema)
	for _, v := range crd.Spec.Versions {
		js, err := json.Marshal(v.Schema.OpenAPIV3Schema)
		require.NoError(t, err)
		s := &spec.Schema{}
		require.NoError(t, json.Unmarshal(js, s))
		s.AddExtension("x-kubernetes-group-version-kind", []any{
			map[string]any{"group": crd.Spec.Group, "version": v.Name, "kind": crd.Spec.Names.Kind},
		})
		models[crd.Spec.Group+"."+v.Name+"."+crd.Spec.Names.Kind] = s
	}
	tc, err := managedfields.NewTypeConverter(models, false)
	require.NoError(t, err)
	return tc
}

func newPluginTestClient(t *testing.T, funcs interceptor.Funcs, objs ...client.Object) client.Client { //nolint:ireturn // the fake is used through the client interface
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, pluginv1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithTypeConverters(pluginTypeConverter(t), managedfields.NewDeducedTypeConverter()).
		WithObjects(objs...).
		WithStatusSubresource(&pluginv1alpha1.Plugin{}).
		WithInterceptorFuncs(funcs).
		WithReturnManagedFields().
		Build()
}

func testPlugin(finalizers ...string) *pluginv1alpha1.Plugin {
	return &pluginv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: "metrics", Finalizers: finalizers},
		Spec: pluginv1alpha1.PluginSpec{
			DisplayName: "Metrics",
			Enabled:     true,
			Frontend:    &pluginv1alpha1.PluginFrontend{BundlePath: "/bundle.js"},
		},
	}
}

// applyOtherCondition writes a condition on the Plugin as another writer
// would, under its own field manager.
func applyOtherCondition(ctx context.Context, c client.Client, plugin *pluginv1alpha1.Plugin) error {
	cond := ssa.NewCondition(plugin, "Compatible", metav1.ConditionFalse, "CoreTooOld", "needs a newer core")
	return ssa.ApplyCondition(ctx, c, plugin, nil, otherFieldOwner, cond)
}

// statusOwners returns the field managers that own fields of obj's status.
func statusOwners(obj client.Object) []string {
	var owners []string
	for _, mf := range obj.GetManagedFields() {
		if mf.FieldsV1 != nil && strings.Contains(mf.FieldsV1.GetRawString(), `"f:status"`) {
			owners = append(owners, mf.Manager)
		}
	}
	slices.Sort(owners)
	return owners
}

func TestPluginReconcile_Conditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		mutate          func(*pluginv1alpha1.Plugin)
		wantAvailable   metav1.ConditionStatus
		wantAvailReason string
		wantReady       metav1.ConditionStatus
		wantReadyReason string
	}{
		{
			name:            "enabled and configured",
			mutate:          func(*pluginv1alpha1.Plugin) {},
			wantAvailable:   metav1.ConditionTrue,
			wantAvailReason: reasonReconciling,
			wantReady:       metav1.ConditionTrue,
			wantReadyReason: reasonBackendReady,
		},
		{
			name:            "no display name",
			mutate:          func(p *pluginv1alpha1.Plugin) { p.Spec.DisplayName = "" },
			wantAvailable:   metav1.ConditionFalse,
			wantAvailReason: reasonInvalidSpec,
			wantReady:       metav1.ConditionTrue,
			wantReadyReason: reasonBackendReady,
		},
		{
			name:            "disabled",
			mutate:          func(p *pluginv1alpha1.Plugin) { p.Spec.Enabled = false },
			wantAvailable:   metav1.ConditionTrue,
			wantAvailReason: reasonReconciling,
			wantReady:       metav1.ConditionFalse,
			wantReadyReason: reasonDisabled,
		},
		{
			name:            "neither frontend nor backend",
			mutate:          func(p *pluginv1alpha1.Plugin) { p.Spec.Frontend = nil },
			wantAvailable:   metav1.ConditionTrue,
			wantAvailReason: reasonReconciling,
			wantReady:       metav1.ConditionFalse,
			wantReadyReason: reasonNoBackend,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plugin := testPlugin()
			tt.mutate(plugin)
			c := newPluginTestClient(t, interceptor.Funcs{}, plugin)
			r := &PluginReconciler{Client: c, Scheme: c.Scheme()}
			req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(plugin)}

			_, err := r.Reconcile(t.Context(), req)
			require.NoError(t, err)

			got := &pluginv1alpha1.Plugin{}
			require.NoError(t, c.Get(t.Context(), req.NamespacedName, got))
			assert.Contains(t, got.Finalizers, pluginFinalizer)
			avail := meta.FindStatusCondition(got.Status.Conditions, ConditionTypeAvailable)
			require.NotNil(t, avail)
			assert.Equal(t, tt.wantAvailable, avail.Status)
			assert.Equal(t, tt.wantAvailReason, avail.Reason)
			assert.Equal(t, got.Generation, avail.ObservedGeneration)
			ready := meta.FindStatusCondition(got.Status.Conditions, ConditionTypeReady)
			require.NotNil(t, ready)
			assert.Equal(t, tt.wantReady, ready.Status)
			assert.Equal(t, tt.wantReadyReason, ready.Reason)
			assert.Equal(t, got.Generation, ready.ObservedGeneration)
			assert.Equal(t, []string{pluginFieldOwner}, statusOwners(got))
		})
	}
}

// TestPluginReconcile_KeepsOtherWritersConditions checks that the status
// write leaves the conditions of other writers alone, including one written
// between the controller's read and its write, which a merge patch built
// from that read would erase.
func TestPluginReconcile_KeepsOtherWritersConditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setAfterRead bool
	}{
		{name: "set before the read"},
		{name: "set after the read", setAfterRead: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plugin := testPlugin(pluginFinalizer)
			var funcs interceptor.Funcs
			if tt.setAfterRead {
				written := false
				funcs.Get = func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if err := c.Get(ctx, key, obj, opts...); err != nil {
						return err
					}
					if p, ok := obj.(*pluginv1alpha1.Plugin); ok && !written {
						written = true
						return applyOtherCondition(ctx, c, p)
					}
					return nil
				}
			}
			c := newPluginTestClient(t, funcs, plugin)
			if !tt.setAfterRead {
				require.NoError(t, applyOtherCondition(t.Context(), c, plugin))
			}
			r := &PluginReconciler{Client: c, Scheme: c.Scheme()}
			req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(plugin)}

			_, err := r.Reconcile(t.Context(), req)
			require.NoError(t, err)

			got := &pluginv1alpha1.Plugin{}
			require.NoError(t, c.Get(t.Context(), req.NamespacedName, got))
			other := meta.FindStatusCondition(got.Status.Conditions, "Compatible")
			require.NotNil(t, other, "the other writer's condition is kept")
			assert.Equal(t, metav1.ConditionFalse, other.Status)
			assert.Equal(t, "CoreTooOld", other.Reason)
			assert.Equal(t, "needs a newer core", other.Message)
			assert.True(t, meta.IsStatusConditionTrue(got.Status.Conditions, ConditionTypeAvailable))
			assert.True(t, meta.IsStatusConditionTrue(got.Status.Conditions, ConditionTypeReady))
			assert.Equal(t, []string{pluginFieldOwner, otherFieldOwner}, statusOwners(got))
		})
	}
}

func TestPluginReconcile_KeepsLastTransitionTime(t *testing.T) {
	t.Parallel()
	since := metav1.NewTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	plugin := testPlugin(pluginFinalizer)
	plugin.Status.Conditions = []metav1.Condition{
		{Type: ConditionTypeAvailable, Status: metav1.ConditionTrue, Reason: reasonReconciling, LastTransitionTime: since},
		{Type: ConditionTypeReady, Status: metav1.ConditionFalse, Reason: reasonDisabled, LastTransitionTime: since},
	}
	c := newPluginTestClient(t, interceptor.Funcs{}, plugin)
	r := &PluginReconciler{Client: c, Scheme: c.Scheme()}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(plugin)}

	_, err := r.Reconcile(t.Context(), req)
	require.NoError(t, err)

	got := &pluginv1alpha1.Plugin{}
	require.NoError(t, c.Get(t.Context(), req.NamespacedName, got))
	avail := meta.FindStatusCondition(got.Status.Conditions, ConditionTypeAvailable)
	require.NotNil(t, avail)
	assert.True(t, since.Equal(&avail.LastTransitionTime), "unchanged status keeps lastTransitionTime, got %s", avail.LastTransitionTime)
	ready := meta.FindStatusCondition(got.Status.Conditions, ConditionTypeReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionTrue, ready.Status)
	assert.True(t, ready.LastTransitionTime.After(since.Time), "a status change moves lastTransitionTime, got %s", ready.LastTransitionTime)
}

func TestPluginReconcile_StatusWriteError(t *testing.T) {
	t.Parallel()
	gr := schema.GroupResource{Group: pluginv1alpha1.GroupVersion.Group, Resource: "plugins"}

	tests := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{name: "plugin deleted after the read", err: apierrors.NewNotFound(gr, "metrics")},
		{name: "other error", err: apierrors.NewServiceUnavailable("etcd is down"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plugin := testPlugin(pluginFinalizer)
			funcs := interceptor.Funcs{
				SubResourceApply: func(context.Context, client.Client, string, runtime.ApplyConfiguration, ...client.SubResourceApplyOption) error {
					return tt.err
				},
			}
			c := newPluginTestClient(t, funcs, plugin)
			r := &PluginReconciler{Client: c, Scheme: c.Scheme()}

			_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(plugin)})
			if tt.wantErr {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
