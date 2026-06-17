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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// backupMirrorReconciler watches operator-emitted backup CRs and creates a
// matching Backup CR for each one whose provider's Mirror method returns a
// non-nil target. AlreadyExists is treated as success so the loop is safely
// repeatable: once SyncBackup adopts the operator resource, subsequent events
// are no-ops.
type backupMirrorReconciler struct {
	client       client.Client
	mirror       controller.BackupMirror
	sourceType   client.Object
	providerName string
}

func setupBackupMirrorReconciler(mgr ctrl.Manager, bm controller.BackupMirror, providerName string) error {
	src := bm.OperatorBackupType()
	// Ensure the operator type is registered in the manager's scheme. The
	// provider's Types() registration already covers this in practice, but
	// asserting it here produces a clearer error if a provider forgets.
	if _, _, err := mgr.GetScheme().ObjectKinds(src); err != nil {
		return fmt.Errorf("BackupMirror operator type %T not registered in scheme: %w", src, err)
	}
	r := &backupMirrorReconciler{
		client:       mgr.GetClient(),
		mirror:       bm,
		sourceType:   src,
		providerName: providerName,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(src).
		Named(providerName + "-backup-mirror").
		Complete(r)
}

func (r *backupMirrorReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("provider", r.providerName, "operatorBackup", req.NamespacedName)

	src := r.sourceType.DeepCopyObject().(client.Object)
	if err := r.client.Get(ctx, req.NamespacedName, src); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	if !src.GetDeletionTimestamp().IsZero() {
		return reconcile.Result{}, nil
	}

	target, err := r.mirror.Mirror(ctx, r.client, src)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("mirror: %w", err)
	}
	if target == nil {
		return reconcile.Result{}, nil
	}

	if err := r.client.Create(ctx, target); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("create Backup: %w", err)
	}
	logger.Info("mirrored operator backup into Backup CR", "backup", client.ObjectKeyFromObject(target))
	return reconcile.Result{}, nil
}
