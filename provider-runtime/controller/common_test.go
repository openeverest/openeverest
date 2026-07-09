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
	corev1 "k8s.io/api/core/v1"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func TestStatus_ToV2Alpha1(t *testing.T) {
	t.Parallel()

	s := Provisioning("waiting for cluster...")
	s.Components = []ComponentStatus{
		{
			Name:  "test-component",
			Ready: 1,
			Total: 2,
			State: "InProgress",
			Pods: []corev1.LocalObjectReference{
				{Name: "pod-1"},
			},
		},
	}

	status := s.ToV2Alpha1()

	assert.Equal(t, v1alpha1.InstancePhaseProvisioning, status.Phase)
	assert.Equal(t, "waiting for cluster...", status.Message)
	assert.Len(t, status.Components, 1)
	assert.Equal(t, "test-component", status.Components[0].Name)
	assert.Equal(t, int32(1), *status.Components[0].Ready)
	assert.Equal(t, int32(2), *status.Components[0].Total)
	assert.Equal(t, "InProgress", status.Components[0].State)
	assert.Len(t, status.Components[0].Pods, 1)
	assert.Equal(t, "pod-1", status.Components[0].Pods[0].Name)
}
