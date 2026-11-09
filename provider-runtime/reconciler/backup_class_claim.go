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

	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/internal/backupclass"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

// backupClassClaimReconciler claims the BackupClasses that name the
// provider as their executor: it applies their Accepted condition, and only
// that, under the provider's status field owner (spec 019 §4.2). Classes
// executed by anyone else are ignored.
type backupClassClaimReconciler struct {
	client   client.Client
	provider string
	shipped  []backupv1alpha1.BackupClass
}

// setupBackupClassClaimReconciler starts claiming the classes that name
// provider, judging them against shipped.
func setupBackupClassClaimReconciler(mgr ctrl.Manager, provider string, shipped []backupv1alpha1.BackupClass) error {
	r := &backupClassClaimReconciler{client: mgr.GetClient(), provider: provider, shipped: shipped}
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.BackupClass{}, builder.WithPredicates(predicate.NewPredicateFuncs(r.claims))).
		Named(provider + "-backup-class-claim").
		Complete(r)
}

func (r *backupClassClaimReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	class := &backupv1alpha1.BackupClass{}
	if err := r.client.Get(ctx, req.NamespacedName, class); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	if !r.claims(class) || !class.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}
	cond, err := backupclass.Claim(class, r.shipped, r.provider)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cur := meta.FindStatusCondition(class.Status.Conditions, cond.Type); cur != nil &&
		cur.Status == cond.Status && cur.Reason == cond.Reason && cur.Message == cond.Message &&
		cur.ObservedGeneration == cond.ObservedGeneration {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, monitoring.ApplyCondition(ctx, r.client, class, class.Status.Conditions,
		controller.StatusFieldOwner(r.provider), cond,
	)
}

// claims reports whether obj is a BackupClass this provider executes.
func (r *backupClassClaimReconciler) claims(obj client.Object) bool {
	class, ok := obj.(*backupv1alpha1.BackupClass)
	return ok && class.Spec.ExecutorProvider() == r.provider
}
