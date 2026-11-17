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

package rbac

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListEvents returns the lifecycle events the caller can read.
func (h *rbacHandler) ListEvents(ctx context.Context, cluster string, namespaces []string, limit int64, continueToken string) (*eventsv1.EventList, error) {
	list, err := h.next.ListEvents(ctx, cluster, namespaces, limit, continueToken)
	if err != nil {
		return nil, fmt.Errorf("ListEvents failed: %w", err)
	}
	filtered := make([]eventsv1.Event, 0, len(list.Items))
	for i := range list.Items {
		allowed, err := h.canReadEvent(ctx, cluster, &list.Items[i])
		if err != nil {
			return nil, err
		}
		if allowed {
			filtered = append(filtered, list.Items[i])
		}
	}
	list.Items = filtered
	return list, nil
}

// WatchEvents streams the lifecycle events the caller can read. Bookmark and error
// events carry no event data and are passed through.
//
//nolint:ireturn // watch.Interface is the client-go streaming contract
func (h *rbacHandler) WatchEvents(
	ctx context.Context, cluster string, namespaces []string, resourceVersion string,
) (watch.Interface, error) {
	w, err := h.next.WatchEvents(ctx, cluster, namespaces, resourceVersion)
	if err != nil {
		return nil, fmt.Errorf("WatchEvents failed: %w", err)
	}
	return watch.Filter(w, func(in watch.Event) (watch.Event, bool) {
		ev, ok := in.Object.(*eventsv1.Event)
		if !ok || !carriesEvent(in.Type) {
			return in, true
		}
		allowed, err := h.canReadEvent(ctx, cluster, ev)
		if err != nil {
			h.log.Errorf("failed to authorize a lifecycle event: %v", err)
			return in, false
		}
		return in, allowed
	}), nil
}

// carriesEvent reports whether a watch event of type t carries a lifecycle event.
// Bookmarks carry only a resourceVersion, in an otherwise empty object of the watched
// kind.
func carriesEvent(t watch.EventType) bool {
	return t == watch.Added || t == watch.Modified || t == watch.Deleted
}

// eventResources maps the kinds lifecycle events are recorded for to their RBAC
// resource.
//
//nolint:gochecknoglobals // immutable lookup table
var eventResources = map[schema.GroupKind]string{
	{Group: corev1alpha1.GroupVersion.Group, Kind: "Instance"}:                    rbac.ResourceInstances,
	{Group: backupv1alpha1.GroupVersion.Group, Kind: "Backup"}:                    rbac.ResourceBackups,
	{Group: backupv1alpha1.GroupVersion.Group, Kind: "Restore"}:                   rbac.ResourceRestores,
	{Group: backupv1alpha1.GroupVersion.Group, Kind: "BackupImport"}:              rbac.ResourceBackupImports,
	{Group: backupv1alpha1.GroupVersion.Group, Kind: "BackupStorage"}:             rbac.ResourceBackupStorages,
	{Group: monitoringv1alpha1.GroupVersion.Group, Kind: "MonitoringDestination"}: rbac.ResourceMonitoringDestinations,
	{Group: extensionsv1alpha1.GroupVersion.Group, Kind: "Plugin"}:                rbac.ResourcePlugins,
	{Group: extensionsv1alpha1.GroupVersion.Group, Kind: "InstalledExtension"}:    rbac.ResourceInstalledExtensions,
	{Group: corev1.GroupName, Kind: "Namespace"}:                                  rbac.ResourceNamespaces,
}

// eventResource returns the RBAC resource of the object ref refers to.
func eventResource(ref corev1.ObjectReference) (string, bool) {
	gv, err := schema.ParseGroupVersion(ref.APIVersion)
	if err != nil {
		return "", false
	}
	resource, ok := eventResources[gv.WithKind(ref.Kind).GroupKind()]
	return resource, ok
}

// canReadEvent reports whether the caller can read ev: read-events on the object
// the event is about, or read on platform-events for logins and logouts. Events
// about kinds without a resource are never shown.
func (h *rbacHandler) canReadEvent(ctx context.Context, cluster string, ev *eventsv1.Event) (bool, error) {
	ref := ev.Regarding
	resource, action, object := rbac.ResourcePlatformEvents, rbac.ActionRead, "*"
	if ref.Kind != "User" {
		var ok bool
		if resource, ok = eventResource(ref); !ok {
			return false, nil
		}
		action = rbac.ActionReadEvents
		object = rbac.ClusterNamespacedObjectName(cluster, ref.Namespace, ref.Name)
		if ref.Namespace == "" {
			object = rbac.ClusterObjectName(cluster, ref.Name)
		}
	}
	err := h.enforce(ctx, resource, action, object)
	if errors.Is(err, ErrInsufficientPermissions) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("enforce failed: %w", err)
	}
	return true, nil
}
