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

package minio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// newTestScheme mirrors provider-runtime/reconciler's own newTestScheme
// (cascade_delete_test.go): the real reconciler registers corev1,
// corev1alpha1, and backupv1alpha1 by default (see reconciler/provider.go),
// plus whatever the provider's own Types() adds.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, corev1alpha1.AddToScheme(s))
	require.NoError(t, backupv1alpha1.AddToScheme(s))
	require.NoError(t, AddToScheme(s))
	return s
}

// testProvider returns a Provider CR shaped like manifest/provider.yaml,
// with one extra non-default version so version-resolution can be exercised.
func testProvider() *corev1alpha1.Provider {
	return &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: "minio"},
		Spec: corev1alpha1.ProviderSpec{
			ComponentTypes: map[string]corev1alpha1.ComponentType{
				serverComponentType: {
					Versions: []corev1alpha1.ComponentVersion{
						{Version: "RELEASE.2025-04-08T15-41-24Z", Image: "quay.io/minio/minio:RELEASE.2025-04-08T15-41-24Z", Default: true},
						{Version: "RELEASE.2024-01-01T00-00-00Z", Image: "quay.io/minio/minio:RELEASE.2024-01-01T00-00-00Z"},
					},
				},
			},
		},
	}
}

// testInstanceName is the name every test fixture Instance/Tenant/
// BackupStorage in this file shares; each test builds its own isolated fake
// client, so a shared name causes no cross-test interference.
const testInstanceName = "minio-poc"

func testInstance(mutate func(*corev1alpha1.Instance)) *corev1alpha1.Instance {
	in := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: testInstanceName, Namespace: "default"},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef: common.ObjectRef{Name: "minio"},
		},
	}
	if mutate != nil {
		mutate(in)
	}
	return in
}

func newTestContext(t *testing.T, c client.Client, in *corev1alpha1.Instance) *controller.Context {
	t.Helper()
	return controller.NewContext(context.Background(), c, in, "minio")
}

func TestResolveServerImage(t *testing.T) {
	t.Parallel()

	spec := &testProvider().Spec

	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"empty version resolves to the default bundle's image", "", "quay.io/minio/minio:RELEASE.2025-04-08T15-41-24Z"},
		{"known non-default version resolves to its own image", "RELEASE.2024-01-01T00-00-00Z", "quay.io/minio/minio:RELEASE.2024-01-01T00-00-00Z"},
		{"unknown version falls back to the default bundle's image", "RELEASE.9999-01-01T00-00-00Z", "quay.io/minio/minio:RELEASE.2025-04-08T15-41-24Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, resolveServerImage(spec, tt.version))
		})
	}

	t.Run("componentType missing entirely falls back to the operator's own default", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, defaultMinIOImage, resolveServerImage(&corev1alpha1.ProviderSpec{}, ""))
	})
}

func TestSync_RendersTenantFromInstanceDefaults(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	instance := testInstance(nil)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testProvider(), instance).Build()
	ctx := newTestContext(t, c, instance)

	p := &Provider{}
	require.NoError(t, p.Sync(ctx))

	tenant := &Tenant{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "minio-poc", Namespace: "default"}, tenant))

	assert.Equal(t, "quay.io/minio/minio:RELEASE.2025-04-08T15-41-24Z", tenant.Spec.Image)
	require.Len(t, tenant.Spec.Pools, 1)
	assert.EqualValues(t, 1, tenant.Spec.Pools[0].Servers, "defaults to 1 replica when the Instance sets no server component")
	assert.EqualValues(t, 1, tenant.Spec.Pools[0].VolumesPerServer)
	require.NotNil(t, tenant.Spec.Pools[0].VolumeClaimTemplate)
	assert.Equal(t, defaultVolumeSize, tenant.Spec.Pools[0].VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage])
	require.NotNil(t, tenant.Spec.RequestAutoCert)
	assert.False(t, *tenant.Spec.RequestAutoCert)
	require.NotNil(t, tenant.Spec.Configuration)
	assert.Equal(t, "minio-poc"+configSecretSuffix, tenant.Spec.Configuration.Name)
	require.Len(t, tenant.Spec.Buckets, 1)
	assert.Equal(t, "minio-poc"+backupBucketSuffix, tenant.Spec.Buckets[0].Name)

	// The credentials Secret referenced by Configuration must actually
	// exist, own a config.env the MinIO Operator can parse, and expose the
	// plain rootUser/rootPassword keys ensureBackupStorage relies on.
	secret := &corev1.Secret{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: tenant.Spec.Configuration.Name, Namespace: "default"}, secret))
	assert.NotEmpty(t, secret.Data["config.env"])
	assert.Equal(t, rootUser, string(secret.Data["rootUser"]))
	assert.NotEmpty(t, secret.Data["rootPassword"])
}

