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
	watchtools "k8s.io/client-go/tools/watch"
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
func (k *Kubernetes) ListLifecycleEvents(ctx context.Context, namespace string) (*eventsv1.EventList, error) {
	list := &eventsv1.EventList{}
	err := k.k8sClient.List(ctx, list,
		ctrlclient.InNamespace(namespace),
		ctrlclient.MatchingLabels{events.LabelLifecycleEvent: events.LabelLifecycleEventValue},
	)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// WatchLifecycleEvents watches the lifecycle events in namespace, or in all namespaces
// when namespace is empty, recorded after resourceVersion, or from now on when it is
// empty. The watch resumes on its own when the API server closes it. When
// resourceVersion is too old to resume from, it sends an error event with a 410 status
// and stops.
//
//nolint:ireturn // watch.Interface is the client-go streaming contract
func (k *Kubernetes) WatchLifecycleEvents(ctx context.Context, namespace, resourceVersion string) (watch.Interface, error) {
	wc, err := ctrlclient.NewWithWatch(k.restConfig, ctrlclient.Options{Scheme: k.k8sClient.Scheme()})
	if err != nil {
		return nil, fmt.Errorf("create watch client: %w", err)
	}
	if resourceVersion == "" {
		// The retrying watcher needs a starting point: the current resourceVersion of the
		// events, so that only events recorded from now on are sent.
		list := &eventsv1.EventList{}
		if err := wc.List(ctx, list, ctrlclient.InNamespace(namespace), ctrlclient.Limit(1)); err != nil {
			return nil, fmt.Errorf("get the current resourceVersion of the events: %w", err)
		}
		resourceVersion = list.ResourceVersion
	}
	return watchtools.NewRetryWatcherWithContext(ctx, resourceVersion, lifecycleEventsWatcher{client: wc, namespace: namespace})
}

type lifecycleEventsWatcher struct {
	client    ctrlclient.WithWatch
	namespace string
}

//nolint:ireturn // implements cache.WatcherWithContext
func (w lifecycleEventsWatcher) WatchWithContext(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.LabelSelector = labels.SelectorFromSet(labels.Set{events.LabelLifecycleEvent: events.LabelLifecycleEventValue}).String()
	return w.client.Watch(ctx, &eventsv1.EventList{}, ctrlclient.InNamespace(w.namespace), &ctrlclient.ListOptions{Raw: &opts})
}
