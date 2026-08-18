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
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDeployment returns k8s deployment that matches the criteria.
func (k *Kubernetes) GetDeployment(ctx context.Context, key ctrlclient.ObjectKey) (*appsv1.Deployment, error) {
	result := &appsv1.Deployment{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateDeployment updates a deployment and returns the updated object.
func (k *Kubernetes) UpdateDeployment(ctx context.Context, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	if err := k.k8sClient.Update(ctx, deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

// ListDeployments returns a list of deployments that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListDeployments(ctx context.Context, opts ...ctrlclient.ListOption) (*appsv1.DeploymentList, error) {
	result := &appsv1.DeploymentList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteDeployment deletes a deployment that matches the criteria.
func (k *Kubernetes) DeleteDeployment(ctx context.Context, obj *appsv1.Deployment) error {
	return k.k8sClient.Delete(ctx, obj)
}

// RestartDeployment restarts deployment that matches the criteria.
func (k *Kubernetes) RestartDeployment(ctx context.Context, key ctrlclient.ObjectKey) error {
	// Get the Deployment and add restart annotation to pod template.
	// We retry this operatation since there may be update conflicts.
	var b backoff.BackOff
	b = backoff.NewConstantBackOff(backoffInterval)
	b = backoff.WithMaxRetries(b, backoffMaxRetries)
	b = backoff.WithContext(b, ctx)
	if err := backoff.Retry(
		func() error {
			// Get the deployment.
			deployment, err := k.GetDeployment(ctx, key)
			if err != nil {
				return err
			}
			// Set restart annotation.
			annotations := deployment.Spec.Template.Annotations
			if annotations == nil {
				annotations = make(map[string]string)
			}
			annotations[deploymentRestartAnnotation] = time.Now().Format(time.RFC3339)
			deployment.Spec.Template.SetAnnotations(annotations)
			// Update deployment.
			if _, err := k.UpdateDeployment(ctx, deployment); err != nil {
				return err
			}
			return nil
		}, b,
	); err != nil {
		return errors.Join(err, fmt.Errorf("cannot add restart annotation to deployment='%s' in namespace='%s'", key.Name, key.Namespace))
	}
	// Wait for pods to be ready.
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		deployment, err := k.GetDeployment(ctx, key)
		if err != nil {
			return false, err
		}
		ready := deployment.Status.ReadyReplicas == deployment.Status.Replicas &&
			deployment.Status.Replicas == deployment.Status.UpdatedReplicas &&
			deployment.Status.UnavailableReplicas == 0 &&
			deployment.GetGeneration() == deployment.Status.ObservedGeneration

		return ready, nil
	})
}

// WaitForRollout waits for rollout of deployment that matches the criteria.
func (k *Kubernetes) WaitForRollout(ctx context.Context, key ctrlclient.ObjectKey) error {
	rolloutComplete := func(ctx context.Context) (bool, error) {
		deployment, err := k.GetDeployment(ctx, key)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// Waiting for Deployment to appear
				return false, nil
			}
			return false, err
		}

		if deployment.Generation <= deployment.Status.ObservedGeneration {
			cond := deploymentutil.GetDeploymentCondition(deployment.Status, appsv1.DeploymentProgressing)
			if cond != nil && cond.Reason == deploymentutil.TimedOutReason {
				return false, errors.New("progress deadline exceeded")
			}
			if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
				// Waiting for Deployment to roll out. Not all replicas have been updated
				return false, nil
			}
			if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
				// Waiting for Deployment to roll out. Old replicas are pending termination
				return false, nil
			}
			if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
				// Waiting for Deployment to roll out. Not all updated replicas are available
				return false, nil
			}
			// Deployment successfully rolled out
			return true, nil
		}
		// Waiting for Deployment to roll out: waiting for deployment spec update to be observed
		return false, nil
	}
	return wait.PollUntilContextCancel(ctx, time.Second, true, rolloutComplete)
}
