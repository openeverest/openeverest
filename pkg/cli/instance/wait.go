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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/cli/deletion"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
	"github.com/openeverest/openeverest/v2/pkg/instancestatus"
)

// instanceCondition classifies an Instance for --wait. The verdict comes from
// the same reading of the status that is displayed, so a wait ends when what
// is shown says so: Ready for the current spec is success; Stalled for the
// current spec, which nothing but a person resolves, and a deletion are
// failures; anything else keeps waiting.
func instanceCondition(inst *client.Instance) (wait.Outcome, string) {
	in := statusInput(inst)
	shown := instancestatus.Describe(in, time.Now())
	switch instancestatus.WaitOutcome(in) {
	case instancestatus.Succeeded:
		return wait.Succeeded, progressMessage(inst, shown)
	case instancestatus.Failed:
		return wait.Failed, failureMessage(inst, shown)
	case instancestatus.Pending:
	}
	return wait.Pending, progressMessage(inst, shown)
}

// statusInput adapts the API client's Instance to the shared reading of an
// Instance's status.
func statusInput(inst *client.Instance) instancestatus.Input {
	if inst == nil {
		return instancestatus.Input{}
	}
	in := instancestatus.Input{Meta: inst.Metadata}
	if inst.Status == nil {
		return in
	}
	if inst.Status.ObservedGeneration != nil {
		in.ObservedGeneration = *inst.Status.ObservedGeneration
	}
	if inst.Status.Phase != nil {
		in.Phase = string(*inst.Status.Phase)
	}
	if inst.Status.Conditions == nil {
		return in
	}
	in.Conditions = make([]metav1.Condition, 0, len(*inst.Status.Conditions))
	for _, c := range *inst.Status.Conditions {
		cond := metav1.Condition{
			Type:               c.Type,
			Status:             metav1.ConditionStatus(c.Status),
			Reason:             c.Reason,
			Message:            c.Message,
			LastTransitionTime: metav1.NewTime(c.LastTransitionTime),
		}
		if c.ObservedGeneration != nil {
			cond.ObservedGeneration = *c.ObservedGeneration
		}
		in.Conditions = append(in.Conditions, cond)
	}
	return in
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

// failureMessage says why waiting for the Instance to become ready is over.
func failureMessage(inst *client.Instance, shown instancestatus.Display) string {
	if inst != nil && inst.Metadata != nil && inst.Metadata.DeletionTimestamp != nil {
		return "instance is being deleted"
	}
	msg := "instance will not become ready without intervention: " + shown.Label
	if detail := strings.TrimSpace(shown.Message); detail != "" {
		msg += ": " + detail
	}
	return msg
}

// progressMessage renders a one-line status + component summary, e.g.
// "Creating (engine 1/3 ready)".
func progressMessage(inst *client.Instance, shown instancestatus.Display) string {
	comps := componentSummary(inst)
	if comps == "" {
		return shown.Sentence
	}
	return fmt.Sprintf("%s (%s)", shown.Sentence, comps)
}

func componentSummary(inst *client.Instance) string {
	if inst == nil || inst.Status == nil || inst.Status.Components == nil {
		return ""
	}
	parts := make([]string, 0, len(*inst.Status.Components))
	for _, comp := range *inst.Status.Components {
		var ready, replicas int32
		if comp.ReadyReplicas != nil {
			ready = *comp.ReadyReplicas
		}
		if comp.Replicas != nil {
			replicas = *comp.Replicas
		}
		parts = append(parts, fmt.Sprintf("%s %d/%d ready", comp.Name, ready, replicas))
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
		return progressMessage(v, instancestatus.Describe(statusInput(v), time.Now()))
	})(inst)
}