func TestSync_HonorsExplicitReplicasAndStorage(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	replicas := int32(4)
	storageClass := "fast-ssd"
	instance := testInstance(func(in *corev1alpha1.Instance) {
		in.Spec.Components = map[string]corev1alpha1.ComponentSpec{
			"server": {
				Type:     serverComponentType,
				Replicas: &replicas,
				Storage: &corev1alpha1.Storage{
					Size:         resource.MustParse("10Gi"),
					StorageClass: &storageClass,
				},
			},
		}
	})
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testProvider(), instance).Build()
	ctx := newTestContext(t, c, instance)

	require.NoError(t, (&Provider{}).Sync(ctx))

	tenant := &Tenant{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "minio-poc", Namespace: "default"}, tenant))

	require.Len(t, tenant.Spec.Pools, 1)
	assert.EqualValues(t, 4, tenant.Spec.Pools[0].Servers)
	assert.Equal(t, resource.MustParse("10Gi"), tenant.Spec.Pools[0].VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage])
	require.NotNil(t, tenant.Spec.Pools[0].VolumeClaimTemplate.Spec.StorageClassName)
	assert.Equal(t, "fast-ssd", *tenant.Spec.Pools[0].VolumeClaimTemplate.Spec.StorageClassName)
}

// TestSync_CredentialsSecretIsStableAcrossReconciles is a regression test:
// ensureCredentialsSecret must only generate the root password once.
// Calling Sync again (as the reconciler routinely does) must not desync the
// stored password from whatever the MinIO server already booted with.
func TestSync_CredentialsSecretIsStableAcrossReconciles(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	instance := testInstance(nil)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testProvider(), instance).Build()
	ctx := newTestContext(t, c, instance)

	require.NoError(t, (&Provider{}).Sync(ctx))
	first := &corev1.Secret{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "minio-poc" + configSecretSuffix, Namespace: "default"}, first))

	require.NoError(t, (&Provider{}).Sync(ctx))
	second := &corev1.Secret{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "minio-poc" + configSecretSuffix, Namespace: "default"}, second))

	assert.Equal(t, first.Data["rootPassword"], second.Data["rootPassword"])
	assert.Equal(t, first.Data["config.env"], second.Data["config.env"])
}

func TestStatus_TenantNotFound_ReturnsProvisioning(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	instance := testInstance(nil)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testProvider(), instance).Build()
	ctx := newTestContext(t, c, instance)

	status, err := (&Provider{}).Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, corev1alpha1.InstancePhaseProvisioning, status.Phase)
}

func TestStatus_TenantNotInitialized_ReturnsProvisioning(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	instance := testInstance(nil)
	tenant := &Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: testInstanceName, Namespace: "default"},
		Status:     TenantStatus{CurrentState: "empty tenant credentials"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testProvider(), instance, tenant).Build()
	ctx := newTestContext(t, c, instance)

	status, err := (&Provider{}).Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, corev1alpha1.InstancePhaseProvisioning, status.Phase)

	// A not-yet-ready Tenant must not advertise a BackupStorage: nothing
	// should reference a backup target that can't accept writes yet.
	bs := &backupv1alpha1.BackupStorage{}
	err = c.Get(context.Background(), client.ObjectKey{Name: "minio-poc", Namespace: "default"}, bs)
	assert.Error(t, err, "no BackupStorage should be created before the Tenant is Initialized")
}

// TestStatus_TenantInitialized_ReturnsReadyAndRegistersBackupStorage is a
// regression test for the backup bridge: once the Tenant is Initialized,
// Status must both report Ready and register a BackupStorage pointing at
// this Instance's own Tenant, reusing its root credentials.
func TestStatus_TenantInitialized_ReturnsReadyAndRegistersBackupStorage(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme(t)
	instance := testInstance(nil)
	credsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "minio-poc" + configSecretSuffix, Namespace: "default"},
		Data: map[string][]byte{
			"rootUser":     []byte("minio"),
			"rootPassword": []byte("s3cr3t-test-password"),
		},
	}
	tenant := &Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: testInstanceName, Namespace: "default"},
		Status:     TenantStatus{CurrentState: tenantStatusInitialized, AvailableReplicas: 1},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testProvider(), instance, tenant, credsSecret).Build()
	ctx := newTestContext(t, c, instance)

	status, err := (&Provider{}).Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, corev1alpha1.InstancePhaseReady, status.Phase)

	bs := &backupv1alpha1.BackupStorage{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "minio-poc", Namespace: "default"}, bs))
	assert.Equal(t, backupv1alpha1.BackupStorageTypeS3, bs.Spec.Type)
	require.NotNil(t, bs.Spec.S3)
	assert.Equal(t, "minio-poc"+backupBucketSuffix, bs.Spec.S3.Bucket)
	assert.Equal(t, "http://minio-poc-hl.default.svc.cluster.local:9000", bs.Spec.S3.EndpointURL)
	assert.Equal(t, "minio-poc"+backupCredsSecretSuffix, bs.Spec.S3.CredentialsSecretRef.Name)

	backupSecret := &corev1.Secret{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: "minio-poc" + backupCredsSecretSuffix, Namespace: "default"}, backupSecret))
	assert.Equal(t, "minio", string(backupSecret.Data["AWS_ACCESS_KEY_ID"]))
	assert.Equal(t, "s3cr3t-test-password", string(backupSecret.Data["AWS_SECRET_ACCESS_KEY"]))
}
