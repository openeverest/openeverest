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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func TestStatus_ToV2Alpha1(t *testing.T) {
	t.Parallel()

	status := Provisioning("waiting for cluster...").ToV2Alpha1()

	assert.Equal(t, v1alpha1.InstancePhaseProvisioning, status.Phase)
	assert.Equal(t, "waiting for cluster...", status.Message)
}
