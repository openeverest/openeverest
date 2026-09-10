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
	"errors"

	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// isValidationError reports whether err is a Backup/Restore
// reference-validation failure (a referenced object doesn't exist, or
// doesn't satisfy a compatibility rule) as opposed to an unexpected
// runtime/infrastructure error (API server unavailable, connection refused,
// timeout, etc.). Every reference-validation sentinel satisfies
// errors.Is(err, controller.ErrInvalidReference), so new sentinels are
// classified correctly here without updating this function.
//
// CreateBackup and CreateRestore use this to decide whether an error from
// their validation step should be reported to the client as an invalid
// request (ErrInvalidRequest, HTTP 400) or propagated unchanged so it is
// handled as an internal error instead of being misrepresented as a client
// mistake.
func isValidationError(err error) bool {
	return errors.Is(err, controller.ErrInvalidReference)
}
