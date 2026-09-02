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

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func deprecationTestProvider() *corev1alpha1.Provider {
	return &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: "test-provider"},
		Spec: corev1alpha1.ProviderSpec{
			Release: &corev1alpha1.Release{Version: "0.2"},
			Components: map[string]corev1alpha1.Component{
				"engine": {Type: "mongod"},
			},
			ComponentTypes: map[string]corev1alpha1.ComponentType{
				"mongod": {Versions: []corev1alpha1.ComponentVersion{
					{Version: "6.0.19-16", Deprecated: true, RemovedInVersion: "0.3"},
					{Version: "8.0.12-4", Default: true},
				}},
			},
			Versions: []corev1alpha1.VersionBundle{
				{Name: "8.0.12", Default: true, Components: map[string]string{"engine": "8.0.12-4"}},
			},
		},
	}
}

func deprecationTestInstance(engineVersion string) *corev1alpha1.Instance {
	return &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default"},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef: common.ObjectRef{Name: "test-provider"},
			Components: map[string]corev1alpha1.ComponentSpec{
				"engine": {Version: engineVersion},
			},
		},
	}
}

func findDeprecationCondition(in *corev1alpha1.Instance) *metav1.Condition {
	for i := range in.Status.Conditions {
		if in.Status.Conditions[i].Type == corev1alpha1.ConditionComponentVersionDeprecated {
			return &in.Status.Conditions[i]
		}
	}
	return nil
}

func TestSetDeprecationCondition(t *testing.T) {
	t.Parallel()

	t.Run("deprecated version sets True with runway message", func(t *testing.T) {
		t.Parallel()

		in := deprecationTestInstance("6.0.19-16")
		r := &ProviderReconciler{Client: newFakeClient(newTestScheme(), deprecationTestProvider(), in)}

		r.setDeprecationCondition(t.Context(), in)

		cond := findDeprecationCondition(in)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, corev1alpha1.ReasonScheduledForRemoval, cond.Reason)
		assert.Contains(t, cond.Message, "6.0.19-16")
		assert.Contains(t, cond.Message, "0.3")
	})

	t.Run("supported version leaves condition absent", func(t *testing.T) {
		t.Parallel()

		in := deprecationTestInstance("8.0.12-4")
		r := &ProviderReconciler{Client: newFakeClient(newTestScheme(), deprecationTestProvider(), in)}

		r.setDeprecationCondition(t.Context(), in)

		assert.Nil(t, findDeprecationCondition(in))
	})

	t.Run("remediation flips existing condition to False", func(t *testing.T) {
		t.Parallel()

		in := deprecationTestInstance("8.0.12-4")
		in.Status.Conditions = []metav1.Condition{{
			Type:   corev1alpha1.ConditionComponentVersionDeprecated,
			Status: metav1.ConditionTrue,
			Reason: corev1alpha1.ReasonScheduledForRemoval,
		}}
		r := &ProviderReconciler{Client: newFakeClient(newTestScheme(), deprecationTestProvider(), in)}

		r.setDeprecationCondition(t.Context(), in)

		cond := findDeprecationCondition(in)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, corev1alpha1.ReasonVersionsSupported, cond.Reason)
	})

	t.Run("frozen bundle version is resolved and flagged", func(t *testing.T) {
		t.Parallel()

		provider := deprecationTestProvider()
		provider.Spec.Versions = append(provider.Spec.Versions,
			corev1alpha1.VersionBundle{Name: "6.0.19", Components: map[string]string{"engine": "6.0.19-16"}})
		in := deprecationTestInstance("")
		in.Status.Version = "6.0.19"
		r := &ProviderReconciler{Client: newFakeClient(newTestScheme(), provider, in)}

		r.setDeprecationCondition(t.Context(), in)

		cond := findDeprecationCondition(in)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
	})

	t.Run("missing provider is a no-op", func(t *testing.T) {
		t.Parallel()

		in := deprecationTestInstance("6.0.19-16")
		r := &ProviderReconciler{Client: newFakeClient(newTestScheme(), in)}

		r.setDeprecationCondition(t.Context(), in)

		assert.Nil(t, findDeprecationCondition(in))
	})
}
