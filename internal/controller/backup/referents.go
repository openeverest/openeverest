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

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// referentAppearedOrGone passes only creations and deletions: an operation
// waits for its referents to exist, not for their contents.
func referentAppearedOrGone() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc:  func(event.UpdateEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

func backupUnfinished(b *backupv1alpha1.Backup) bool {
	return b.Status.State != backupv1alpha1.BackupStateSucceeded && b.Status.State != backupv1alpha1.BackupStateFailed
}

func restoreUnfinished(rs *backupv1alpha1.Restore) bool {
	return rs.Status.State != backupv1alpha1.RestoreStateSucceeded && rs.Status.State != backupv1alpha1.RestoreStateFailed
}

func importUnfinished(imp *backupv1alpha1.BackupImport) bool {
	return imp.Status.State != backupv1alpha1.BackupImportStateSucceeded &&
		imp.Status.State != backupv1alpha1.BackupImportStateFailed
}

// operationsFor enqueues the operations of list type L that keep selects for
// the referent in the event, from the event object's namespace or, when
// namespaced is false, from every namespace.
func operationsFor[L any, PL interface { //nolint:ireturn
	*L
	client.ObjectList
}](
	c client.Client,
	namespaced bool,
	keep func(operation, referent client.Object) bool,
) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, referent client.Object) []ctrl.Request {
		var items L
		list := PL(&items)
		var opts []client.ListOption
		if namespaced {
			opts = append(opts, client.InNamespace(referent.GetNamespace()))
		}
		if err := c.List(ctx, list, opts...); err != nil {
			log.FromContext(ctx).Error(err, "Listing operations to enqueue failed")
			return nil
		}
		var reqs []ctrl.Request
		_ = meta.EachListItem(list, func(item runtime.Object) error {
			if o, ok := item.(client.Object); ok && keep(o, referent) {
				reqs = append(reqs, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(o)})
			}
			return nil
		})
		return reqs
	})
}

func backupNamesClass(o, class client.Object) bool {
	b, ok := o.(*backupv1alpha1.Backup)
	return ok && backupUnfinished(b) && b.Spec.ClassRef.Name == class.GetName()
}

func backupNamesInstance(o, instance client.Object) bool {
	b, ok := o.(*backupv1alpha1.Backup)
	return ok && backupUnfinished(b) &&
		b.Spec.Origin.InstanceRef != nil && b.Spec.Origin.InstanceRef.Name == instance.GetName()
}

// restoreWaitsForClass selects the Restores still waiting for a referent; the
// class is reached through the source Backup or Instance, so it cannot be
// matched by name.
func restoreWaitsForClass(o, _ client.Object) bool {
	rs, ok := o.(*backupv1alpha1.Restore)
	return ok && (rs.Status.State == "" || rs.Status.State == backupv1alpha1.RestoreStatePending)
}

func restoreNamesBackup(o, backup client.Object) bool {
	rs, ok := o.(*backupv1alpha1.Restore)
	return ok && restoreUnfinished(rs) &&
		rs.Spec.DataSource.Backup != nil && rs.Spec.DataSource.Backup.BackupRef.Name == backup.GetName()
}

func restoreNamesInstance(o, instance client.Object) bool {
	rs, ok := o.(*backupv1alpha1.Restore)
	return ok && restoreUnfinished(rs) &&
		(rs.Spec.InstanceRef.Name == instance.GetName() || controller.RestoreStreamInstanceName(rs) == instance.GetName())
}

func importNamesClass(o, class client.Object) bool {
	imp, ok := o.(*backupv1alpha1.BackupImport)
	return ok && importUnfinished(imp) && imp.Spec.ClassRef.Name == class.GetName()
}
