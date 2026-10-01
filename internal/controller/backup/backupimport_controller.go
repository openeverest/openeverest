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
	"crypto/md5" //nolint:gosec // non-cryptographic, deterministic job name derivation
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AlekSi/pointer"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/jobspec"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	importRBACCleanupFinalizer = "backup.openeverest.io/import-rbac-cleanup"
	defaultImportRequeue       = 5 * time.Second
)

// BackupImportReconciler reconciles Job-mode BackupImport resources. It spawns
// a discovery Job described by the referenced BackupClass's spec.job.import;
// that Job lists and parses the referenced BackupStorage and creates external
// Backup CRs directly. The controller only manages the Job (plus its payload
// Secret and optional RBAC) and mirrors the Job outcome into
// BackupImport.status. ProviderManaged BackupImports are reconciled by the
// provider-runtime, not here.
type BackupImportReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupImportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("BackupImport").
		For(&backupv1alpha1.BackupImport{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&rbacv1.Role{}).
		Watches(&rbacv1.ClusterRoleBinding{}, importClusterWideResourceHandler()).
		Watches(&rbacv1.ClusterRole{}, importClusterWideResourceHandler()).
		Complete(r)
}

// importClusterWideResourceHandler enqueues the owning BackupImport when one of
// its label-linked cluster-scoped RBAC resources changes. It uses backupRefName
// label to identify the owning BackupImport.
func importClusterWideResourceHandler() handler.EventHandler { //nolint:ireturn
	return handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []ctrl.Request {
		labels := o.GetLabels()
		name, ok := labels[backupRefNameLabel]
		if !ok {
			return nil
		}
		namespace, ok := labels[backupRefNamespaceLabel]
		if !ok {
			return nil
		}
		return []ctrl.Request{{NamespacedName: client.ObjectKey{Name: name, Namespace: namespace}}}
	})
}

//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backupimports,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backupimports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backupimports/finalizers,verbs=update
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backupstorages,verbs=get;list;watch

// Reconcile drives a Job-mode BackupImport through discovery.
func (r *BackupImportReconciler) Reconcile( //nolint:nonamedreturns
	ctx context.Context,
	req ctrl.Request,
) (rr ctrl.Result, rerr error) {
	logger := log.FromContext(ctx).
		WithName("BackupImportReconciler").
		WithValues("name", req.Name, "namespace", req.Namespace)
	logger.Info("Reconciling")
	defer func() { logger.Info("Reconciled") }()

	imp := &backupv1alpha1.BackupImport{}
	if err := r.Client.Get(ctx, req.NamespacedName, imp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !imp.GetDeletionTimestamp().IsZero() {
		ok, err := r.handleFinalizers(ctx, imp)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !ok {
			return ctrl.Result{RequeueAfter: defaultImportRequeue}, nil
		}
		return ctrl.Result{}, nil
	}

	bc := &backupv1alpha1.BackupClass{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: imp.Spec.ClassRef.Name}, bc); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get backup class: %w", err)
	}

	// ProviderManaged imports are reconciled by the provider-runtime. Bail out
	// without touching the BackupImport so the provider can take over.
	if bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeJob {
		return ctrl.Result{}, nil
	}

	// Succeeded and Failed are terminal: the spec is immutable, so discovery
	// re-runs only when the BackupImport is recreated.
	if imp.Status.State == backupv1alpha1.BackupImportStateSucceeded ||
		imp.Status.State == backupv1alpha1.BackupImportStateFailed {
		return ctrl.Result{}, nil
	}

	if bc.Spec.Job == nil || bc.Spec.Job.Import == nil || bc.Spec.Job.Import.JobSpec == nil {
		return r.setTerminal(ctx, imp, backupv1alpha1.BackupImportStateFailed,
			"BackupClass does not support import: spec.job.import.jobSpec is not defined")
	}

	imprt := bc.Spec.Job.Import

	if err := r.ensurePayloadSecret(ctx, imp); err != nil {
		return r.setTransientError(ctx, imp, fmt.Errorf("failed to create payload secret: %w", err))
	}

	requiresRbac := len(imprt.Permissions) > 0 || len(imprt.ClusterPermissions) > 0
	if requiresRbac { //nolint:nestif
		if controllerutil.AddFinalizer(imp, importRBACCleanupFinalizer) {
			if err := r.Client.Update(ctx, imp); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to add finalizer to backup import: %w", err)
			}
		}
		if err := r.ensureServiceAccount(ctx, imp); err != nil {
			return r.setTransientError(ctx, imp, fmt.Errorf("failed to ensure service account: %w", err))
		}
		if err := r.ensureRBACResources(ctx, imp, imprt.Permissions, imprt.ClusterPermissions); err != nil {
			return r.setTransientError(ctx, imp, fmt.Errorf("failed to ensure RBAC resources: %w", err))
		}
	}

	if err := r.ensureJob(ctx, imp, imprt, requiresRbac); err != nil {
		return r.setTransientError(ctx, imp, fmt.Errorf("failed to create import job: %w", err))
	}

	return r.observeJob(ctx, imp)
}

