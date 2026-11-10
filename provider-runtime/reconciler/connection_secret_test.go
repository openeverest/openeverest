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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestReconcile_ConnectionSecretNamesTheProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		details controller.ConnectionDetails
		want    map[string]string
	}{
		{
			name: "a different provider value is replaced",
			details: controller.ConnectionDetails{
				Type:     "cassandra",
				Provider: "cassandra",
				Host:     "orders.team-a.svc",
			},
			want: map[string]string{
				"type":     "cassandra",
				"provider": "test-provider",
				"host":     "orders.team-a.svc",
			},
		},
		{
			name: "a missing provider value is filled",
			details: controller.ConnectionDetails{
				Type: "mongodb",
				Port: "27017",
			},
			want: map[string]string{
				"type":     "mongodb",
				"provider": "test-provider",
				"port":     "27017",
			},
		},
		{
			name: "a provider value in AdditionalProperties is replaced",
			details: controller.ConnectionDetails{
				Host:                 "orders.team-a.svc",
				AdditionalProperties: map[string]string{"provider": "engine", "tls": "true"},
			},
			want: map[string]string{
				"provider": "test-provider",
				"host":     "orders.team-a.svc",
				"tls":      "true",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := &readyTestProvider{status: controller.ReadyWithConnectionDetails(tc.details)}
			provider := &v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}
			r := newReadyTestReconciler(p, provider, readyTestInstance(nil))

			got, err := reconcileAndGet(t, r)
			require.NoError(t, err)
			require.NotNil(t, got.Status.ConnectionSecretRef)
			assert.Equal(t, "orders"+controller.ConnectionSecretSuffix, got.Status.ConnectionSecretRef.Name)

			secret := &corev1.Secret{}
			key := client.ObjectKey{Namespace: "team-a", Name: got.Status.ConnectionSecretRef.Name}
			require.NoError(t, r.Get(t.Context(), key, secret))
			data := make(map[string]string, len(secret.Data))
			for k, v := range secret.Data {
				data[k] = string(v)
			}
			assert.Equal(t, tc.want, data)
			require.Len(t, secret.OwnerReferences, 1)
			assert.Equal(t, "orders", secret.OwnerReferences[0].Name)
		})
	}
}

func TestReconcile_NoConnectionDetailsWritesNoSecret(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{status: controller.Ready()}
	provider := &v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}
	r := newReadyTestReconciler(p, provider, readyTestInstance(nil))

	got, err := reconcileAndGet(t, r)
	require.NoError(t, err)
	assert.Nil(t, got.Status.ConnectionSecretRef)
	c := meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionConnectionDetailsReady)
	require.NotNil(t, c)
	assert.Equal(t, metav1.ConditionFalse, c.Status)

	err = r.Get(t.Context(), client.ObjectKey{Namespace: "team-a", Name: "orders" + controller.ConnectionSecretSuffix}, &corev1.Secret{})
	assert.True(t, apierrors.IsNotFound(err), "got %v", err)
}
