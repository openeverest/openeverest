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

package restore

import (
	"context"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/cli/deletion"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

// restore state values (client.Restore.Status.State).
const (
	restoreStatePending   = "Pending"
	restoreStateRunning   = "Running"
	restoreStateSucceeded = "Succeeded"
	restoreStateFailed    = "Failed"
)

// restoreCondition maps Succeeded/Failed to terminal outcomes and everything
// else to Pending.
func restoreCondition(r *client.Restore) (wait.Outcome, string) {
	state := restoreState(r)
	switch state {
	case restoreStateSucceeded:
		return wait.Succeeded, state
	case restoreStateFailed:
		return wait.Failed, restoreFailureMessage(r)
	default:
		return wait.Pending, state
	}
}

// newRestorePoll returns a PollFunc for the restore; a 404 is terminal here,
// unlike the delete-side poll.
func newRestorePoll(
	c *client.ClientWithResponses,
	cluster, namespace, name string,
) wait.PollFunc[*client.Restore] {
	return wait.FetchPoll("restore", name, wait.NotFoundTerminal, fetchRestore(c, cluster, namespace, name))
}

// fetchRestore adapts GetRestoreWithResponse to wait.FetchPoll's fetch shape.
func fetchRestore(c *client.ClientWithResponses, cluster, namespace, name string) func(context.Context) (int, string, *client.Restore, error) {
	return func(ctx context.Context) (int, string, *client.Restore, error) {
		resp, err := c.GetRestoreWithResponse(ctx, cluster, namespace, name)
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

func restoreFailureMessage(r *client.Restore) string {
	if r == nil || r.Status == nil || r.Status.Message == nil || *r.Status.Message == "" {
		return "restore entered the Failed state"
	}
	return "restore entered the Failed state: " + *r.Status.Message
}

// newRestoreDeletePoll checks if the restore still exists. A 404 means it's
// gone, which is success here (unlike the create-side poll).
func newRestoreDeletePoll(
	c *client.ClientWithResponses,
	cluster, namespace, name string,
) wait.PollFunc[*client.Restore] {
	return deletion.GonePoll("restore", name, fetchRestore(c, cluster, namespace, name))
}

func deleteCondition(r *client.Restore) (wait.Outcome, string) {
	return deletion.GoneCondition("restore deleted", func(v *client.Restore) string {
		return "restore still exists (state: " + restoreState(v) + ")"
	})(r)
}
