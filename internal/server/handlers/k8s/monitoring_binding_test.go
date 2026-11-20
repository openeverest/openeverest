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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestK8s_ListMonitoringBindings(t *testing.T) {
	t.Parallel()

	binding := func(namespace, name, destination string) *monitoringv1alpha1.MonitoringBinding {
		return &monitoringv1alpha1.MonitoringBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: name, Namespace: namespace,
				Labels: map[string]string{monitoringv1alpha1.BindingDestinationLabel: destination},
			},
		}
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithObjects(
			binding("ns1", "db1-pmm-0a1b2c3d", "pmm-prod"),
			binding("ns1", "db2-pmm-1a2b3c4d", "pmm-prod"),
			binding("ns1", "db1-prom-2a3b4c5d", "prometheus"),
			binding("ns2", "db3-pmm-3a4b5c6d", "pmm-prod"),
		).
		Build()
	handler := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
		log:           zap.NewNop().Sugar(),
	}

	testCases := []struct {
		desc   string
		params *api.ListMonitoringBindingsParams
		want   []string
	}{
		{
			desc:   "no filter returns the namespace",
			params: &api.ListMonitoringBindingsParams{},
			want:   []string{"db1-pmm-0a1b2c3d", "db2-pmm-1a2b3c4d", "db1-prom-2a3b4c5d"},
		},
		{
			desc:   "nil params returns the namespace",
			params: nil,
			want:   []string{"db1-pmm-0a1b2c3d", "db2-pmm-1a2b3c4d", "db1-prom-2a3b4c5d"},
		},
		{
			desc:   "destination filter",
			params: &api.ListMonitoringBindingsParams{Destination: new("pmm-prod")},
			want:   []string{"db1-pmm-0a1b2c3d", "db2-pmm-1a2b3c4d"},
		},
		{
			desc:   "empty destination is no filter",
			params: &api.ListMonitoringBindingsParams{Destination: new("")},
			want:   []string{"db1-pmm-0a1b2c3d", "db2-pmm-1a2b3c4d", "db1-prom-2a3b4c5d"},
		},
		{
			desc:   "unused destination",
			params: &api.ListMonitoringBindingsParams{Destination: new("victoriametrics")},
			want:   []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			list, err := handler.ListMonitoringBindings(context.Background(), "prod", "ns1", tc.params)
			require.NoError(t, err)

			names := make([]string, 0, len(list.Items))
			for _, b := range list.Items {
				names = append(names, b.Name)
			}
			assert.ElementsMatch(t, tc.want, names)
		})
	}
}
