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
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestK8s_ListMonitoringClasses_ProviderFilter(t *testing.T) {
	t.Parallel()

	class := func(name string, providers ...monitoringv1alpha1.ProviderCompatibility) ctrlclient.Object {
		return &monitoringv1alpha1.MonitoringClass{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Status:     monitoringv1alpha1.MonitoringClassStatus{Providers: providers},
		}
	}
	objs := []ctrlclient.Object{
		class("pmm",
			monitoringv1alpha1.ProviderCompatibility{Name: "percona-xtradb-cluster", Mode: corev1alpha1.MonitoringExecutionModeProviderManaged},
			monitoringv1alpha1.ProviderCompatibility{Name: "percona-postgresql", Mode: corev1alpha1.MonitoringExecutionModeProviderManaged},
		),
		class("prometheus",
			monitoringv1alpha1.ProviderCompatibility{Name: "percona-xtradb-cluster", Mode: corev1alpha1.MonitoringExecutionModeExtensionManaged},
			monitoringv1alpha1.ProviderCompatibility{Name: "percona-postgresql", Reason: monitoringv1alpha1.ReasonIncompatible},
		),
		class("unassessed"),
	}
	h := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(
			fake.NewClientBuilder().WithScheme(kubernetes.CreateScheme()).WithObjects(objs...).Build(),
		),
		log: zap.NewNop().Sugar(),
	}

	tests := map[string]struct {
		provider string
		want     []string
	}{
		"no filter returns every class":                       {provider: "", want: []string{"pmm", "prometheus", "unassessed"}},
		"provider with a mode on both classes":                {provider: "percona-xtradb-cluster", want: []string{"pmm", "prometheus"}},
		"incompatible entry and unassessed class are skipped": {provider: "percona-postgresql", want: []string{"pmm"}},
		"unknown provider matches nothing":                    {provider: "percona-server-mongodb", want: []string{}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			list, err := h.ListMonitoringClasses(context.Background(), "prod", tc.provider)
			require.NoError(t, err)
			got := make([]string, 0, len(list.Items))
			for _, c := range list.Items {
				got = append(got, c.Name)
			}
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}
