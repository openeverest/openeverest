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

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

func TestDeleteUnusedMonitoringDestinations(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	require.NoError(t, monitoringv1alpha1.AddToScheme(scheme))
	destination := func(namespace, name string, attached int32) *monitoringv1alpha1.MonitoringDestination {
		return &monitoringv1alpha1.MonitoringDestination{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Status:     monitoringv1alpha1.MonitoringDestinationStatus{AttachedInstances: attached},
		}
	}
	k := NewEmpty(zap.NewNop().Sugar(), "")
	k.k8sClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		destination("team-a", "unused", 0),
		destination("team-a", "in-use", 2),
		destination("team-b", "unused", 0),
	).Build()

	require.NoError(t, k.DeleteUnusedMonitoringDestinations(t.Context(), ctrlclient.InNamespace("team-a")))

	list := &monitoringv1alpha1.MonitoringDestinationList{}
	require.NoError(t, k.k8sClient.List(t.Context(), list))
	left := make([]string, 0, len(list.Items))
	for _, d := range list.Items {
		left = append(left, d.Namespace+"/"+d.Name)
	}
	assert.ElementsMatch(t, []string{"team-a/in-use", "team-b/unused"}, left)
}
