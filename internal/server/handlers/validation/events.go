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

	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// ListEvents proxies the request to the next handler.
func (h *validateHandler) ListEvents(ctx context.Context, cluster string, namespaces []string) (*eventsv1.EventList, error) {
	return h.next.ListEvents(ctx, cluster, namespaces)
}

// WatchEvents proxies the request to the next handler.
//
//nolint:ireturn // watch.Interface is the client-go streaming contract
func (h *validateHandler) WatchEvents(
	ctx context.Context, cluster string, namespaces []string, resourceVersion string,
) (watch.Interface, error) {
	return h.next.WatchEvents(ctx, cluster, namespaces, resourceVersion)
}