// observeJob inspects the discovery Job and mirrors its outcome into the
// BackupImport status. On completion it counts the external Backups the Job
// created (labeled with this import) to populate the discovered/created counts.
func (r *BackupImportReconciler) observeJob(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
) (ctrl.Result, error) {
	job := &batchv1.Job{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      importJobName(imp),
		Namespace: imp.GetNamespace(),
	}, job); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: defaultImportRequeue}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get import job: %w", err)
	}

	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			count, err := r.countImportedBackups(ctx, imp)
			if err != nil {
				return r.setTransientError(ctx, imp, fmt.Errorf("failed to count imported backups: %w", err))
			}
			if err := r.deletePayloadSecret(ctx, imp); err != nil {
				return ctrl.Result{}, err
			}
			imp.Status.DiscoveredCount = count
			imp.Status.CreatedCount = count
			return r.setTerminal(ctx, imp, backupv1alpha1.BackupImportStateSucceeded, "")
		}
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return r.setTerminal(ctx, imp, backupv1alpha1.BackupImportStateFailed, c.Message)
		}
	}

	// Job still running: record progress and requeue.
	if imp.Status.State != backupv1alpha1.BackupImportStateError {
		imp.Status.LastObservedGeneration = imp.GetGeneration()
		if err := r.Client.Status().Update(ctx, imp); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update backup import status: %w", err)
		}
	}
	return ctrl.Result{RequeueAfter: defaultImportRequeue}, nil
}

// countImportedBackups returns the number of external Backups in the import's
// namespace that the Job created for this BackupImport, identified by the
// BackupImportNameLabel.
func (r *BackupImportReconciler) countImportedBackups(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
) (int32, error) {
	list := &backupv1alpha1.BackupList{}
	if err := r.Client.List(ctx, list,
		client.InNamespace(imp.GetNamespace()),
		client.MatchingLabels{common.BackupImportNameLabel: imp.GetName()},
	); err != nil {
		return 0, fmt.Errorf("failed to list imported backups: %w", err)
	}
	return int32(len(list.Items)), nil //nolint:gosec // bounded by storage listing
}

// setTerminal records a terminal state (Succeeded/Failed) and stops reconciling.
func (r *BackupImportReconciler) setTerminal(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
	state backupv1alpha1.BackupImportState,
	message string,
) (ctrl.Result, error) {
	imp.Status.State = state
	imp.Status.Message = message
	imp.Status.LastObservedGeneration = imp.GetGeneration()
	if err := r.Client.Status().Update(ctx, imp); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update backup import status: %w", err)
	}
	return ctrl.Result{}, nil
}

// setTransientError records a retryable error on the status and returns the
// cause so the controller requeues with backoff.
func (r *BackupImportReconciler) setTransientError(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
	cause error,
) (ctrl.Result, error) {
	imp.Status.State = backupv1alpha1.BackupImportStateError
	imp.Status.Message = cause.Error()
	imp.Status.LastObservedGeneration = imp.GetGeneration()
	_ = r.Client.Status().Update(ctx, imp)
	return ctrl.Result{}, cause
}

// ensurePayloadSecret writes the import job payload (target namespace, class and
// storage references, and the resolved S3 storage details) into a Secret owned
// by the BackupImport and mounted into the job container.
func (r *BackupImportReconciler) ensurePayloadSecret(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
) error {
	spec := jobspec.ImportSpec{
		Namespace:  imp.GetNamespace(),
		ImportName: imp.GetName(),
		ClassRef:   imp.Spec.ClassRef.Name,
		StorageRef: imp.Spec.StorageRef.Name,
	}

	storage := &backupv1alpha1.BackupStorage{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      imp.Spec.StorageRef.Name,
		Namespace: imp.GetNamespace(),
	}, storage); err != nil {
		return fmt.Errorf("failed to get BackupStorage %q: %w", imp.Spec.StorageRef.Name, err)
	}
	if s3 := storage.Spec.S3; s3 != nil {
		details, err := buildS3StorageDetails(ctx, r.Client, imp.GetNamespace(), s3)
		if err != nil {
			return err
		}
		spec.Storage = details
	}

	reqJSON, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      importPayloadSecretName(imp),
			Namespace: imp.GetNamespace(),
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Data = map[string][]byte{backupJobJSONSecretKey: reqJSON}
		return controllerutil.SetControllerReference(imp, secret, r.Scheme)
	}); err != nil {
		return fmt.Errorf("failed to create or update payload secret: %w", err)
	}
	return nil
}

