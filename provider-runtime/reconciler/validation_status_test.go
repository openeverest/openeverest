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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func TestFailValidationSurfacesReason(t *testing.T) {
	t.Parallel()

	in := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default"},
		Status:     corev1alpha1.InstanceStatus{Phase: corev1alpha1.InstancePhaseReady, Message: "stale"},
	}
	c := fake.NewClientBuilder().
		WithScheme(newTestScheme()).
		WithObjects(in).
		WithStatusSubresource(&corev1alpha1.Instance{}).
		Build()
	r := &ProviderReconciler{Client: c}

	r.failValidation(t.Context(), in, errors.New(`topology cannot be changed from cluster to standalone`))

	stored := &corev1alpha1.Instance{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(in), stored))
	assert.Equal(t, corev1alpha1.InstancePhaseFailed, stored.Status.Phase)
	assert.Equal(t, "topology cannot be changed from cluster to standalone", stored.Status.Message)
}
