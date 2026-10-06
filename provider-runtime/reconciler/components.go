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

package reconciler

import (
	"context"
	"fmt"
	"time"

	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

const podPermissionRetry = 10 * time.Second

// watchPods caches the pods labelled for this provider and reconciles their
// Instance when the component counts may change. It runs once the manager has
// started, and keeps the pod cache apart from the manager's so the provider's
// own pod reads stay unfiltered. Without list and watch permission on pods,
// status.components stays empty.
func (r *ProviderReconciler) watchPods(ctx context.Context, c ctrlcontroller.Controller) error {
	logger := log.FromContext(ctx)

	allowed, err := canListAndWatchPods(ctx, r.Client)
	for err != nil {
		logger.Error(err, "Failed to check the permissions on pods, retrying")
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(podPermissionRetry):
		}
		allowed, err = canListAndWatchPods(ctx, r.Client)
	}
	if !allowed {
		logger.Info("Not reporting status.components: the provider may not list and watch pods")
		return nil
	}

	pods, err := cache.New(r.manager.GetConfig(), cache.Options{
		Scheme:               r.manager.GetScheme(),
		Mapper:               r.manager.GetRESTMapper(),
		DefaultLabelSelector: labels.SelectorFromSet(labels.Set{controller.ProviderLabel: r.provider.Name()}),
		DefaultTransform:     cache.TransformStripManagedFields(),
	})
	if err != nil {
		return fmt.Errorf("failed to create the pod cache: %w", err)
	}
	go func() {
		if err := pods.Start(ctx); err != nil {
			logger.Error(err, "Pod cache stopped")
		}
	}()
	src := source.Kind(pods, &corev1.Pod{},
		handler.TypedEnqueueRequestsFromMapFunc(instanceOfPod),
		predicate.TypedFuncs[*corev1.Pod]{UpdateFunc: podCountChanged},
	)
	if err := c.Watch(src); err != nil {
		return fmt.Errorf("failed to watch pods: %w", err)
	}
	r.pods.Store(client.Reader(pods))

	return nil
}

func canListAndWatchPods(ctx context.Context, c client.Client) (bool, error) {
	for _, verb := range []string{"list", "watch"} {
		review := &authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{Verb: verb, Resource: "pods"},
			},
		}
		if err := c.Create(ctx, review); err != nil {
			return false, fmt.Errorf("failed to check the %s permission on pods: %w", verb, err)
		}
		if !review.Status.Allowed {
			return false, nil
		}
	}

	return true, nil
}

func instanceOfPod(_ context.Context, pod *corev1.Pod) []reconcile.Request {
	name := pod.Labels[controller.InstanceLabel]
	if name == "" {
		return nil
	}

	return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: pod.Namespace, Name: name}}}
}

func podCountChanged(e event.TypedUpdateEvent[*corev1.Pod]) bool {
	return podCounted(e.ObjectOld) != podCounted(e.ObjectNew) || podReady(e.ObjectOld) != podReady(e.ObjectNew)
}

// setComponentStatuses counts the pods of each labelled component into
// in.Status.Components. Until the pod watch runs it leaves them untouched.
func (r *ProviderReconciler) setComponentStatuses(ctx context.Context, in *v1alpha1.Instance, components []string) {
	pods, ok := r.pods.Load().(client.Reader)
	if !ok {
		return
	}

	statuses := make([]v1alpha1.ComponentStatus, 0, len(components))
	for _, name := range components {
		selector := labels.SelectorFromSet(labels.Set{controller.InstanceLabel: in.Name, controller.ComponentLabel: name})
		list := &corev1.PodList{}
		if err := pods.List(ctx, list, client.InNamespace(in.Namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
			log.FromContext(ctx).Error(err, "Failed to list the component's pods", "component", name)
			return
		}

		var replicas, ready int32
		for i := range list.Items {
			if !podCounted(&list.Items[i]) {
				continue
			}
			replicas++
			if podReady(&list.Items[i]) {
				ready++
			}
		}
		statuses = append(statuses, v1alpha1.ComponentStatus{
			Name:          name,
			Selector:      selector.String(),
			Replicas:      &replicas,
			ReadyReplicas: &ready,
		})
	}
	in.Status.Components = statuses
}

// podCounted reports whether the pod is neither terminating nor terminated.
func podCounted(pod *corev1.Pod) bool {
	return pod.DeletionTimestamp == nil &&
		pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed
}

func podReady(pod *corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}

	return false
}
