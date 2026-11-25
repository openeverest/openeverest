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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract/summary"
)

// The fixtures' summary conditions keep the rules provider-runtime writes
// them by, so the clients are tested against statuses a provider can write.
func TestFixtures_SummaryConditions(t *testing.T) {
	t.Parallel()

	for _, tc := range loadFixtures(t).Cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			var inst corev1alpha1.Instance
			require.NoError(t, json.Unmarshal(tc.Instance, &inst))

			assert.NoError(t, summary.CheckConditions(inst.Status.Conditions))
		})
	}
}
