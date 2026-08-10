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
	"fmt"
	"os"
	"strings"

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

// GonePoll builds a wait.PollFunc for a delete-poll around fetch: a 404 means
// the resource is gone (nil, nil). Everything else is classified the same way
// every polling command is, via wait.FetchPoll.
func GonePoll[T any](
	kind, name string,
	fetch func(ctx context.Context) (statusCode int, statusText string, body *T, err error),
) wait.PollFunc[*T] {
	return wait.FetchPoll(kind, name, wait.NotFoundIsSuccess, fetch)
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
