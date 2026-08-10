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
	corev1 "k8s.io/api/core/v1"
)

func TestIsContainerInState(t *testing.T) {
	t.Parallel()

	statuses := []corev1.ContainerStatus{
		{
			Name: "init-a",
			State: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{},
			},
		},
		{
			Name: "init-b",
			State: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{},
			},
		},
		{
			Name: "init-c",
			State: corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{},
			},
		},
	}

	assert.True(t, IsContainerInState(statuses, "init-a", ContainerStateTerminated))
	assert.False(t, IsContainerInState(statuses, "init-b", ContainerStateTerminated))
	assert.False(t, IsContainerInState(statuses, "init-missing", ContainerStateTerminated))
	assert.True(t, IsContainerInState(statuses, "init-c", ContainerStateWaiting))
	assert.False(t, IsContainerInState(statuses, "init-a", ContainerStateWaiting))
}
