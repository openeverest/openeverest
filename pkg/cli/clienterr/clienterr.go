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

// Package clienterr provides helpers for reading error bodies from generated
// OpenAPI client responses.
package clienterr

import "github.com/openeverest/openeverest/v2/client"

// Message returns the first message found among the given candidates, or false
// if none carry one. Callers pass the error fields their operation declares in
// the OpenAPI spec (typically resp.JSON400, resp.JSON500, resp.JSONDefault) —
// when an operation gains a declared status, its call sites must be updated to
// pass the new field, or the message will be silently dropped again.
func Message(candidates ...*client.Error) (string, bool) {
	for _, c := range candidates {
		if c != nil && c.Message != nil {
			return *c.Message, true
		}
	}
	return "", false
}
