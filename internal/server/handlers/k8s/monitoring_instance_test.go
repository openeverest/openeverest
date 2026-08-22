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

package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/kubernetes"
)

func TestDeleteMonitoringInstance(t *testing.T) {
	t.Parallel()

	const (
		namespace = "everest"
		name      = "pmm-local"
	)

	monitoringConfig := func() ctrlclient.Object {
		return &everestv1alpha1.MonitoringConfig{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}
	}
	apiKeySecret := func() ctrlclient.Object {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}
	}

	testCases := []struct {
		name string
		objs []ctrlclient.Object
	}{
		{
			name: "monitoring config and secret both present",
			objs: []ctrlclient.Object{monitoringConfig(), apiKeySecret()},
		},
		{
			// Regression test for the case where the Secret holding the PMM API
			// key was removed out of band. Deleting the monitoring instance used
			// to return the Secret's NotFound error, so the API answered 500
			// even though the MonitoringConfig had already been deleted.
			name: "secret already deleted out of band",
			objs: []ctrlclient.Object{monitoringConfig()},
		},
		{
			name: "monitoring config and secret both already gone",
			objs: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockClient := fakeclient.NewClientBuilder().
				WithScheme(kubernetes.CreateScheme()).
				WithObjects(tc.objs...).
				Build()

			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient)
			k8sH := New(zap.NewNop().Sugar(), k, "")

			err := k8sH.DeleteMonitoringInstance(context.Background(), namespace, name)
			require.NoError(t, err)

			key := types.NamespacedName{Namespace: namespace, Name: name}

			err = mockClient.Get(context.Background(), key, &everestv1alpha1.MonitoringConfig{})
			assert.True(t, k8serrors.IsNotFound(err), "monitoring config should be gone, got %v", err)

			err = mockClient.Get(context.Background(), key, &corev1.Secret{})
			assert.True(t, k8serrors.IsNotFound(err), "secret should be gone, got %v", err)
		})
	}
}
