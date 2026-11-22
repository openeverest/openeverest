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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// A watch can queue an Instance of another provider, for example a Restore
// that names it. The reconciler leaves it untouched, also while it is deleted.
func TestReconcile_IgnoresInstancesOfOtherProviders(t *testing.T) {
	t.Parallel()

	foreign := func(in *v1alpha1.Instance) {
		in.Spec.ProviderRef = common.ObjectRef{Name: "other-provider"}
		in.Labels = map[string]string{controller.ProviderLabel: "other-provider"}
		in.Status.ObservedGeneration = 1
	}
	tests := []struct {
		name   string
		mutate func(*v1alpha1.Instance)
	}{
		{name: "live", mutate: foreign},
		{name: "being deleted", mutate: func(in *v1alpha1.Instance) {
			foreign(in)
			deleted := metav1.NewTime(time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC))
			in.DeletionTimestamp = &deleted
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cleaned := false
			p := &readyTestProvider{status: controller.Ready(), cleanup: func(*controller.Context) { cleaned = true }}
			in := readyTestInstance(tc.mutate)
			st := newStatusTest(t, p, readyTestProviderObject(), in)

			got, res, err := st.reconcile()
			require.NoError(t, err)
			assert.Equal(t, in.ResourceVersion, got.ResourceVersion, "the Instance is not written")
			assert.Equal(t, "other-provider", got.Labels[controller.ProviderLabel])
			assert.Equal(t, []string{finalizerName}, got.Finalizers)
			assert.Zero(t, st.writes, "no status write")
			assert.Zero(t, p.syncs, "no Sync")
			assert.Zero(t, p.calls, "no Validate")
			assert.False(t, cleaned, "no Cleanup")
			assert.Zero(t, res.RequeueAfter)
		})
	}
}
