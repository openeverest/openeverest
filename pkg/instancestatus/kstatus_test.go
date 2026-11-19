// everest
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

package instancestatus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utiljson "k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// GitOps tools that wait for what they apply, such as Flux and Helm, do not
// know what an Instance is: they read its status with kstatus. Each fixture
// records what kstatus makes of the Instance, and this runs kstatus itself
// to hold the fixtures to it.
func TestFixtures_Kstatus(t *testing.T) {
	t.Parallel()

	for _, tc := range loadFixtures(t).Cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			// This decoder keeps whole numbers as integers, which is how an
			// object read from the cluster arrives and what kstatus expects.
			var object map[string]any
			require.NoError(t, utiljson.Unmarshal(tc.Instance, &object))
			instance := &unstructured.Unstructured{Object: object}
			instance.SetGroupVersionKind(corev1alpha1.GroupVersion.WithKind("Instance"))

			got, err := status.Compute(instance)

			require.NoError(t, err)
			assert.Equal(t, tc.Want.Kstatus, got.Status.String())
		})
	}
}
