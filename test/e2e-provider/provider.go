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

package main

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

const (
	providerName      = "core-e2e-test"
	controlAnnotation = "test.openeverest.io/control"
	controlFail       = "fail"
	controlWait       = "wait"
	workloadImage     = "registry.k8s.io/pause:3.10"
)

var _ controller.ProviderInterface = (*testProvider)(nil)
var _ controller.BackupProvider = (*testProvider)(nil)

type testProvider struct {
	controller.BaseProvider
}

func newTestProvider() *testProvider {
	return &testProvider{BaseProvider: controller.BaseProvider{
		ProviderName: providerName,
		WatchConfigs: []controller.WatchConfig{
			controller.WatchOwned(&corev1.Pod{}),
			controller.WatchOwned(&backupv1alpha1.Restore{}),
		},
	}}
}

func (p *testProvider) Validate(_ *controller.Context) error {
	return nil
}

func (p *testProvider) Sync(c *controller.Context) error {
	if _, err := c.ReconcileDataSource(); err != nil {
		return fmt.Errorf("reconcile test data source: %w", err)
	}
	labels := map[string]string{"app.kubernetes.io/name": providerName, "app.kubernetes.io/instance": c.Name()}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: c.Name(), Namespace: c.Namespace(), Labels: labels},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "workload", Image: workloadImage}}},
	}
	if err := c.Apply(pod); err != nil {
		return fmt.Errorf("apply test workload: %w", err)
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: c.Name(), Namespace: c.Namespace(), Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    []corev1.ServicePort{{Name: "test", Port: 1}},
		},
	}
	if err := c.Apply(service); err != nil {
		return fmt.Errorf("apply test service: %w", err)
	}
	return nil
}

func (p *testProvider) Status(c *controller.Context) (controller.Status, error) {
	switch c.Annotations()[controlAnnotation] {
	case controlFail:
		return controller.Failed("test failure requested"), nil
	case controlWait:
		return controller.Initializing("test is waiting"), nil
	}
	if source := c.GetDataSourceStatus(); source != nil && !source.Done {
		return controller.Restoring(source.Message), nil
	}
	if source := c.GetDataSourceStatus(); source != nil && source.State == controller.DataSourceStateFailed {
		return controller.Failed(source.Message), nil
	}
	pod := &corev1.Pod{}
	if err := c.Get(pod, c.Name()); err != nil {
		if apierrors.IsNotFound(err) {
			return controller.Provisioning("waiting for test workload"), nil
		}
		return controller.Status{}, fmt.Errorf("get test workload: %w", err)
	}
	if !podReady(pod) {
		return controller.Initializing("waiting for test workload to become ready"), nil
	}
	host := fmt.Sprintf("%s.%s.svc", c.Name(), c.Namespace())
	return controller.ReadyWithConnectionDetails(controller.ConnectionDetails{
		Type: "test", Provider: providerName, Host: host, Port: "1", URI: "test://" + host + ":1",
	}), nil
}

func podReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (p *testProvider) Cleanup(_ *controller.Context) error {
	return nil
}
