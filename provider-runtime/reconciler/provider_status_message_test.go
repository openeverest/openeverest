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

package reconciler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// noopTestProvider is a providerAdapter whose Validate always succeeds, used
// to isolate the validateVersionBundle failure path in Reconcile.
type noopTestProvider struct{}

func (p *noopTestProvider) Name() string                       { return "test-provider" }
func (p *noopTestProvider) Types() func(*runtime.Scheme) error { return nil }
func (p *noopTestProvider) Validate(*controller.Context) error { return nil }
func (p *noopTestProvider) Sync(*controller.Context) error     { return nil }
func (p *noopTestProvider) Cleanup(*controller.Context) error  { return nil }
func (p *noopTestProvider) Status(*controller.Context) (controller.Status, error) {
	return controller.Status{}, nil
}

// TestReconcile_VersionBundleValidationFailureSetsStatusMessage is a
// regression test for #3346: when validateVersionBundle fails, the Instance
// is marked Phase=Failed but Status.Message is left empty, hiding the actual
// error from everestctl/UI users.
func TestReconcile_VersionBundleValidationFailureSetsStatusMessage(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	provider := &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: "test-provider"},
		Spec: corev1alpha1.ProviderSpec{
			Versions: []corev1alpha1.VersionBundle{
				{Name: "1.0", Components: map[string]string{"engine": "1.0"}},
			},
		},
	}
	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "my-db",
			Namespace:  "default",
			Labels:     map[string]string{controller.ProviderLabel: "test-provider"},
			Finalizers: []string{finalizerName},
		},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef: common.ObjectRef{Name: "test-provider"},
			Version:     "does-not-exist",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(provider, instance).
		WithStatusSubresource(&corev1alpha1.Instance{}).
		Build()
	r := &ProviderReconciler{provider: &noopTestProvider{}, Client: c}

	_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(instance)})
	require.Error(t, err)

	got := &corev1alpha1.Instance{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(instance), got))
	assert.Equal(t, corev1alpha1.InstancePhaseFailed, got.Status.Phase)
	assert.Contains(t, got.Status.Message, "is not defined by provider")
}
