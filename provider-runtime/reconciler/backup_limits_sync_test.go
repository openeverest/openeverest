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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// syncTrackingProvider is a providerAdapter stub that records whether Sync
// and Status were invoked, so tests can assert the reconciler still
// converges the engine even when backup config validation fails.
type syncTrackingProvider struct {
	syncCalled   bool
	statusCalled bool
}

func (p *syncTrackingProvider) Name() string                       { return "test-provider" }
func (p *syncTrackingProvider) Types() func(*runtime.Scheme) error { return nil }
func (p *syncTrackingProvider) Validate(*controller.Context) error { return nil }
func (p *syncTrackingProvider) Cleanup(*controller.Context) error  { return nil }
func (p *syncTrackingProvider) Sync(*controller.Context) error {
	p.syncCalled = true
	return nil
}

func (p *syncTrackingProvider) Status(*controller.Context) (controller.Status, error) {
	p.statusCalled = true
	return controller.Ready(), nil
}

func TestReconcile_BackupLimitsViolation_StillRunsSync(t *testing.T) {
	t.Parallel()
	scheme := newTestScheme()

	maxStorages := int32(1)
	backupClass := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "test-class"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				Limits: &backupv1alpha1.BackupClassLimits{
					MaxStorages: &maxStorages,
				},
			},
		},
	}

	provider := &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: "test-provider"},
	}

	// Two storages violates the class's MaxStorages: 1 limit. The finalizer
	// and provider label are pre-set so Reconcile runs the real validate/
	// sync/status logic on the first call, instead of only adding them and
	// requeuing (its behavior for a brand new Instance).
	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "my-instance",
			Namespace:  "default",
			Finalizers: []string{finalizerName},
			Labels:     map[string]string{controller.ProviderLabel: "test-provider"},
		},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef: common.ObjectRef{Name: "test-provider"},
			Backup: &corev1alpha1.InstanceBackupSpec{
				Enabled:  true,
				ClassRef: common.ObjectRef{Name: "test-class"},
				Storages: []corev1alpha1.InstanceBackupStorage{
					{StorageRef: common.ObjectRef{Name: "storage-a"}},
					{StorageRef: common.ObjectRef{Name: "storage-b"}},
				},
			},
		},
	}

	// Reconcile writes status via the Status subresource client, which the
	// fake client only serves correctly for types registered via
	// WithStatusSubresource — hence a local builder instead of the package's
	// newFakeClient (whose existing callers never exercise that path).
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&corev1alpha1.Instance{}).
		WithObjects(backupClass, provider, instance).
		Build()
	testProvider := &syncTrackingProvider{}
	r := &ProviderReconciler{
		provider: testProvider,
		Client:   fakeClient,
	}

	req := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(instance)}

	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	assert.True(t, testProvider.syncCalled,
		"Sync() must still run when only the backup config is invalid — the engine itself is healthy")
	assert.True(t, testProvider.statusCalled,
		"Status() must still run when only the backup config is invalid")

	var got corev1alpha1.Instance
	require.NoError(t, fakeClient.Get(context.Background(), req.NamespacedName, &got))

	found := false
	for _, c := range got.Status.Conditions {
		if c.Type == corev1alpha1.ConditionBackupConfigured {
			found = true
			assert.Equal(t, metav1.ConditionFalse, c.Status)
			assert.Equal(t, controller.LimitsExceededReason, c.Reason)
		}
	}
	assert.True(t, found, "BackupConfigured condition must be set to False, not silently dropped")
}
