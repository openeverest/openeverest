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
	"strings"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/cli/deletion"
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

// newInstancePoll returns a PollFunc for the instance; a 404 is terminal
// here, unlike the delete-side poll.
func newInstancePoll(
	c *client.ClientWithResponses,
	cluster, namespace, name string,
) wait.PollFunc[*client.Instance] {
	return wait.FetchPoll("instance", name, wait.NotFoundTerminal, fetchInstance(c, cluster, namespace, name))
}

// fetchInstance adapts GetInstanceWithResponse to wait.FetchPoll's fetch
// shape.
func fetchInstance(c *client.ClientWithResponses, cluster, namespace, name string) func(context.Context) (int, string, *client.Instance, error) {
	return func(ctx context.Context) (int, string, *client.Instance, error) {
		resp, err := c.GetInstanceWithResponse(ctx, cluster, namespace, name)
		if err != nil {
			return 0, "", nil, err
		}
		statusText := resp.Status()
		if msg, ok := clienterr.Message(resp.JSONDefault); ok {
			statusText = msg
		}
		return resp.StatusCode(), statusText, resp.JSON200, nil
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

// newInstanceDeletePoll checks if the instance still exists. A 404 means
// it's gone, which is success here (unlike the create-side poll).
func newInstanceDeletePoll(
	c *client.ClientWithResponses,
	cluster, namespace, name string,
) wait.PollFunc[*client.Instance] {
	return deletion.GonePoll("instance", name, fetchInstance(c, cluster, namespace, name))
}

func deleteCondition(inst *client.Instance) (wait.Outcome, string) {
	return deletion.GoneCondition("instance deleted", func(v *client.Instance) string {
		return progressMessage(v, instancePhase(v))
	})(inst)
}
