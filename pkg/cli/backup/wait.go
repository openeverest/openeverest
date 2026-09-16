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

package backup

import (
	"context"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/clienterr"
	"github.com/openeverest/openeverest/v2/pkg/cli/deletion"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

// backupCondition maps Succeeded/Failed to terminal outcomes and everything
// else to Pending. Error is deliberately non-terminal: the controller may
// still retry it.
func backupCondition(b *client.Backup) (wait.Outcome, string) {
	state := backupState(b)
	switch state {
	case client.BackupStatusStateSucceeded:
		return wait.Succeeded, string(state)
	case client.BackupStatusStateFailed:
		return wait.Failed, backupFailureMessage(b)
	default:
		display := string(state)
		if display == "" {
			display = "-"
		}
		return wait.Pending, display
	}
}

// newBackupPoll returns a PollFunc for the backup; a 404 is terminal here,
// unlike the delete-side poll.
func newBackupPoll(
	c *client.ClientWithResponses,
	cluster, namespace, name string,
) wait.PollFunc[*client.Backup] {
	return wait.FetchPoll("backup", name, wait.NotFoundTerminal, fetchBackup(c, cluster, namespace, name))
}

// fetchBackup adapts GetBackupWithResponse to wait.FetchPoll's fetch shape.
func fetchBackup(c *client.ClientWithResponses, cluster, namespace, name string) func(context.Context) (int, string, *client.Backup, error) {
	return func(ctx context.Context) (int, string, *client.Backup, error) {
		resp, err := c.GetBackupWithResponse(ctx, cluster, namespace, name)
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

func backupFailureMessage(b *client.Backup) string {
	if b == nil || b.Status == nil || b.Status.Message == nil || *b.Status.Message == "" {
		return "backup entered the Failed state"
	}
	return "backup entered the Failed state: " + *b.Status.Message
}

// newBackupDeletePoll checks if the backup still exists. A 404 means it's
// gone, which is success here (unlike the create-side poll).
func newBackupDeletePoll(
	c *client.ClientWithResponses,
	cluster, namespace, name string,
) wait.PollFunc[*client.Backup] {
	return deletion.GonePoll("backup", name, fetchBackup(c, cluster, namespace, name))
}

func deleteCondition(b *client.Backup) (wait.Outcome, string) {
	return deletion.GoneCondition("backup deleted", func(v *client.Backup) string {
		if state, _ := backupStateForGuard(v); state != "" {
			return "backup still exists (state: " + string(state) + ")"
		}
		return "backup still exists"
	})(b)
}
