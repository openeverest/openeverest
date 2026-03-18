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

// Package backup contains the BackupReconciler which manages Backup resources.
package backup

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/AlekSi/pointer"
	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
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
)

const (
	backupToolRequestSecretNameSuffix = "-data-import-request" //nolint:gosec
	backupJobJSONSecretKey            = "request.json"
	payloadMountPath                  = "/payload"

	// Label constants for backup resource handling
	backupRefNameLabel         = "backup.openeverest.io/ref-name"
	backupRefNamespaceLabel    = "backup.openeverest.io/ref-namespace"
	backupRBACCleanupFinalizer = "backup.openeverest.io/rbac-cleanup"

	kindRole        = "Role"
	kindClusterRole = "ClusterRole"
)

// BackupReconciler reconciles Backup resources.
type BackupReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("Backup").
		For(&backupv1alpha1.Backup{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&rbacv1.Role{}).
		Watches(&rbacv1.ClusterRoleBinding{}, bclusterWideResourceHandler()).
		Watches(&rbacv1.ClusterRole{}, bclusterWideResourceHandler()).
		Complete(r)
}

// bclusterWideResourceHandler returns an event handler that enqueues requests for Backup
// when cluster-wide resources like ClusterRole or ClusterRoleBinding are created, updated, or deleted.
// It uses the `backupRefNameLabel` to find the owner Backup and enqueue a request for it.
func bclusterWideResourceHandler() handler.EventHandler { //nolint:ireturn
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
		return []ctrl.Request{
			{
				NamespacedName: client.ObjectKey{
					Name:      name,
					Namespace: namespace,
				},
			},
		}
	})
}

//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backups/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=create;get;list;watch;create;update;patch;delete;escalate;bind
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=create;get;list;watch;create;update;patch;delete;escalate;bind
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backupclasses,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *BackupReconciler) Reconcile( //nolint:nonamedreturns
	ctx context.Context,
	req ctrl.Request,
) (rr ctrl.Result, rerr error) {
	logger := log.FromContext(ctx).
		WithName("BackupReconciler").
		WithValues(
			"name", req.Name,
			"namespace", req.Namespace,
		)
	logger.Info("Reconciling")
	defer func() {
		logger.Info("Reconciled")
	}()

	backup := &backupv1alpha1.Backup{}
	if err := r.Client.Get(ctx, req.NamespacedName, backup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !backup.GetDeletionTimestamp().IsZero() {
		ok, err := r.handleFinalizers(ctx, backup)
		if err != nil {
			logger.Error(err, "Failed to handle finalizers")
			return ctrl.Result{}, err
		}

		result := ctrl.Result{}
		if !ok {
			result.RequeueAfter = 5 * time.Second //nolint:mnd
		}

		return result, nil
	}

	if backup.Status.State == backupv1alpha1.BackupStateSucceeded ||
		backup.Status.State == backupv1alpha1.BackupStateFailed {
		// Already complete, no need to reconcile again.
		return ctrl.Result{}, nil
	}

	// Reset the status, we will build a new one by observing the current state on each reconcile.
	startedAt := backup.Status.StartedAt
	backup.Status = backupv1alpha1.BackupStatus{}
	backup.Status.LastObservedGeneration = backup.GetGeneration()
	if startedAt != nil && !startedAt.Time.IsZero() {
		backup.Status.StartedAt = startedAt
	}

	// Sync status on finishing reconciliation.
	defer func() {
		if updErr := r.Client.Status().Update(ctx, backup); updErr != nil {
			logger.Error(updErr, "Failed to update backup status")
			rerr = errors.Join(rerr, updErr)
		}
	}()

	// Get the referenced backup class.
	bc := &backupv1alpha1.BackupClass{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name: backup.Spec.BackupClassName,
	}, bc); err != nil {
		backup.Status.State = backupv1alpha1.BackupStateError
		backup.Status.Message = fmt.Errorf("failed to get backup class: %w", err).Error()
		return ctrl.Result{}, err
	}

	// Create RBAC resources.
	requiresRbac := len(bc.Spec.Permissions) > 0 || len(bc.Spec.ClusterPermissions) > 0
	if requiresRbac { //nolint:nestif
		if err := r.ensureServiceAccount(ctx, backup); err != nil {
			backup.Status.State = backupv1alpha1.BackupStateError
			backup.Status.Message = fmt.Errorf("failed to ensure service account: %w", err).Error()
			return ctrl.Result{}, err
		}
		if err := r.ensureRBACResources(ctx, backup, bc.Spec.Permissions, bc.Spec.ClusterPermissions); err != nil {
			backup.Status.State = backupv1alpha1.BackupStateError
			backup.Status.Message = fmt.Errorf("failed to ensure RBAC resources: %w", err).Error()
			return ctrl.Result{}, err
		}

		if controllerutil.AddFinalizer(backup, backupRBACCleanupFinalizer) {
			if err := r.Client.Update(ctx, backup); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to add finalizer to backup: %w", err)
			}
		}
	}

	// Create backup job.
	if err := r.ensureBackupJob(ctx, requiresRbac, backup, bc); err != nil {
		backup.Status.State = backupv1alpha1.BackupStateError
		backup.Status.Message = fmt.Errorf("failed to create backup job: %w", err).Error()
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BackupReconciler) ensureBackupJob(
	ctx context.Context,
	useServiceAccount bool,
	backup *backupv1alpha1.Backup,
	bc *backupv1alpha1.BackupClass,
) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupJobName(backup),
			Namespace: backup.GetNamespace(),
		},
	}

	backup.Status.JobName = job.GetName()

	// Check if the job already exists.
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(job), job); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get backup job: %w", err)
		}
	} else {
		return nil
	}

	serviceAccount := ""
	if useServiceAccount {
		serviceAccount = r.getServiceAccountName(backup)
	}
	job.Spec = r.getJobSpec(backup, bc, serviceAccount)
	if err := controllerutil.SetControllerReference(backup, job, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}
	if err := r.Client.Create(ctx, job); err != nil {
		return err
	}
	backup.Status.StartedAt = pointer.To(metav1.Now())
	return nil
}

