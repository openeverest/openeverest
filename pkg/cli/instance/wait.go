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

package instance

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

// instanceCondition classifies an Instance: Ready -> Succeeded,
// Failed/Terminating -> Failed, anything else (incl. Suspended) -> Pending.
func instanceCondition(inst *client.Instance) (wait.Outcome, string) {
	phase := instancePhase(inst)
	switch phase {
	case string(client.InstanceStatusPhaseReady):
		return wait.Succeeded, progressMessage(inst, phase)
	case string(client.InstanceStatusPhaseFailed):
		msg := instanceStatusMessage(inst)
		if msg == "" {
			msg = "instance entered the Failed phase"
		} else {
			msg = "instance entered the Failed phase: " + msg
		}
		return wait.Failed, msg
	case string(client.InstanceStatusPhaseTerminating):
		return wait.Failed, "instance is being deleted"
	default:
		return wait.Pending, progressMessage(inst, phase)
	}
}

// newInstancePoll returns a PollFunc for the instance. A 404 is a terminal
// error (the resource is gone), matching `instance status --watch`.
func newInstancePoll(
	c *client.ClientWithResponses,
	cluster, namespace, name string,
) wait.PollFunc[*client.Instance] {
	return func(ctx context.Context) (*client.Instance, error) {
		resp, err := c.GetInstanceWithResponse(ctx, cluster, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch instance %q: %w", name, err)
		}
		switch resp.StatusCode() {
		case http.StatusOK:
			if resp.JSON200 == nil {
				return nil, fmt.Errorf("empty response body fetching instance %q", name)
			}
			return resp.JSON200, nil
		case http.StatusNotFound:
			return nil, fmt.Errorf("instance %q was deleted while waiting", name)
		default:
			return nil, fmt.Errorf("unexpected response fetching instance %q: %s", name, resp.Status())
		}
	}
}

func instancePhase(inst *client.Instance) string {
	if inst == nil || inst.Status == nil || inst.Status.Phase == nil {
		return ""
	}
	return string(*inst.Status.Phase)
}

func instanceStatusMessage(inst *client.Instance) string {
	if inst == nil || inst.Status == nil || inst.Status.Message == nil {
		return ""
	}
	return strings.TrimSpace(*inst.Status.Message)
}

// progressMessage renders a one-line phase + component summary, e.g.
// "Provisioning (engine 1/3 ready)".
func progressMessage(inst *client.Instance, phase string) string {
	if phase == "" {
		phase = "-"
	}
	comps := componentSummary(inst)
	if comps == "" {
		return phase
	}
	return fmt.Sprintf("%s (%s)", phase, comps)
}

func componentSummary(inst *client.Instance) string {
	if inst == nil || inst.Status == nil || inst.Status.Components == nil {
		return ""
	}
	parts := make([]string, 0, len(*inst.Status.Components))
	for _, comp := range *inst.Status.Components {
		state := "-"
		if comp.State != nil {
			state = *comp.State
		}
		var ready, total int32
		if comp.Ready != nil {
			ready = *comp.Ready
		}
		if comp.Total != nil {
			total = *comp.Total
		}
		parts = append(parts, fmt.Sprintf("%s %d/%d ready", state, ready, total))
	}
	return strings.Join(parts, ", ")
}
