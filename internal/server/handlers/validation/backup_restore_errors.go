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

package validation

import (
	"context"
	"errors"

	"github.com/openeverest/openeverest/v2/internal/backupadmission"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// admitOperation applies the decision on a Backup, Restore or BackupImport
// as core's validating webhook does, and also rejects one that references an
// object that does not exist: operation objects may reject dangling
// references (spec 015), and a person picking from lists never means one.
// The warnings of an admitted operation reach the client as Warning headers,
// once even when the webhook raises them too.
func admitOperation(ctx context.Context, d backupadmission.Decision, err error) error {
	if err != nil {
		return err
	}
	if d.Denied == nil && len(d.Missing) > 0 {
		d.Denied = d.Missing[0]
	}
	if d.Denied != nil {
		if isValidationError(d.Denied) {
			return errors.Join(ErrInvalidRequest, d.Denied)
		}
		return d.Denied
	}
	for _, w := range d.Warnings() {
		kubernetes.AddWarning(ctx, w)
	}
	return nil
}

// isValidationError reports whether err is a Backup/Restore
// reference-validation failure (a referenced object doesn't exist, or
// doesn't satisfy a compatibility rule) as opposed to an unexpected
// runtime/infrastructure error (API server unavailable, connection refused,
// timeout, etc.). Every reference-validation sentinel satisfies
// errors.Is(err, controller.ErrInvalidReference), so new sentinels are
// classified correctly here without updating this function.
//
// admitOperation uses this to decide whether a rejection should be reported
// to the client as an invalid request (ErrInvalidRequest, HTTP 400) or
// propagated unchanged so it is handled as an internal error instead of
// being misrepresented as a client mistake.
func isValidationError(err error) bool {
	return errors.Is(err, controller.ErrInvalidReference)
}
