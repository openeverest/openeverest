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

import "fmt"

// ReadyCount is how many of a component's pods are Ready, out of how many it
// has or will have.
type ReadyCount struct {
	// Ready is readyReplicas, or 0 when it is absent.
	Ready int32
	// Total is desiredReplicas when the provider reports it, even when it
	// is 0, and replicas otherwise.
	Total int32
	// TotalIsDesired is true when Total is desiredReplicas.
	TotalIsDesired bool
}

// CountReady reads the readyReplicas, replicas and desiredReplicas of an
// entry of status.components. It asks whether each is present, never whether
// it is zero: a component scaled to zero has a desired count of 0, and one
// whose provider does not report it falls back to the pods that exist.
func CountReady(readyReplicas, replicas, desiredReplicas *int32) ReadyCount {
	var c ReadyCount
	if readyReplicas != nil {
		c.Ready = *readyReplicas
	}
	switch {
	case desiredReplicas != nil:
		c.Total, c.TotalIsDesired = *desiredReplicas, true
	case replicas != nil:
		c.Total = *replicas
	}

	return c
}

// String formats the count as "2/3". While a component shrinks it can read
// "4/3", as kubectl shows a Deployment.
func (c ReadyCount) String() string {
	return fmt.Sprintf("%d/%d", c.Ready, c.Total)
}
