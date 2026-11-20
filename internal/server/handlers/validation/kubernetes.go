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

// Package validation provides the validation handler.
package validation

import (
	"context"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

func (h *validateHandler) GetUserPermissions(ctx context.Context) (*api.UserPermissions, error) {
	return h.next.GetUserPermissions(ctx)
}

func (h *validateHandler) GetSettings(ctx context.Context) (*api.Settings, error) {
	return h.next.GetSettings(ctx)
}
