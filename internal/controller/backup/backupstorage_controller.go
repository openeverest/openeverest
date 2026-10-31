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
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

const (
	backupStorageFinalizer = "backupstorage.backup.openeverest.io/in-use-protection"

	// backupStorageFieldOwner is the field manager of the status core writes
	// on BackupStorages.
	backupStorageFieldOwner = "openeverest-backup-storage"

	// instanceBackupStorageRefField indexes Instances by the BackupStorages
	// they reference: backup storage entries, whether backups are enabled or
	// not, and a point-in-time data source.
	instanceBackupStorageRefField = ".spec.backup.storages.storageRef.name"
	// backupStorageNameField indexes Backups by their storage.
	backupStorageNameField = ".spec.storageName"
	// restoreStorageRefField indexes Restores by the storage of their
	// point-in-time source.
	restoreStorageRefField = ".spec.dataSource.pointInTime.source.storageRef.name"
	// backupImportStorageRefField indexes BackupImports by their storage.
	backupImportStorageRefField = ".spec.storageRef.name"
	// backupStorageSecretsField indexes BackupStorages by the Secrets they
	// reference.
	backupStorageSecretsField = ".spec.secretNames"
	// backupStorageClassField indexes BackupStorages by their class.
	backupStorageClassField = ".spec.backupStorageClassRef.name"
)

// BackupStorageReconciler owns the status of BackupStorages: Accepted, the
// BackupClasses that can write to the storage, and Deleting while a deleted
// storage waits for its referrers.
type BackupStorageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorages,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorages/finalizers,verbs=update
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorageclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backups;restores;backupimports,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.openeverest.io,resources=instances,verbs=get;list;watch

// Reconcile publishes whether the storage is valid for its class and which
// BackupClasses can write to it, and holds a deleted storage until nothing
// references it.
func (r *BackupStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	bs := &backupv1alpha1.BackupStorage{}
	if err := r.Get(ctx, req.NamespacedName, bs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if bs.GetDeletionTimestamp().IsZero() && !controllerutil.ContainsFinalizer(bs, backupStorageFinalizer) {
		controllerutil.AddFinalizer(bs, backupStorageFinalizer)
		if err := r.Update(ctx, bs); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	accepted, err := r.acceptedCondition(ctx, bs)
	if err != nil {
		return ctrl.Result{}, err
	}
	conditions := []metav1.Condition{accepted}
	if !bs.GetDeletionTimestamp().IsZero() && controllerutil.ContainsFinalizer(bs, backupStorageFinalizer) {
		referrers, err := r.referrersOf(ctx, bs.Namespace, bs.Name)
		if err != nil {
			return ctrl.Result{}, err
		}
		if len(referrers) == 0 {
			controllerutil.RemoveFinalizer(bs, backupStorageFinalizer)
			if err := r.Update(ctx, bs); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
			return ctrl.Result{}, nil
		}
		logf.FromContext(ctx).Info("BackupStorage is deleted but still referenced", "referrers", strings.Join(referrers, ", "))
		conditions = append(conditions, rtmonitoring.NewCondition(bs, backupv1alpha1.BackupStorageConditionDeleting, metav1.ConditionTrue,
			backupv1alpha1.ReasonWaitingForReferrers, "still referenced by "+strings.Join(referrers, ", "),
		))
	}
	classes, err := r.backupClasses(ctx, bs)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, r.applyStatus(ctx, bs, conditions, classes)
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.initIndexers(context.Background(), mgr); err != nil {
		return fmt.Errorf("init field indexers: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.BackupStorage{}).
		Named("backup-backupstorage").
		Watches(&backupv1alpha1.BackupStorageClass{},
			handler.EnqueueRequestsFromMapFunc(r.storagesMatching(backupStorageClassField, false))).
		Watches(&backupv1alpha1.BackupClass{},
			handler.EnqueueRequestsFromMapFunc(r.allStorages)).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.storagesMatching(backupStorageSecretsField, true))).
		Watches(&corev1alpha1.Instance{},
			handler.EnqueueRequestsFromMapFunc(r.referencedStorages(instanceStorageNames))).
		Watches(&backupv1alpha1.Backup{},
			handler.EnqueueRequestsFromMapFunc(r.referencedStorages(backupStorageNames))).
		Watches(&backupv1alpha1.Restore{},
			handler.EnqueueRequestsFromMapFunc(r.referencedStorages(restoreStorageNames))).
		Watches(&backupv1alpha1.BackupImport{},
			handler.EnqueueRequestsFromMapFunc(r.referencedStorages(backupImportStorageNames))).
		Complete(r)
}

// acceptedCondition judges the storage against its class.
func (r *BackupStorageReconciler) acceptedCondition(ctx context.Context, bs *backupv1alpha1.BackupStorage) (metav1.Condition, error) {
	class := &backupv1alpha1.BackupStorageClass{}
	if err := r.Get(ctx, types.NamespacedName{Name: bs.Spec.BackupStorageClassRef.Name}, class); err != nil {
		if !apierrors.IsNotFound(err) {
			return metav1.Condition{}, fmt.Errorf("get BackupStorageClass %q: %w", bs.Spec.BackupStorageClassRef.Name, err)
		}
		return rtmonitoring.NewCondition(bs, backupv1alpha1.BackupStorageConditionAccepted, metav1.ConditionFalse,
			backupv1alpha1.ReasonClassNotFound, fmt.Sprintf("BackupStorageClass %q not found", bs.Spec.BackupStorageClassRef.Name),
		), nil
	}
	in := storageclasses.AcceptInput{
		Location:       bs.Spec.Location,
		Parameters:     bs.Spec.Parameters,
		Authentication: bs.Spec.Authentication,
		TLS:            bs.Spec.TLS,
	}
	var err error
	if ref := bs.Spec.Authentication.CredentialsSecretRef; ref != nil {
		if in.Credentials, err = r.secretContent(ctx, bs.Namespace, ref.Name); err != nil {
			return metav1.Condition{}, err
		}
	}
	if t := bs.Spec.TLS; t != nil && t.CA != nil && t.CA.SecretKeyRef != nil {
		if in.CA, err = r.secretContent(ctx, bs.Namespace, t.CA.SecretKeyRef.Name); err != nil {
			return metav1.Condition{}, err
		}
	}
	v := storageclasses.Accept(class, in)
	status := metav1.ConditionFalse
	if v.Accepted() {
		status = metav1.ConditionTrue
	}
	return rtmonitoring.NewCondition(bs, backupv1alpha1.BackupStorageConditionAccepted, status, v.Reason, v.Message), nil
}

func (r *BackupStorageReconciler) secretContent(ctx context.Context, namespace, name string) (storageclasses.SecretContent, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return storageclasses.SecretContent{}, nil
		}
		return storageclasses.SecretContent{}, fmt.Errorf("get Secret %q: %w", name, err)
	}
	return storageclasses.SecretContent{Found: true, Data: secret.Data}, nil
}

