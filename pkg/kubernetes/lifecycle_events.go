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

package kubernetes

import (
	"context"
	"fmt"
	"os"

	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/events"
)

// RecordLifecycleEvent records r as a lifecycle event.
func (k *Kubernetes) RecordLifecycleEvent(ctx context.Context, r events.Record) error {
	reportingInstance, err := os.Hostname()
	if err != nil {
		reportingInstance = "everest-server"
	}
	return events.NewEmitter(k.k8sClient, reportingInstance).Emit(ctx, r)
}

// ListLifecycleEvents lists the lifecycle events (see pkg/events) in namespace, or in
// all namespaces when namespace is empty.
func (k *Kubernetes) ListLifecycleEvents(ctx context.Context, namespace string, limit int64, continueToken string) (*eventsv1.EventList, error) {
	list := &eventsv1.EventList{}
	err := k.k8sClient.List(ctx, list,
		ctrlclient.InNamespace(namespace),
		ctrlclient.MatchingLabels{events.LabelLifecycleEvent: events.LabelLifecycleEventValue},
		ctrlclient.Limit(limit),
		ctrlclient.Continue(continueToken),
	)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// WatchLifecycleEvents watches lifecycle events recorded after resourceVersion, or from
// now on when it is empty. The caller reconnects from its last received resourceVersion
// if the watch closes.
//
//nolint:ireturn // watch.Interface is the client-go streaming contract
func (k *Kubernetes) WatchLifecycleEvents(ctx context.Context, namespace, resourceVersion string) (watch.Interface, error) {
	if k.watchClient == nil {
		return nil, fmt.Errorf("watch client is not configured")
	}
	if resourceVersion == "" {
		list := &eventsv1.EventList{}
		if err := k.watchClient.List(ctx, list,
			ctrlclient.InNamespace(namespace),
			ctrlclient.MatchingLabels{events.LabelLifecycleEvent: events.LabelLifecycleEventValue},
			ctrlclient.Limit(1),
		); err != nil {
			return nil, fmt.Errorf("get the current resourceVersion of the events: %w", err)
		}
		resourceVersion = list.ResourceVersion
	}
	opts := &metav1.ListOptions{
		ResourceVersion:     resourceVersion,
		AllowWatchBookmarks: true,
		LabelSelector:       labels.SelectorFromSet(labels.Set{events.LabelLifecycleEvent: events.LabelLifecycleEventValue}).String(),
	}
	return k.watchClient.Watch(ctx, &eventsv1.EventList{},
		ctrlclient.InNamespace(namespace),
		&ctrlclient.ListOptions{Raw: opts},
	)
}
