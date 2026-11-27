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

package bootstrap_test

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/controller/bootstrap"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

const namespace = "everest-system"

func newFakeClient(t *testing.T, objs ...client.Object) client.Client { //nolint:ireturn // the fake is used through the client interface
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithTypeConverters(managedfields.NewDeducedTypeConverter()).
		WithObjects(objs...).
		Build()
}

func TestRunCreatesMissingDefaults(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := newFakeClient(t, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, UID: "ns-uid"}})

	b := &bootstrap.Bootstrap{Client: c, Namespace: namespace}
	require.NoError(t, b.Run(ctx))

	jwt := &corev1.Secret{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: common.EverestJWTSecretName}, jwt))
	block, _ := pem.Decode(jwt.Data["id_rsa"])
	require.NotNil(t, block, "id_rsa holds no PEM block")
	_, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err, "the API server reads a PKCS #1 RSA key")

	// The API server and everestctl read the account through the accounts
	// client: the admin can log in, and its password is still the initial,
	// plain-text one.
	accounts := kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(c).Accounts()
	admin, err := accounts.Get(ctx, common.EverestAdminUser)
	require.NoError(t, err)
	assert.True(t, admin.Enabled)
	assert.True(t, admin.HasCapability("login"))
	require.NoError(t, accounts.Verify(ctx, common.EverestAdminUser, admin.PasswordHash))
	secure, err := accounts.IsSecure(ctx, common.EverestAdminUser)
	require.NoError(t, err)
	assert.False(t, secure)

	rbac := &corev1.ConfigMap{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: common.EverestRBACConfigMapName}, rbac))
	assert.Equal(t, map[string]string{"enabled": "false", "policy.csv": "g, admin, role:admin\n"}, rbac.Data)

	settings := &corev1.ConfigMap{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: common.EverestSettingsConfigMapName}, settings))
	assert.Empty(t, settings.Data)
}

func TestRunNeverChangesExistingDefaults(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	existing := []client.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: common.EverestJWTSecretName},
			Data:       map[string][]byte{"id_rsa": []byte("the user's key")},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: common.EverestAccountsSecretName},
			Data:       map[string][]byte{common.EverestAccountsFileName: []byte("alice: {}\n")},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: common.EverestRBACConfigMapName},
			Data:       map[string]string{"enabled": "true", "policy.csv": "g, alice, role:admin\n"},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: common.EverestSettingsConfigMapName},
			Data:       map[string]string{"oidc.config": "issuerUrl: https://idp.example.com\n"},
		},
	}
	c := newFakeClient(t, existing...)

	b := &bootstrap.Bootstrap{Client: c, Namespace: namespace}
	require.NoError(t, b.Run(ctx))

	for _, want := range existing {
		got := want.DeepCopyObject().(client.Object) //nolint:forcetypeassert // a copy of a client.Object
		require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(want), got))
		switch w := want.(type) {
		case *corev1.Secret:
			assert.Equal(t, w.Data, got.(*corev1.Secret).Data, w.Name) //nolint:forcetypeassert // same type as want
		case *corev1.ConfigMap:
			assert.Equal(t, w.Data, got.(*corev1.ConfigMap).Data, w.Name) //nolint:forcetypeassert // same type as want
		}
	}
}

func TestRunAppliesBuiltinManifests(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := newFakeClient(t)

	b := &bootstrap.Bootstrap{Client: c, Namespace: namespace}
	require.NoError(t, b.Run(ctx))
	// A later start, or a resync, applies the same objects again.
	require.NoError(t, b.Run(ctx))

	policies := &admissionregistrationv1.ValidatingAdmissionPolicyList{}
	require.NoError(t, c.List(ctx, policies))
	names := make(map[string]admissionregistrationv1.ValidatingAdmissionPolicy, len(policies.Items))
	for _, p := range policies.Items {
		names[p.Name] = p
	}
	assert.Len(t, names, 5)
	actor, ok := names["lifecycle-actor-authoring.openeverest.io"]
	require.True(t, ok, "the lifecycle policies are applied")
	require.Len(t, actor.Spec.Validations, 1)
	assert.Contains(t, actor.Spec.Validations[0].Expression, `"system:serviceaccount:everest-system:everest-admin"`,
		"the policy names the API server of this namespace",
	)

	bindings := &admissionregistrationv1.ValidatingAdmissionPolicyBindingList{}
	require.NoError(t, c.List(ctx, bindings))
	assert.Len(t, bindings.Items, len(names))
	for _, binding := range bindings.Items {
		assert.Contains(t, names, binding.Spec.PolicyName, "binding %s names a policy that is not applied", binding.Name)
	}

	s3 := &backupv1alpha1.BackupStorageClass{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Name: "s3"}, s3))
	assert.NotEmpty(t, s3.Annotations["openeverest.io/contract-version"], "the stamped manifest is applied")
}

// TestManagerRoleCoversBuiltinManifests keeps the kubebuilder markers on
// Bootstrap in step with the manifests: the controller may patch only the
// built-in objects it names, so a new policy or class without its name in
// the markers would fail to apply on every install.
func TestManagerRoleCoversBuiltinManifests(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "config", "rbac", "role.yaml"))
	require.NoError(t, err)
	role := &rbacv1.ClusterRole{}
	require.NoError(t, yaml.Unmarshal(raw, role))

	allowed := func(group, resource, verb, name string) bool {
		for _, r := range role.Rules {
			if slices.Contains(r.APIGroups, group) && slices.Contains(r.Resources, resource) &&
				slices.Contains(r.Verbs, verb) && (len(r.ResourceNames) == 0 || slices.Contains(r.ResourceNames, name)) {
				return true
			}
		}
		return false
	}
	resources := map[string]string{
		"ValidatingAdmissionPolicy":        "validatingadmissionpolicies",
		"ValidatingAdmissionPolicyBinding": "validatingadmissionpolicybindings",
		"BackupStorageClass":               "backupstorageclasses",
	}

	objs, err := bootstrap.BuiltinManifests(namespace)
	require.NoError(t, err)
	for _, u := range objs {
		resource, ok := resources[u.GetKind()]
		require.True(t, ok, "add the resource of kind %s to this test", u.GetKind())
		group := u.GroupVersionKind().Group
		for _, verb := range []string{"create", "patch"} {
			assert.True(t, allowed(group, resource, verb, u.GetName()),
				"the manager role does not allow %s on %s %s: add it to the kubebuilder markers on Bootstrap", verb, resource, u.GetName(),
			)
		}
	}
}
