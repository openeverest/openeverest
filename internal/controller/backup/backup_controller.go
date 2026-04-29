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
	"encoding/json"
	"errors"
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
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
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

	// Get the referenced backup class before touching status so that
	// ProviderManaged backups can bail out without the deferred Status().Update()
	// fighting with the provider-runtime reconciler.
	bc := &backupv1alpha1.BackupClass{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name: backup.Spec.BackupClassName,
	}, bc); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get backup class: %w", err)
	}

	// ProviderManaged classes are reconciled by the provider-runtime; this
	// controller only handles Job execution mode. Bail out without touching
	// the Backup so the provider's runtime can take over.
	if bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeJob {
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

	if bc.Spec.Job == nil || bc.Spec.Job.JobSpec == nil {
		backup.Status.State = backupv1alpha1.BackupStateFailed
		backup.Status.Message = "BackupClass uses Job execution mode but does not define spec.job.jobSpec"
		return ctrl.Result{}, nil
	}

	// Get the source database instance.
	instance := &corev1alpha1.Instance{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      backup.Spec.InstanceName,
		Namespace: backup.GetNamespace(),
	}, instance); err != nil {
		backup.Status.State = backupv1alpha1.BackupStateError
		backup.Status.Message = fmt.Errorf("failed to get instance: %w", err).Error()
		return ctrl.Result{}, err
	}

	// Create payload secret.
	if err := r.ensurePayloadSecret(ctx, backup, instance); err != nil {
		backup.Status.State = backupv1alpha1.BackupStateError
		backup.Status.Message = fmt.Errorf("failed to create payload secret: %w", err).Error()
		return ctrl.Result{}, err
	}

	// Create RBAC resources.
	requiresRbac := len(bc.Spec.Job.Permissions) > 0 || len(bc.Spec.Job.ClusterPermissions) > 0
	if requiresRbac { //nolint:nestif
		if controllerutil.AddFinalizer(backup, backupRBACCleanupFinalizer) {
			if err := r.Client.Update(ctx, backup); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to add finalizer to backup: %w", err)
			}
		}
		if err := r.ensureServiceAccount(ctx, backup); err != nil {
			backup.Status.State = backupv1alpha1.BackupStateError
			backup.Status.Message = fmt.Errorf("failed to ensure service account: %w", err).Error()
			return ctrl.Result{}, err
		}
		if err := r.ensureRBACResources(ctx, backup, bc.Spec.Job.Permissions, bc.Spec.Job.ClusterPermissions); err != nil {
			backup.Status.State = backupv1alpha1.BackupStateError
			backup.Status.Message = fmt.Errorf("failed to ensure RBAC resources: %w", err).Error()
			return ctrl.Result{}, err
		}
	}

	// Create backup job.
	if err := r.ensureBackupJob(ctx, requiresRbac, backup, bc); err != nil {
		backup.Status.State = backupv1alpha1.BackupStateError
		backup.Status.Message = fmt.Errorf("failed to create backup job: %w", err).Error()
		return ctrl.Result{}, err
	}

	// Observe job state.
	if err := r.observeJobStatus(ctx, backup); err != nil {
		backup.Status.State = backupv1alpha1.BackupStateError
		backup.Status.Message = fmt.Errorf("failed to observe state: %w", err).Error()
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BackupReconciler) observeJobStatus(ctx context.Context, backup *backupv1alpha1.Backup) error {
	jobName := backup.Status.JobName
	if jobName == "" {
		backup.Status.State = backupv1alpha1.BackupStatePending
		return nil
	}

	job := &batchv1.Job{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      jobName,
		Namespace: backup.GetNamespace(),
	}, job); err != nil {
		return fmt.Errorf("failed to get backup job: %w", err)
	}

	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			// Job is complete, delete the payload secret.
			if err := r.Client.Delete(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backupToolRequestSecretName(backup),
					Namespace: backup.GetNamespace(),
				},
			}); client.IgnoreNotFound(err) != nil {
				return fmt.Errorf("failed to delete payload secret: %w", err)
			}
			backup.Status.State = backupv1alpha1.BackupStateSucceeded
			backup.Status.CompletedAt = job.Status.CompletionTime
			return nil
		}

		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			backup.Status.State = backupv1alpha1.BackupStateFailed
			backup.Status.Message = c.Message
			return nil
		}
	}
	backup.Status.State = backupv1alpha1.BackupStateRunning
	return nil
}

