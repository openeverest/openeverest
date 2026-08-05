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

// Package deletion holds the plumbing shared by every `<resource> delete
// --wait` implementation: the "gone" wait.Condition, the GET-until-404 poll
// shape, and the three output lines (deleted / already-gone /
// wait-succeeded).
package deletion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// GoneCondition returns a wait.Condition for a delete-poll: nil means the
// resource is gone (Succeeded); a non-nil value is Pending, with its
// message computed by pendingMsg (which may inspect the live object, e.g.
// to report phase/progress).
func GoneCondition[T any](goneMsg string, pendingMsg func(*T) string) wait.Condition[*T] {
	return func(v *T) (wait.Outcome, string) {
		if v == nil {
			return wait.Succeeded, goneMsg
		}
		return wait.Pending, pendingMsg(v)
	}
}

// GonePoll builds a wait.PollFunc for a delete-poll around fetch,
// classifying responses the same way every `<resource> delete --wait`
// already does: a 404 means gone (nil, nil); a failed token refresh and a
// 401 are terminal; everything else unexpected is retryable.
func GonePoll[T any](
	kind, name string,
	fetch func(ctx context.Context) (statusCode int, statusText string, body *T, err error),
) wait.PollFunc[*T] {
	return func(ctx context.Context) (*T, error) {
		status, statusText, body, err := fetch(ctx)
		if err != nil {
			if errors.Is(err, authcli.ErrTokenRefresh) {
				return nil, fmt.Errorf("failed to fetch %s %q: %w", kind, name, err)
			}
			return nil, &wait.RetryableError{Err: fmt.Errorf("failed to fetch %s %q: %w", kind, name, err)}
		}
		switch status {
		case http.StatusOK:
			if body == nil {
				return nil, &wait.RetryableError{Err: fmt.Errorf("empty response body fetching %s %q", kind, name)}
			}
			return body, nil
		case http.StatusNotFound:
			return nil, nil // gone
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("server rejected credentials — run 'everestctl auth login' again")
		default:
			return nil, &wait.RetryableError{Err: fmt.Errorf("unexpected response fetching %s %q: %s", kind, name, statusText)}
		}
	}
}

// EmitDeleted prints success: a line in pretty mode, JSON otherwise.
func EmitDeleted(pretty bool, kind, name, namespace string) error {
	if pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("%s %q deleted", titleFirst(kind), name))
		return nil
	}
	return writeResultJSON(name, namespace, true)
}

// EmitAlreadyGone reports a --ignore-not-found short-circuit: nothing was
// deleted, so "deleted" must stay false — it should only ever mean this
// run actually deleted something.
func EmitAlreadyGone(pretty bool, kind, name, namespace string) error {
	if pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Info("%s %q not found in namespace %q; nothing to delete", titleFirst(kind), name, namespace))
		return nil
	}
	return writeResultJSON(name, namespace, false)
}

// EmitWaitSucceeded prints the --wait completion line: a line in pretty
// mode, JSON otherwise. Wording ("is deleted") deliberately differs from
// EmitDeleted's ("deleted") — this is reported after waiting *for* the
// resource to become gone, not immediately after issuing the delete.
func EmitWaitSucceeded(pretty bool, kind, name, namespace string) error {
	if pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("%s %q is deleted", titleFirst(kind), name))
		return nil
	}
	return writeResultJSON(name, namespace, true)
}

func writeResultJSON(name, namespace string, deleted bool) error {
	result := struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Deleted   bool   `json:"deleted"`
	}{Name: name, Namespace: namespace, Deleted: deleted}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		return fmt.Errorf("failed to encode delete result: %w", err)
	}
	return nil
}

func titleFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
