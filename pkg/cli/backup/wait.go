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
	"errors"
	"fmt"
	"net/http"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

// backupCondition maps Succeeded/Failed to terminal outcomes and everything
// else to Pending. Error is deliberately non-terminal: the controller may
// still retry it.
func backupCondition(b *client.Backup) (wait.Outcome, string) {
	state := backupState(b)
	switch state {
	case backupStateSucceeded:
		return wait.Succeeded, state
	case backupStateFailed:
		return wait.Failed, backupFailureMessage(b)
	default:
		return wait.Pending, state
	}
}

// newBackupPoll returns a PollFunc for the backup, mirroring the error
// handling of `instance create --wait`/`status --watch`: transient fetch
// failures and unexpected statuses are retried (wait.RetryableError), while a
// 404 (deleted), a 401 (credentials rejected), and a failed token refresh are
// terminal.
func newBackupPoll(
	c *client.ClientWithResponses,
	cluster, namespace, name string,
) wait.PollFunc[*client.Backup] {
	return func(ctx context.Context) (*client.Backup, error) {
		resp, err := c.GetBackupWithResponse(ctx, cluster, namespace, name)
		if err != nil {
			// A failed token refresh is terminal; other fetch errors are transient.
			if errors.Is(err, authcli.ErrTokenRefresh) {
				return nil, fmt.Errorf("failed to fetch backup %q: %w", name, err)
			}
			return nil, &wait.RetryableError{Err: fmt.Errorf("failed to fetch backup %q: %w", name, err)}
		}
		switch resp.StatusCode() {
		case http.StatusOK:
			if resp.JSON200 == nil {
				return nil, &wait.RetryableError{Err: fmt.Errorf("empty response body fetching backup %q", name)}
			}
			return resp.JSON200, nil
		case http.StatusNotFound:
			return nil, fmt.Errorf("backup %q was deleted while waiting", name)
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("server rejected credentials — run 'everestctl auth login' again")
		default:
			return nil, &wait.RetryableError{Err: fmt.Errorf("unexpected response fetching backup %q: %s", name, resp.Status())}
		}
	}
}

func backupFailureMessage(b *client.Backup) string {
	if b == nil || b.Status == nil || b.Status.Message == nil || *b.Status.Message == "" {
		return "backup entered the Failed state"
	}
	return "backup entered the Failed state: " + *b.Status.Message
}
