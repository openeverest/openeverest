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

package controller

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

const applyTestNamespace = "db"

func applyTestContext(t *testing.T, funcs interceptor.Funcs) *Context {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	cl := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(funcs).Build()
	in := &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "inst", Namespace: applyTestNamespace, UID: "inst-uid"},
	}
	return NewContext(t.Context(), cl, in, "test")
}

func configMap(c *Context, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: c.ObjectMeta("cm"), Data: data}
}

func getConfigMap(t *testing.T, c *Context) *corev1.ConfigMap {
	t.Helper()
	got := &corev1.ConfigMap{}
	require.NoError(t, c.Get(got, "cm"))
	return got
}

func TestApplyCreatesWithControllerReference(t *testing.T) {
	t.Parallel()
	c := applyTestContext(t, interceptor.Funcs{})

	require.NoError(t, c.Apply(configMap(c, map[string]string{"a": "1"})))

	got := getConfigMap(t, c)
	assert.Equal(t, map[string]string{"a": "1"}, got.Data)
	require.Len(t, got.OwnerReferences, 1)
	assert.Equal(t, "inst", got.OwnerReferences[0].Name)
	assert.True(t, *got.OwnerReferences[0].Controller)
}

func TestApplyPreservesFieldsOwnedByAnotherManager(t *testing.T) {
	t.Parallel()
	c := applyTestContext(t, interceptor.Funcs{})

	require.NoError(t, c.Apply(configMap(c, map[string]string{"a": "1"})))
	require.NoError(t, c.Client().Apply(t.Context(),
		corev1ac.ConfigMap("cm", applyTestNamespace).WithData(map[string]string{"operator": "injected"}),
		client.FieldOwner("engine-operator"),
	))

	require.NoError(t, c.Apply(configMap(c, map[string]string{"a": "2"})))

	assert.Equal(t, map[string]string{"a": "2", "operator": "injected"}, getConfigMap(t, c).Data)
}

func TestApplyRemovesFieldsNoLongerSet(t *testing.T) {
	t.Parallel()
	c := applyTestContext(t, interceptor.Funcs{})

	require.NoError(t, c.Apply(configMap(c, map[string]string{"a": "1", "b": "2"})))
	require.NoError(t, c.Apply(configMap(c, map[string]string{"a": "1"})))

	assert.Equal(t, map[string]string{"a": "1"}, getConfigMap(t, c).Data)
}

func TestApplyRejectsObjectFromGet(t *testing.T) {
	t.Parallel()
	c := applyTestContext(t, interceptor.Funcs{})
	require.NoError(t, c.Apply(configMap(c, map[string]string{"a": "1"})))

	existing := getConfigMap(t, c)
	existing.Data["a"] = "changed"

	require.ErrorContains(t, c.Apply(existing), "freshly built object")
	assert.Equal(t, "1", getConfigMap(t, c).Data["a"])
}

func TestApplyBodyOmitsStatusAndNulls(t *testing.T) {
	t.Parallel()
	var body map[string]any
	c := applyTestContext(t, interceptor.Funcs{
		Apply: func(ctx context.Context, cl client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
			data, err := json.Marshal(obj)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(data, &body))
			return cl.Apply(ctx, obj, opts...)
		},
	})

	// A StatefulSet serializes an empty status and a null spec.selector.
	require.NoError(t, c.Apply(&appsv1.StatefulSet{
		ObjectMeta: c.ObjectMeta("sts"),
		Spec:       appsv1.StatefulSetSpec{ServiceName: "svc"},
	}))

	require.NotNil(t, body)
	assert.NotContains(t, body, "status")
	spec, ok := body["spec"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, spec, "selector")
}

func TestPruneNulls(t *testing.T) {
	t.Parallel()

	got := map[string]any{
		"keep":  "value",
		"drop":  nil,
		"empty": make(map[string]any),
		"nested": map[string]any{
			"drop": nil,
			"keep": false,
		},
		"list": []any{
			map[string]any{"drop": nil, "keep": int64(0)},
			"item",
		},
	}
	pruneNulls(got)

	assert.Equal(
		t,
		map[string]any{
			"keep":   "value",
			"empty":  make(map[string]any),
			"nested": map[string]any{"keep": false},
			"list": []any{
				map[string]any{"keep": int64(0)},
				"item",
			},
		},
		got,
	)
}
