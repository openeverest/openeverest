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
	"errors"
	"fmt"
	"net/http"

	"github.com/openeverest/openeverest/v2/client"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

// terminal restore states.
const (
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

// newRestorePoll returns a PollFunc for the restore.
func newRestorePoll(
	c *client.ClientWithResponses,
	cluster, namespace, name string,
) wait.PollFunc[*client.Restore] {
	return func(ctx context.Context) (*client.Restore, error) {
		resp, err := c.GetRestoreWithResponse(ctx, cluster, namespace, name)
		if err != nil {
			// A failed token refresh is terminal; other fetch errors are transient.
			if errors.Is(err, authcli.ErrTokenRefresh) {
				return nil, fmt.Errorf("failed to fetch restore %q: %w", name, err)
			}
			return nil, &wait.RetryableError{Err: fmt.Errorf("failed to fetch restore %q: %w", name, err)}
		}
		switch resp.StatusCode() {
		case http.StatusOK:
			if resp.JSON200 == nil {
				return nil, &wait.RetryableError{Err: fmt.Errorf("empty response body fetching restore %q", name)}
			}
			return resp.JSON200, nil
		case http.StatusNotFound:
			return nil, fmt.Errorf("restore %q was deleted while waiting", name)
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("server rejected credentials — run 'everestctl auth login' again")
		default:
			return nil, &wait.RetryableError{Err: fmt.Errorf("unexpected response fetching restore %q: %s", name, resp.Status())}
		}
	}
}

func restoreFailureMessage(r *client.Restore) string {
	if r == nil || r.Status == nil || r.Status.Message == nil || *r.Status.Message == "" {
		return "restore entered the Failed state"
	}
	return "restore entered the Failed state: " + *r.Status.Message
}