func (r *BackupReconciler) ensurePayloadSecret(
	ctx context.Context,
	backup *backupv1alpha1.Backup,
	instance *corev1alpha1.Instance,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupToolRequestSecretName(backup),
			Namespace: backup.GetNamespace(),
		},
	}

	// If the Secret already exists with the desired key, skip recreation.
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret); err == nil {
		if val, ok := secret.Data[backupJobJSONSecretKey]; ok && len(val) > 0 {
			return nil
		}
	} else if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get payload secret: %w", err)
	}

	spec := jobspec.Spec{
		Instance: jobspec.InstanceRef{
			Name:      instance.GetName(),
			Namespace: instance.GetNamespace(),
		},
	}

	// Read connection details from the Instance's ConnectionSecretRef.
	if instance.Status.ConnectionSecretRef.Name != "" {
		connSecret := &corev1.Secret{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      instance.Status.ConnectionSecretRef.Name,
			Namespace: instance.GetNamespace(),
		}, connSecret); err != nil {
			return fmt.Errorf("failed to get connection secret: %w", err)
		}
		spec.Connection = &jobspec.ConnectionDetails{
			Type:     string(connSecret.Data["type"]),
			Provider: string(connSecret.Data["provider"]),
			Host:     string(connSecret.Data["host"]),
			Port:     string(connSecret.Data["port"]),
			Username: string(connSecret.Data["username"]),
			Password: string(connSecret.Data["password"]),
			URI:      string(connSecret.Data["uri"]),
		}
	}

	// Resolve S3 storage details from the referenced BackupStorage CR.
	if backup.Spec.StorageName != "" {
		storage := &backupv1alpha1.BackupStorage{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      backup.Spec.StorageName,
			Namespace: backup.GetNamespace(),
		}, storage); err != nil {
			return fmt.Errorf("failed to get BackupStorage %q: %w", backup.Spec.StorageName, err)
		}
		if storage.Spec.S3 != nil {
			s3Dest := storage.Spec.S3
			s3Details := &jobspec.S3Details{
				Bucket:         s3Dest.Bucket,
				Region:         s3Dest.Region,
				EndpointURL:    s3Dest.EndpointURL,
				VerifyTLS:      pointer.Get(s3Dest.VerifyTLS),
				ForcePathStyle: pointer.Get(s3Dest.ForcePathStyle),
			}

			// Read S3 credentials from the referenced Secret.
			if s3Dest.CredentialsSecretName != "" {
				credSecret := &corev1.Secret{}
				if err := r.Client.Get(ctx, client.ObjectKey{
					Name:      s3Dest.CredentialsSecretName,
					Namespace: backup.GetNamespace(),
				}, credSecret); err != nil {
					return fmt.Errorf("failed to get S3 credentials secret: %w", err)
				}
				s3Details.AccessKeyID = string(credSecret.Data["AWS_ACCESS_KEY_ID"])
				s3Details.SecretAccessKey = string(credSecret.Data["AWS_SECRET_ACCESS_KEY"])
			}

			spec.Storage = &jobspec.StorageDetails{
				S3: s3Details,
			}
		}
	}

	// Pass through the config.
	if cfg := backup.Spec.Config; cfg != nil {
		cfgMap := map[string]any{}
		if err := json.Unmarshal(cfg.Raw, &cfgMap); err != nil {
			return fmt.Errorf("failed to unmarshal backup config: %w", err)
		}
		spec.Config = cfgMap
	}

	reqJSON, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Data = map[string][]byte{
			backupJobJSONSecretKey: reqJSON,
		}
		return controllerutil.SetControllerReference(backup, secret, r.Scheme)
	}); err != nil {
		return fmt.Errorf("failed to create or update payload secret: %w", err)
	}
	return nil
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
					Image:   bc.Spec.Job.JobSpec.Image,
					Command: bc.Spec.Job.JobSpec.Command,
					Args:    []string{fmt.Sprintf("%s/%s", payloadMountPath, backupJobJSONSecretKey)},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "payload",
							MountPath: payloadMountPath,
							ReadOnly:  true,
						},
					},
				}},
				Volumes: []corev1.Volume{
					{
						Name: "payload",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: backupToolRequestSecretName(backup),
							},
						},
					},
				},
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
func (r *BackupReconciler) deleteRBAC(ctx context.Context, backup *backupv1alpha1.Backup) (bool, error) {
	// List of RBAC resources.
	resources := []client.Object{
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.getRoleBindingName(backup),
				Namespace: backup.GetNamespace(),
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.getRoleName(backup),
				Namespace: backup.GetNamespace(),
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: r.getClusterRoleBindingName(backup),
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.getClusterRoleName(backup),
				Namespace: backup.GetNamespace(),
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.getServiceAccountName(backup),
				Namespace: backup.GetNamespace(),
			},
		},
	}
	allGone := true

	for _, res := range resources {
		err := r.Client.Delete(ctx, res)
		if err == nil {
			allGone = false // Even one successful deletion indicates that not all resources are gone yet.
		} else if client.IgnoreNotFound(err) != nil {
			return false, fmt.Errorf("failed to delete resource %s: %w", res.GetName(), err)
		}
	}

	return allGone, nil
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

	ok, err = r.deleteRBAC(ctx, backup)
	if err != nil {
		return false, fmt.Errorf("failed to delete RBAC resources: %w", err)
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
