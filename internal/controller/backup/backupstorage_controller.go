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

package backup

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// BackupStorageReconciler reconciles a BackupStorage object
type BackupStorageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorages/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;update

// Reconcile reconciles a BackupStorage by adopting the credentials Secret as
// a child resource (setting a controller owner reference on it) so that the
// Secret is garbage-collected when the BackupStorage is deleted.
func (r *BackupStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	bs := &backupv1alpha1.BackupStorage{}
	if err := r.Get(ctx, req.NamespacedName, bs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if bs.Spec.S3 == nil || bs.Spec.S3.CredentialsSecretName == "" {
		return ctrl.Result{}, nil
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      bs.Spec.S3.CredentialsSecretName,
		Namespace: req.Namespace,
	}, secret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Adopt the secret if it has no controller yet.
	if metav1.GetControllerOf(secret) == nil {
		logger.Info("Setting controller reference for the credentials secret", "secret", secret.Name)
		if err := controllerutil.SetControllerReference(bs, secret, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set controller reference on secret: %w", err)
		}
		if err := r.Update(ctx, secret); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update secret with controller reference: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.BackupStorage{}).
		Named("backup-backupstorage").
		Owns(&corev1.Secret{}).
		Complete(r)
}
