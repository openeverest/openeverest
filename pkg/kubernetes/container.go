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
	corev1 "k8s.io/api/core/v1"
)

// ContainerState describes container's state - waiting, running, terminated.
type ContainerState string

const (
	// ContainerStateWaiting represents a state when container requires some
	// operations being done in order to complete start up.
	ContainerStateWaiting ContainerState = "waiting"
	// ContainerStateTerminated indicates that container began execution and
	// then either ran to completion or failed for some reason.
	ContainerStateTerminated ContainerState = "terminated"
)

// IsContainerInState returns true if the named container is in the given state.
func IsContainerInState(containerStatuses []corev1.ContainerStatus, name string, state ContainerState) bool {
	for _, status := range containerStatuses {
		if status.Name != name {
			continue
		}

		switch state {
		case ContainerStateWaiting:
			return status.State.Waiting != nil
		case ContainerStateTerminated:
			return status.State.Terminated != nil
		default:
			return false
		}
	}

	return false
}
