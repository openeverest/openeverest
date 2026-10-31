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

package storageclasses

import "errors"

// Errors returned by the decoders. Executors map them to the reasons they
// report on an Instance storage entry.
var (
	// ErrUnsupportedParameter means spec.parameters carries a key the decoder
	// does not know, typically a parameter newer than the executor.
	ErrUnsupportedParameter = errors.New("unsupported parameter")
	// ErrInvalidParameters means spec.parameters lacks a required parameter
	// or holds a value the class does not accept.
	ErrInvalidParameters = errors.New("invalid parameters")
	// ErrInvalidLocation means spec.location breaks a naming rule of the
	// technology.
	ErrInvalidLocation = errors.New("invalid location")
	// ErrAuthModeUnsupported means the decoder has no authentication mode
	// named by spec.authentication.mode.
	ErrAuthModeUnsupported = errors.New("authentication mode unsupported")
	// ErrCredentialsMissing means the credentials lack a key the
	// authentication mode requires.
	ErrCredentialsMissing = errors.New("credentials missing")
)