// backupClasses lists every BackupClass and whether it can write to the
// storage.
func (r *BackupStorageReconciler) backupClasses(ctx context.Context, bs *backupv1alpha1.BackupStorage) ([]backupv1alpha1.BackupClassCompatibility, error) {
	list := &backupv1alpha1.BackupClassList{}
	if err := r.List(ctx, list); err != nil {
		return nil, fmt.Errorf("list BackupClasses: %w", err)
	}
	out := make([]backupv1alpha1.BackupClassCompatibility, 0, len(list.Items))
	for i := range list.Items {
		c := storageclasses.Compatible(&list.Items[i], bs)
		out = append(out, backupv1alpha1.BackupClassCompatibility{Name: list.Items[i].Name, Compatible: c.Compatible, Reason: c.Reason})
	}
	slices.SortFunc(out, func(a, b backupv1alpha1.BackupClassCompatibility) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// applyStatus server-side-applies every status field core owns, so a
// condition left out (Deleting once the storage is no longer referenced) is
// removed.
func (r *BackupStorageReconciler) applyStatus(ctx context.Context, bs *backupv1alpha1.BackupStorage,
	conditions []metav1.Condition, classes []backupv1alpha1.BackupClassCompatibility,
) error {
	conds := make([]any, 0, len(conditions))
	for _, cond := range conditions {
		cond.LastTransitionTime = metav1.Now()
		if prev := meta.FindStatusCondition(bs.Status.Conditions, cond.Type); prev != nil && prev.Status == cond.Status {
			cond.LastTransitionTime = prev.LastTransitionTime
		}
		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&cond)
		if err != nil {
			return fmt.Errorf("convert condition: %w", err)
		}
		conds = append(conds, raw)
	}
	compat := make([]any, 0, len(classes))
	for i := range classes {
		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&classes[i])
		if err != nil {
			return fmt.Errorf("convert backup class compatibility: %w", err)
		}
		compat = append(compat, raw)
	}
	return rtmonitoring.ApplyStatus(ctx, r.Client, bs, backupStorageFieldOwner, map[string]any{
		"conditions":    conds,
		"backupClasses": compat,
	})
}

// referrersOf describes what still references the BackupStorage name in
// namespace, by kind; empty when nothing does.
func (r *BackupStorageReconciler) referrersOf(ctx context.Context, namespace, name string) ([]string, error) {
	opts := func(field string) *client.ListOptions {
		return &client.ListOptions{Namespace: namespace, FieldSelector: fields.OneTermEqualSelector(field, name)}
	}
	lists := []struct {
		kind  string
		list  client.ObjectList
		field string
	}{
		{"Instance", &corev1alpha1.InstanceList{}, instanceBackupStorageRefField},
		{"Backup", &backupv1alpha1.BackupList{}, backupStorageNameField},
		{"Restore", &backupv1alpha1.RestoreList{}, restoreStorageRefField},
		{"BackupImport", &backupv1alpha1.BackupImportList{}, backupImportStorageRefField},
	}
	var out []string
	for _, l := range lists {
		if err := r.List(ctx, l.list, opts(l.field)); err != nil {
			return nil, fmt.Errorf("list %ss: %w", l.kind, err)
		}
		if n := meta.LenList(l.list); n > 0 {
			out = append(out, fmt.Sprintf("%d %s(s)", n, l.kind))
		}
	}
	return out, nil
}

