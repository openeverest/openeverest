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

package k8s

import (
	"context"
	"fmt"
	"slices"

	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// ListEvents lists the lifecycle events in namespaces, or in all namespaces when empty.
func (h *k8sHandler) ListEvents(ctx context.Context, _ string, namespaces []string) (*eventsv1.EventList, error) {
	list, err := h.kubeConnector.ListLifecycleEvents(ctx, watchedNamespace(namespaces))
	if err != nil {
		return nil, fmt.Errorf("failed to list lifecycle events: %w", err)
	}
	if len(namespaces) > 1 {
		list.Items = slices.DeleteFunc(list.Items, func(ev eventsv1.Event) bool {
			return !slices.Contains(namespaces, ev.Namespace)
		})
	}
	return list, nil
}

// WatchEvents streams the lifecycle events in namespaces, or in all namespaces when empty.
//
//nolint:ireturn // watch.Interface is the client-go streaming contract
func (h *k8sHandler) WatchEvents(
	ctx context.Context, _ string, namespaces []string, resourceVersion string,
) (watch.Interface, error) {
	w, err := h.kubeConnector.WatchLifecycleEvents(ctx, watchedNamespace(namespaces), resourceVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to watch lifecycle events: %w", err)
	}
	if len(namespaces) <= 1 {
		return w, nil
	}
	return filterNamespaces(w, namespaces), nil
}

// filterNamespaces keeps the events of w in namespaces. Bookmarks and errors are kept:
// a bookmark carries only a resourceVersion, in an otherwise empty Event.
//
//nolint:ireturn // watch.Interface is the client-go streaming contract
func filterNamespaces(w watch.Interface, namespaces []string) watch.Interface {
	return watch.Filter(w, func(in watch.Event) (watch.Event, bool) {
		ev, ok := in.Object.(*eventsv1.Event)
		if !ok || in.Type == watch.Bookmark || in.Type == watch.Error {
			return in, true
		}
		return in, slices.Contains(namespaces, ev.Namespace)
	})
}

// watchedNamespace returns the namespace to list or watch: the only one requested, or
// all of them, filtered afterwards, when several are.
func watchedNamespace(namespaces []string) string {
	if len(namespaces) == 1 {
		return namespaces[0]
	}
	return ""
}
