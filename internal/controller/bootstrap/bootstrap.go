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

// Package bootstrap creates, when the controller starts, what OpenEverest
// needs to run but its installer may not ship. An installer that renders
// manifests without a cluster, such as Argo CD rendering the Helm chart, or
// that installs a fixed bundle, such as OLM, cannot generate a signing key
// or an admin password once and keep them: here the controller does.
package bootstrap

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// FieldOwner is the field manager the controller applies the built-in
// manifests under.
const FieldOwner = "openeverest-controller-bootstrap"

// defaultRBAC is the everest-rbac ConfigMap a fresh install starts with:
// RBAC off, and the admin user given the admin role for when it is turned on.
func defaultRBAC() map[string]string {
	return map[string]string{
		"enabled":    "false",
		"policy.csv": fmt.Sprintf("g, %s, %s\n", common.EverestAdminUser, common.EverestAdminRole),
	}
}

// Retry is how long Bootstrap waits after a failed attempt: First after the
// first, doubling up to Most.
type Retry struct {
	First time.Duration
	Most  time.Duration
}

// Bootstrap is a runnable that, on the leader, does what is enabled once,
// retrying until it succeeds or the manager stops. A failure never stops the
// manager: the controllers run regardless.
type Bootstrap struct {
	// Client must not be the manager's cached client: a Get through it would
	// start an informer on every Secret and ConfigMap in the cluster.
	Client client.Client
	// Namespace is where OpenEverest runs, and where the defaults go.
	Namespace string
	// CreateMissingDefaults creates the defaults the API server needs, the
	// everest-jwt and everest-accounts Secrets and the everest-rbac and
	// everest-settings ConfigMaps, if they are missing. It never changes an
	// existing one: once created, they belong to the user.
	CreateMissingDefaults bool
	// ApplyBuiltinManifests server-side applies the admission policies and
	// the built-in BackupStorageClasses this version ships, see
	// BuiltinManifests.
	ApplyBuiltinManifests bool
	Retry                 Retry
}

// +kubebuilder:rbac:groups=core,resources=secrets;configmaps,verbs=get;create

// Start runs Run until it succeeds or ctx ends. It returns no error, so a
// failure does not stop the manager.
func (b *Bootstrap) Start(ctx context.Context) error { //nolint:unparam // the signature of manager.Runnable
	log := logf.FromContext(ctx).WithName("bootstrap")
	ctx = logf.IntoContext(ctx, log)
	delay := b.Retry.First
	for {
		err := b.Run(ctx)
		if err == nil {
			return nil
		}
		log.Error(err, "Failed to bootstrap; retrying", "after", delay)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
		delay = min(2*delay, b.Retry.Most)
	}
}

// Run does once what is enabled.
func (b *Bootstrap) Run(ctx context.Context) error {
	if b.CreateMissingDefaults {
		if err := b.createMissingDefaults(ctx); err != nil {
			return err
		}
	}
	if b.ApplyBuiltinManifests {
		if err := b.applyBuiltinManifests(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bootstrap) createMissingDefaults(ctx context.Context) error {
	defaults := []struct {
		existing client.Object
		build    func() (client.Object, error)
	}{
		{
			existing: named(&corev1.Secret{}, b.Namespace, common.EverestJWTSecretName),
			build: func() (client.Object, error) {
				return kubernetes.NewJWTSecret(b.Namespace)
			},
		},
		{
			existing: named(&corev1.Secret{}, b.Namespace, common.EverestAccountsSecretName),
			build: func() (client.Object, error) {
				return kubernetes.NewInitialAccountsSecret(b.Namespace)
			},
		},
		{
			existing: named(&corev1.ConfigMap{}, b.Namespace, common.EverestRBACConfigMapName),
			build: func() (client.Object, error) {
				return b.configMap(common.EverestRBACConfigMapName, defaultRBAC()), nil
			},
		},
		{
			existing: named(&corev1.ConfigMap{}, b.Namespace, common.EverestSettingsConfigMapName),
			build: func() (client.Object, error) {
				return b.configMap(common.EverestSettingsConfigMapName, nil), nil
			},
		},
	}
	for _, d := range defaults {
		if err := b.createIfMissing(ctx, d.existing, d.build); err != nil {
			return err
		}
	}
	return nil
}

func named[T client.Object](obj T, namespace, name string) T {
	obj.SetName(name)
	obj.SetNamespace(namespace)
	return obj
}

// createIfMissing builds and creates the object named by existing only if
// it is not found: generating the signing key is costly.
func (b *Bootstrap) createIfMissing(ctx context.Context, existing client.Object, build func() (client.Object, error)) error {
	kind := fmt.Sprintf("%T", existing)
	name := existing.GetName()
	err := b.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing)
	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("get %s %s: %w", kind, name, err)
	}
	obj, err := build()
	if err != nil {
		return fmt.Errorf("build %s %s: %w", kind, name, err)
	}
	if err := b.Client.Create(ctx, obj); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("create %s %s: %w", kind, name, err)
	}
	logf.FromContext(ctx).Info("Created a missing default", "kind", kind, "name", name, "namespace", b.Namespace)
	return nil
}

func (b *Bootstrap) configMap(name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: b.Namespace},
		Data:       data,
	}
}

func (b *Bootstrap) applyBuiltinManifests(ctx context.Context) error {
	objs, err := BuiltinManifests(b.Namespace)
	if err != nil {
		return err
	}
	for _, u := range objs {
		if err := b.Client.Apply(ctx, client.ApplyConfigurationFromUnstructured(u),
			client.FieldOwner(FieldOwner), client.ForceOwnership,
		); err != nil {
			return fmt.Errorf("apply %s %s: %w", u.GetKind(), u.GetName(), err)
		}
	}
	logf.FromContext(ctx).Info("Applied the built-in manifests", "objects", len(objs))
	return nil
}