func backupJobName(backup *backupv1alpha1.Backup) string {
	uuid := backup.GetUID()
	hash := md5.Sum([]byte(uuid)) //nolint:gosec
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("%s-%s", backup.GetName(), hashStr[:6])
}

func backupToolRequestSecretName(backup *backupv1alpha1.Backup) string {
	return backupJobName(backup) + backupToolRequestSecretNameSuffix
}

func (r *BackupReconciler) getJobSpec(
	backup *backupv1alpha1.Backup,
	bc *backupv1alpha1.BackupClass,
	serviceAccountName string,
) batchv1.JobSpec {
	spec := batchv1.JobSpec{
		// Setting it to 0 means we will not retry on failure.
		// TODO: In EVEREST-2108, we will implement failurePolicy, and that's where we shall
		// implement retries. For now we disable retries so it can fail fast.
		// See: https://perconadev.atlassian.net/browse/EVEREST-2108
		BackoffLimit: pointer.ToInt32(0),
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: pointer.ToInt64(30), //nolint:mnd  // TODO: make this configurable?
				ServiceAccountName:            serviceAccountName,
				RestartPolicy:                 corev1.RestartPolicyNever,
				Containers: []corev1.Container{{
					Name:    "importer",
					Image:   bc.Spec.JobSpec.Image,
					Command: bc.Spec.JobSpec.Command,
					Args:    []string{fmt.Sprintf("%s/%s", payloadMountPath, backupJobJSONSecretKey)},
				}},
			},
		},
	}
	return spec
}

func (r *BackupReconciler) ensureRBACResources(
	ctx context.Context,
	backup *backupv1alpha1.Backup,
	permissions, clusterPermissions []rbacv1.PolicyRule,
) error {
	if len(permissions) > 0 {
		if err := r.ensureRole(ctx, permissions, backup); err != nil {
			return fmt.Errorf("failed to ensure role: %w", err)
		}
		if err := r.ensureRoleBinding(ctx, backup); err != nil {
			return fmt.Errorf("failed to ensure role binding: %w", err)
		}
	}

	if len(clusterPermissions) > 0 {
		if err := r.ensureClusterRole(ctx, clusterPermissions, backup); err != nil {
			return fmt.Errorf("failed to ensure cluster role: %w", err)
		}
		if err := r.ensureClusterRoleBinding(ctx, backup); err != nil {
			return fmt.Errorf("failed to ensure cluster role binding: %w", err)
		}
	}
	return nil
}

// Returns: [done(bool), error] .
func (r *BackupReconciler) handleFinalizers(
	ctx context.Context,
	backup *backupv1alpha1.Backup,
) (bool, error) {
	if controllerutil.ContainsFinalizer(backup, backupRBACCleanupFinalizer) {
		return r.deleteResourcesInOrder(ctx, backup)
	}

	return true, nil
}