func (r *BackupImportReconciler) deletePayloadSecret(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
) error {
	if err := r.Client.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      importPayloadSecretName(imp),
			Namespace: imp.GetNamespace(),
		},
	}); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete payload secret: %w", err)
	}
	return nil
}

// ensureJob creates the discovery Job if it does not already exist.
func (r *BackupImportReconciler) ensureJob(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
	imprt *backupv1alpha1.JobExecution,
	useServiceAccount bool,
) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      importJobName(imp),
			Namespace: imp.GetNamespace(),
		},
	}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(job), job); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get import job: %w", err)
		}
	} else {
		return nil
	}

	serviceAccount := ""
	if useServiceAccount {
		serviceAccount = r.getServiceAccountName(imp)
	}
	job.Spec = importJobSpec(imp, imprt, serviceAccount)
	if err := controllerutil.SetControllerReference(imp, job, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}
	return r.Client.Create(ctx, job)
}

func importJobSpec(
	imp *backupv1alpha1.BackupImport,
	imprt *backupv1alpha1.JobExecution,
	serviceAccountName string,
) batchv1.JobSpec {
	return batchv1.JobSpec{
		// No retries: fail fast. Re-running discovery means recreating the CR.
		BackoffLimit: pointer.ToInt32(0),
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: pointer.ToInt64(30), //nolint:mnd
				ServiceAccountName:            serviceAccountName,
				RestartPolicy:                 corev1.RestartPolicyNever,
				Containers: []corev1.Container{{
					Name:    "importer",
					Image:   imprt.JobSpec.Image,
					Command: imprt.JobSpec.Command,
					Args:    []string{fmt.Sprintf("%s/%s", payloadMountPath, backupJobJSONSecretKey)},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "payload",
						MountPath: payloadMountPath,
						ReadOnly:  true,
					}},
				}},
				Volumes: []corev1.Volume{{
					Name: "payload",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: importPayloadSecretName(imp),
						},
					},
				}},
			},
		},
	}
}

// ensureRBACResources grants the discovery job the namespace- and
// cluster-scoped permissions declared on spec.job.import. To create the
// external Backup CRs the job pod needs create permission on
// backups.backup.openeverest.io, supplied by the class author here.
func (r *BackupImportReconciler) ensureRBACResources(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
	permissions, clusterPermissions []rbacv1.PolicyRule,
) error {
	if len(permissions) > 0 {
		if err := r.ensureRole(ctx, imp, permissions); err != nil {
			return err
		}
		if err := r.ensureRoleBinding(ctx, imp); err != nil {
			return err
		}
	}
	if len(clusterPermissions) > 0 {
		if err := r.ensureClusterRole(ctx, imp, clusterPermissions); err != nil {
			return err
		}
		if err := r.ensureClusterRoleBinding(ctx, imp); err != nil {
			return err
		}
	}
	return nil
}

func (r *BackupImportReconciler) ensureServiceAccount(ctx context.Context, imp *backupv1alpha1.BackupImport) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: r.getServiceAccountName(imp), Namespace: imp.GetNamespace()},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, sa, func() error {
		return controllerutil.SetControllerReference(imp, sa, r.Scheme)
	}); err != nil {
		return fmt.Errorf("failed to ensure service account: %w", err)
	}
	return nil
}

func (r *BackupImportReconciler) ensureRole(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
	permissions []rbacv1.PolicyRule,
) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: r.getRoleName(imp), Namespace: imp.GetNamespace()},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, role, func() error {
		role.Rules = permissions
		return controllerutil.SetControllerReference(imp, role, r.Scheme)
	}); err != nil {
		return fmt.Errorf("failed to ensure role: %w", err)
	}
	return nil
}

func (r *BackupImportReconciler) ensureRoleBinding(ctx context.Context, imp *backupv1alpha1.BackupImport) error {
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: r.getRoleBindingName(imp), Namespace: imp.GetNamespace()},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, rb, func() error {
		rb.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     kindRole,
			Name:     r.getRoleName(imp),
		}
		rb.Subjects = []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      r.getServiceAccountName(imp),
			Namespace: imp.GetNamespace(),
		}}
		return controllerutil.SetControllerReference(imp, rb, r.Scheme)
	}); err != nil {
		return fmt.Errorf("failed to ensure role binding: %w", err)
	}
	return nil
}

func (r *BackupImportReconciler) ensureClusterRole(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
	permissions []rbacv1.PolicyRule,
) error {
	cr := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: r.getClusterRoleName(imp)}}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, cr, func() error {
		cr.SetLabels(map[string]string{
			backupRefNameLabel:      imp.GetName(),
			backupRefNamespaceLabel: imp.GetNamespace(),
		})
		cr.Rules = permissions
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure cluster role: %w", err)
	}
	return nil
}

