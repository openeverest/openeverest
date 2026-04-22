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

// translatorFunc returns the target object to create for a given source
// object, or nil to skip. This is the package-internal abstraction shared by
// any "watch source X, create Y" controller (currently BackupMirror; future
// import flows can reuse it).
type translatorFunc func(ctx context.Context, c client.Client, src client.Object) (client.Object, error)

// setupWatchAndCreate wires a generic controller that watches sourceType and,
// for each event, invokes translate and idempotently creates the returned
// target object. AlreadyExists is treated as success. Translate returning
// (nil, nil) is the explicit "skip" signal.
func setupWatchAndCreate(
	mgr ctrl.Manager,
	name string,
	sourceType client.Object,
	translate translatorFunc,
) error {
	r := &watchAndCreateReconciler{
		client:     mgr.GetClient(),
		sourceType: sourceType,
		translate:  translate,
		name:       name,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(sourceType).
		Named(name).
		Complete(r)
}

type watchAndCreateReconciler struct {
	client     client.Client
	sourceType client.Object
	translate  translatorFunc
	name       string
}

func (r *watchAndCreateReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("controller", r.name, "source", req.NamespacedName)

	src := r.sourceType.DeepCopyObject().(client.Object)
	if err := r.client.Get(ctx, req.NamespacedName, src); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	if !src.GetDeletionTimestamp().IsZero() {
		return reconcile.Result{}, nil
	}

	target, err := r.translate(ctx, r.client, src)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("translate: %w", err)
	}
	if target == nil {
		return reconcile.Result{}, nil
	}

	if err := r.client.Create(ctx, target); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("create target: %w", err)
	}
	logger.Info("mirrored source into target",
		"target", client.ObjectKeyFromObject(target),
		"targetKind", fmt.Sprintf("%T", target))
	return reconcile.Result{}, nil
}

// setupBackupMirrorReconciler registers a watch-and-create controller for the
// provider's BackupMirror implementation. The translator adapts the typed
// Mirror method into the generic translatorFunc shape.
func setupBackupMirrorReconciler(mgr ctrl.Manager, bm controller.BackupMirror, providerName string) error {
	// Ensure the operator type is registered in the manager's scheme. The
	// provider's Types() registration already covers this in practice, but
	// asserting it here produces a clearer error if a provider forgets.
	gvk, _, err := mgr.GetScheme().ObjectKinds(bm.OperatorBackupType())
	if err != nil || len(gvk) == 0 {
		return fmt.Errorf("BackupMirror operator type %T not registered in scheme: %w", bm.OperatorBackupType(), err)
	}
	translate := func(ctx context.Context, c client.Client, src client.Object) (client.Object, error) {
		out, err := bm.Mirror(ctx, c, src)
		if err != nil {
			return nil, err
		}
		if out == nil {
			return nil, nil
		}
		return out, nil
	}
	return setupWatchAndCreate(mgr, providerName+"-backup-mirror", bm.OperatorBackupType(), translate)
}