func instanceStorageNames(obj client.Object) []string {
	in, ok := obj.(*corev1alpha1.Instance)
	if !ok {
		return nil
	}
	var names []string
	if in.Spec.Backup != nil {
		for _, s := range in.Spec.Backup.Storages {
			names = append(names, s.StorageRef.Name)
		}
	}
	if ds := in.Spec.DataSource; ds != nil && ds.PointInTime != nil && !slices.Contains(names, ds.PointInTime.Source.StorageRef.Name) {
		names = append(names, ds.PointInTime.Source.StorageRef.Name)
	}
	return names
}

func backupStorageNames(obj client.Object) []string {
	b, ok := obj.(*backupv1alpha1.Backup)
	if !ok || b.Spec.StorageRef.Name == "" {
		return nil
	}
	return []string{b.Spec.StorageRef.Name}
}

func restoreStorageNames(obj client.Object) []string {
	rs, ok := obj.(*backupv1alpha1.Restore)
	if !ok || rs.Spec.DataSource.PointInTime == nil {
		return nil
	}
	return []string{rs.Spec.DataSource.PointInTime.Source.StorageRef.Name}
}

func backupImportStorageNames(obj client.Object) []string {
	bi, ok := obj.(*backupv1alpha1.BackupImport)
	if !ok || bi.Spec.StorageRef.Name == "" {
		return nil
	}
	return []string{bi.Spec.StorageRef.Name}
}

func storageSecretNames(obj client.Object) []string {
	bs, ok := obj.(*backupv1alpha1.BackupStorage)
	if !ok {
		return nil
	}
	return bs.Spec.SecretNames()
}

func storageClassName(obj client.Object) []string {
	bs, ok := obj.(*backupv1alpha1.BackupStorage)
	if !ok {
		return nil
	}
	return []string{bs.Spec.BackupStorageClassRef.Name}
}

// initIndexers registers the field indexers required by this controller.
func (r *BackupStorageReconciler) initIndexers(ctx context.Context, mgr ctrl.Manager) error {
	indexes := []struct {
		obj   client.Object
		field string
		fn    client.IndexerFunc
	}{
		{&corev1alpha1.Instance{}, instanceBackupStorageRefField, instanceStorageNames},
		{&backupv1alpha1.Backup{}, backupStorageNameField, backupStorageNames},
		{&backupv1alpha1.Restore{}, restoreStorageRefField, restoreStorageNames},
		{&backupv1alpha1.BackupImport{}, backupImportStorageRefField, backupImportStorageNames},
		{&backupv1alpha1.BackupStorage{}, backupStorageSecretsField, storageSecretNames},
		{&backupv1alpha1.BackupStorage{}, backupStorageClassField, storageClassName},
	}
	for _, idx := range indexes {
		if err := mgr.GetFieldIndexer().IndexField(ctx, idx.obj, idx.field, idx.fn); err != nil {
			return fmt.Errorf("indexing %T by %s: %w", idx.obj, idx.field, err)
		}
	}
	return nil
}

// referencedStorages maps a referrer to the BackupStorages it names in its
// namespace.
func (r *BackupStorageReconciler) referencedStorages(names func(client.Object) []string) handler.MapFunc {
	return func(_ context.Context, obj client.Object) []reconcile.Request {
		storages := names(obj)
		reqs := make([]reconcile.Request, 0, len(storages))
		for _, name := range storages {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: obj.GetNamespace(), Name: name}})
		}
		return reqs
	}
}

// storagesMatching maps an object to the BackupStorages whose index field
// names it, in its namespace when namespaced.
func (r *BackupStorageReconciler) storagesMatching(field string, namespaced bool) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		opts := []client.ListOption{client.MatchingFieldsSelector{Selector: fields.OneTermEqualSelector(field, obj.GetName())}}
		if namespaced {
			opts = append(opts, client.InNamespace(obj.GetNamespace()))
		}
		list := &backupv1alpha1.BackupStorageList{}
		if err := r.List(ctx, list, opts...); err != nil {
			logf.FromContext(ctx).Error(err, "List BackupStorages to enqueue", "field", field, "name", obj.GetName())
			return nil
		}
		return storageRequests(list)
	}
}

func (r *BackupStorageReconciler) allStorages(ctx context.Context, _ client.Object) []reconcile.Request {
	list := &backupv1alpha1.BackupStorageList{}
	if err := r.List(ctx, list); err != nil {
		logf.FromContext(ctx).Error(err, "List BackupStorages to enqueue")
		return nil
	}
	return storageRequests(list)
}

func storageRequests(list *backupv1alpha1.BackupStorageList) []reconcile.Request {
	reqs := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
	}
	return reqs
}