func (r *BackupImportReconciler) ensureClusterRoleBinding(ctx context.Context, imp *backupv1alpha1.BackupImport) error {
	crb := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: r.getClusterRoleBindingName(imp)}}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, crb, func() error {
		crb.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     kindClusterRole,
			Name:     r.getClusterRoleName(imp),
		}
		crb.Subjects = []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      r.getServiceAccountName(imp),
			Namespace: imp.GetNamespace(),
		}}
		crb.SetLabels(map[string]string{
			backupRefNameLabel:      imp.GetName(),
			backupRefNamespaceLabel: imp.GetNamespace(),
		})
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure cluster role binding: %w", err)
	}
	return nil
}

// handleFinalizers tears down the Job and RBAC in order when the BackupImport
// is being deleted. Returns true once everything is gone and the finalizer has
// been removed. The discovered external Backup CRs are intentionally left in
// place: they are standalone records, not owned by the BackupImport.
func (r *BackupImportReconciler) handleFinalizers(
	ctx context.Context,
	imp *backupv1alpha1.BackupImport,
) (bool, error) {
	if !controllerutil.ContainsFinalizer(imp, importRBACCleanupFinalizer) {
		return true, nil
	}

	ok, err := r.deleteJob(ctx, imp)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	ok, err = r.deleteRBAC(ctx, imp)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	if controllerutil.RemoveFinalizer(imp, importRBACCleanupFinalizer) {
		if err := r.Client.Update(ctx, imp); err != nil {
			return false, fmt.Errorf("failed to remove import cleanup finalizer: %w", err)
		}
	}
	return true, nil
}

func (r *BackupImportReconciler) deleteJob(ctx context.Context, imp *backupv1alpha1.BackupImport) (bool, error) {
	jobName := importJobName(imp)
	if err := r.Client.Delete(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: imp.GetNamespace()},
	}, &client.DeleteOptions{
		PropagationPolicy: pointer.To(metav1.DeletePropagationForeground),
	}); client.IgnoreNotFound(err) != nil {
		return false, fmt.Errorf("failed to delete job %s: %w", jobName, err)
	}

	const jobNameLabel = "job-name"
	pods := &corev1.PodList{}
	if err := r.Client.List(ctx, pods,
		client.InNamespace(imp.GetNamespace()),
		client.MatchingLabels{jobNameLabel: jobName},
	); err != nil {
		return false, fmt.Errorf("failed to list pods for job %s: %w", jobName, err)
	}
	return len(pods.Items) == 0, nil
}

func (r *BackupImportReconciler) deleteRBAC(ctx context.Context, imp *backupv1alpha1.BackupImport) (bool, error) {
	resources := []client.Object{
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: r.getRoleBindingName(imp), Namespace: imp.GetNamespace()}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: r.getRoleName(imp), Namespace: imp.GetNamespace()}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: r.getClusterRoleBindingName(imp)}},
		&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: r.getClusterRoleName(imp)}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: r.getServiceAccountName(imp), Namespace: imp.GetNamespace()}},
	}
	allGone := true
	for _, res := range resources {
		err := r.Client.Delete(ctx, res)
		if err == nil {
			allGone = false
		} else if client.IgnoreNotFound(err) != nil {
			return false, fmt.Errorf("failed to delete resource %s: %w", res.GetName(), err)
		}
	}
	return allGone, nil
}

func (r *BackupImportReconciler) getServiceAccountName(imp *backupv1alpha1.BackupImport) string {
	return imp.GetName() + "-import-sa"
}

func (r *BackupImportReconciler) getRoleName(imp *backupv1alpha1.BackupImport) string {
	return imp.GetName() + "-import-role"
}

func (r *BackupImportReconciler) getRoleBindingName(imp *backupv1alpha1.BackupImport) string {
	return imp.GetName() + "-import-rolebinding"
}

func (r *BackupImportReconciler) getClusterRoleName(imp *backupv1alpha1.BackupImport) string {
	return imp.GetName() + "-import-clusterrole"
}

func (r *BackupImportReconciler) getClusterRoleBindingName(imp *backupv1alpha1.BackupImport) string {
	return imp.GetName() + "-import-clusterrolebinding"
}

func importJobName(imp *backupv1alpha1.BackupImport) string {
	hash := md5.Sum([]byte(imp.GetUID())) //nolint:gosec
	return fmt.Sprintf("%s-import-%s", imp.GetName(), hex.EncodeToString(hash[:])[:6])
}

func importPayloadSecretName(imp *backupv1alpha1.BackupImport) string {
	return importJobName(imp) + backupToolRequestSecretNameSuffix
}