// Returns: [done(bool), error] .
func (r *BackupReconciler) deleteJob(ctx context.Context, backup *backupv1alpha1.Backup) (bool, error) {
	jobName := backup.Status.JobName
	if jobName == "" {
		return true, nil
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: backup.GetNamespace(),
		},
	}
	// Terminate the Job.
	if err := r.Client.Delete(ctx, job, &client.DeleteOptions{
		PropagationPolicy: pointer.To(metav1.DeletePropagationForeground),
	}); client.IgnoreNotFound(err) != nil {
		return false, fmt.Errorf("failed to delete job %s: %w", jobName, err)
	}
	// Ensure Pods have terminated before proceeding.
	const jobNameLabel = "job-name"

	pods := &corev1.PodList{}
	if err := r.Client.List(ctx, pods, client.InNamespace(backup.GetNamespace()), client.MatchingLabels{
		jobNameLabel: jobName,
	}); err != nil {
		return false, fmt.Errorf("failed to list pods for job %s: %w", jobName, err)
	}

	return len(pods.Items) == 0, nil // if no pods are running, we can proceed with RBAC cleanup.
}

// Returns: [done(bool), error] .
func (r *BackupReconciler) deleteResourcesInOrder(ctx context.Context, backup *backupv1alpha1.Backup) (bool, error) {
	ok, err := r.deleteJob(ctx, backup)
	if err != nil {
		return false, fmt.Errorf("failed to delete job: %w", err)
	}

	if !ok {
		return false, nil // do not proceed with RBAC cleanup if job deletion is not done
	}

	if !ok {
		return false, nil
	}
	if controllerutil.RemoveFinalizer(backup, backupRBACCleanupFinalizer) {
		if err := r.Client.Update(ctx, backup); err != nil {
			return false, fmt.Errorf("failed to remove ordered cleanup finalizer: %w", err)
		}
	}

	return true, nil
}

func (r *BackupReconciler) getClusterRoleBindingName(backup *backupv1alpha1.Backup) string {
	return backup.GetName() + "-clusterrolebinding"
}

func (r *BackupReconciler) ensureClusterRoleBinding(
	ctx context.Context,
	backup *backupv1alpha1.Backup,
) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getClusterRoleBindingName(backup),
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, clusterRoleBinding, func() error {
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     kindClusterRole,
			Name:     r.getClusterRoleName(backup),
		}
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      r.getServiceAccountName(backup),
				Namespace: backup.GetNamespace(),
			},
		}
		clusterRoleBinding.SetLabels(map[string]string{
			backupRefNameLabel:      backup.GetName(),
			backupRefNamespaceLabel: backup.GetNamespace(),
		})
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure cluster role binding: %w", err)
	}
	return nil
}

func (r *BackupReconciler) getClusterRoleName(backup *backupv1alpha1.Backup) string {
	return backup.GetName() + "-clusterrole"
}

func (r *BackupReconciler) ensureClusterRole(
	ctx context.Context,
	permissions []rbacv1.PolicyRule,
	backup *backupv1alpha1.Backup,
) error {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getClusterRoleName(backup),
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, clusterRole, func() error {
		clusterRole.SetLabels(map[string]string{
			backupRefNameLabel:      backup.GetName(),
			backupRefNamespaceLabel: backup.GetNamespace(),
		})
		clusterRole.Rules = permissions
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure cluster role: %w", err)
	}
	return nil
}

func (r *BackupReconciler) getRoleBindingName(backup *backupv1alpha1.Backup) string {
	return backup.GetName() + "-rolebinding"
}

func (r *BackupReconciler) ensureRoleBinding(
	ctx context.Context,
	backup *backupv1alpha1.Backup,
) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getRoleBindingName(backup),
			Namespace: backup.GetNamespace(),
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, roleBinding, func() error {
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     kindRole,
			Name:     r.getRoleName(backup),
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      r.getServiceAccountName(backup),
				Namespace: backup.GetNamespace(),
			},
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure role binding: %w", err)
	}
	return nil
}

func (r *BackupReconciler) getRoleName(backup *backupv1alpha1.Backup) string {
	return backup.GetName() + "-role"
}

func (r *BackupReconciler) ensureRole(
	ctx context.Context,
	permissions []rbacv1.PolicyRule,
	backup *backupv1alpha1.Backup,
) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getRoleName(backup),
			Namespace: backup.GetNamespace(),
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, role, func() error {
		role.Rules = permissions
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure role: %w", err)
	}
	return nil
}

func (r *BackupReconciler) getServiceAccountName(backup *backupv1alpha1.Backup) string {
	return backup.GetName() + "-sa"
}

func (r *BackupReconciler) ensureServiceAccount(
	ctx context.Context,
	backup *backupv1alpha1.Backup,
) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getServiceAccountName(backup),
			Namespace: backup.GetNamespace(),
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, serviceAccount, func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure service account: %w", err)
	}
	return nil
}
