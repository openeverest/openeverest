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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// TestReconcile_MetadataWritesKeepUnknownSpec runs the reconciler with a
// client that decodes Instances without spec.monitoring, as a provider built
// against an older core does, and checks the stored spec survives the
// runtime's label and finalizer writes.
func TestReconcile_MetadataWritesKeepUnknownSpec(t *testing.T) {
	t.Parallel()

	monitoringSpec := &v1alpha1.InstanceMonitoringSpec{
		Destinations: []v1alpha1.InstanceMonitoringDestination{{
			Name:           "pmm",
			DestinationRef: common.MonitoringDestinationRef{Kind: "MonitoringDestination", Name: "pmm-prod"},
		}},
	}
	tests := []struct {
		name           string
		in             *v1alpha1.Instance
		wantFinalizers []string
	}{
		{
			name: "adding the provider label and finalizer",
			in: readyTestInstance(func(in *v1alpha1.Instance) {
				in.Labels = nil
				in.Finalizers = nil
				in.Spec.Monitoring = monitoringSpec.DeepCopy()
			}),
			wantFinalizers: []string{finalizerName},
		},
		{
			name: "removing the finalizer on deletion",
			in: readyTestInstance(func(in *v1alpha1.Instance) {
				in.Finalizers = []string{finalizerName, "example.com/keep"}
				in.DeletionTimestamp = new(metav1.Now())
				in.Spec.Monitoring = monitoringSpec.DeepCopy()
				in.Status.Phase = v1alpha1.InstancePhaseTerminating
			}),
			wantFinalizers: []string{"example.com/keep"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stored := fake.NewClientBuilder().
				WithScheme(newTestScheme()).
				WithObjects(tc.in).
				WithStatusSubresource(&v1alpha1.Instance{}).
				Build()
			olderDecoder := interceptor.NewClient(stored, interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if err := c.Get(ctx, key, obj, opts...); err != nil {
						return err
					}
					if in, ok := obj.(*v1alpha1.Instance); ok {
						in.Spec.Monitoring = nil
					}
					return nil
				},
			})
			r := &ProviderReconciler{provider: &readyTestProvider{}, Client: olderDecoder}

			key := client.ObjectKeyFromObject(tc.in)
			_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
			require.NoError(t, err)

			got := &v1alpha1.Instance{}
			require.NoError(t, stored.Get(t.Context(), key, got))
			assert.Equal(t, monitoringSpec, got.Spec.Monitoring)
			assert.Equal(t, tc.wantFinalizers, got.Finalizers)
			assert.Equal(t, "test-provider", got.Labels[controller.ProviderLabel])
		})
	}
}
