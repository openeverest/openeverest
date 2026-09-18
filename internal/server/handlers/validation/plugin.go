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

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListPlugins proxies the request to the next handler.
func (h *validateHandler) ListPlugins(ctx context.Context, cluster string) (api.PluginDescriptorList, error) {
	return h.next.ListPlugins(ctx, cluster)
}

// GetPluginContext proxies the request to the next handler.
func (h *validateHandler) GetPluginContext(ctx context.Context, cluster string) (*api.PluginContext, error) {
	return h.next.GetPluginContext(ctx, cluster)
}
