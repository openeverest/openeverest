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
	"github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	restoreRefNameLabel         = "restore.openeverest.io/ref-name"
	restoreRefNamespaceLabel    = "restore.openeverest.io/ref-namespace"
	restoreRBACCleanupFinalizer = "restore.openeverest.io/rbac-cleanup"
)

// RestoreReconciler reconciles Restore resources.
type RestoreReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("Restore").
		For(&backupv1alpha1.Restore{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&rbacv1.Role{}).
		Watches(&rbacv1.ClusterRoleBinding{}, rclusterWideResourceHandler()).
		Watches(&rbacv1.ClusterRole{}, rclusterWideResourceHandler()).
		Complete(r)
}

// rclusterWideResourceHandler returns an event handler that enqueues requests for Restore
// when cluster-wide resources like ClusterRole or ClusterRoleBinding are created, updated, or deleted.
func rclusterWideResourceHandler() handler.EventHandler { //nolint:ireturn
	return handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []ctrl.Request {
		labels := o.GetLabels()
		name, ok := labels[restoreRefNameLabel]
		if !ok {
			return nil
		}
		namespace, ok := labels[restoreRefNamespaceLabel]
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

//+kubebuilder:rbac:groups=backup.openeverest.io,resources=restores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=restores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=restores/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=create;get;list;watch;update;patch;delete;escalate;bind
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=create;get;list;watch;update;patch;delete;escalate;bind
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backupclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backups,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RestoreReconciler) Reconcile( //nolint:nonamedreturns
	ctx context.Context,
	req ctrl.Request,
) (rr ctrl.Result, rerr error) {
	logger := log.FromContext(ctx).
		WithName("RestoreReconciler").
		WithValues(
			"name", req.Name,
			"namespace", req.Namespace,
		)
	logger.Info("Reconciling")
	defer func() {
		logger.Info("Reconciled")
	}()

	restore := &backupv1alpha1.Restore{}
	if err := r.Client.Get(ctx, req.NamespacedName, restore); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !restore.GetDeletionTimestamp().IsZero() {
		ok, err := r.handleFinalizers(ctx, restore)
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

	// Ensure the instance name label is set on the Restore resource.
	if err := r.ensureInstanceNameLabel(ctx, restore); err != nil {
		return ctrl.Result{}, err
	}

	if restore.Status.State == backupv1alpha1.RestoreStateSucceeded ||
		restore.Status.State == backupv1alpha1.RestoreStateFailed {
		return ctrl.Result{}, nil
	}

	// Resolve the BackupClass before touching status so that ProviderManaged
	// restores can bail out without the deferred Status().Update() fighting
	// with the provider-runtime reconciler.
	bc, err := r.resolveBackupClass(ctx, restore)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to resolve backup class: %w", err)
	}

	if bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeJob {
		// ProviderManaged classes are reconciled by the provider-runtime; this
		// controller only handles Job execution mode. Bail out without touching
		// the Restore so the provider's runtime can take over.
		return ctrl.Result{}, nil
	}

	// Reset the status, we will build a new one by observing the current state on each reconcile.
	startedAt := restore.Status.StartedAt
	restore.Status = backupv1alpha1.RestoreStatus{}
	restore.Status.LastObservedGeneration = restore.GetGeneration()
	if startedAt != nil && !startedAt.Time.IsZero() {
		restore.Status.StartedAt = startedAt
	}

	// Sync status on finishing reconciliation.
	defer func() {
		if updErr := r.Client.Status().Update(ctx, restore); updErr != nil {
			logger.Error(updErr, "Failed to update restore status")
			rerr = errors.Join(rerr, updErr)
		}
	}()

	if bc.Spec.RestoreJob == nil || bc.Spec.RestoreJob.JobSpec == nil {
		restore.Status.State = backupv1alpha1.RestoreStateFailed
		restore.Status.Message = "BackupClass does not define spec.restoreJob.jobSpec"
		return ctrl.Result{}, nil
	}

	// Create RBAC resources.
	requiresRbac := len(bc.Spec.RestoreJob.Permissions) > 0 || len(bc.Spec.RestoreJob.ClusterPermissions) > 0
	if requiresRbac { //nolint:nestif
		if controllerutil.AddFinalizer(restore, restoreRBACCleanupFinalizer) {
			if err := r.Client.Update(ctx, restore); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to add finalizer to restore: %w", err)
			}
		}
		if err := r.ensureServiceAccount(ctx, restore); err != nil {
			restore.Status.State = backupv1alpha1.RestoreStateError
			restore.Status.Message = fmt.Errorf("failed to ensure service account: %w", err).Error()
			return ctrl.Result{}, err
		}
		if err := r.ensureRBACResources(ctx, restore, bc.Spec.RestoreJob.Permissions, bc.Spec.RestoreJob.ClusterPermissions); err != nil {
			restore.Status.State = backupv1alpha1.RestoreStateError
			restore.Status.Message = fmt.Errorf("failed to ensure RBAC resources: %w", err).Error()
			return ctrl.Result{}, err
		}
	}

	// Fetch the target Instance.
	instance := &corev1alpha1.Instance{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      restore.Spec.InstanceName,
		Namespace: restore.GetNamespace(),
	}, instance); err != nil {
		restore.Status.State = backupv1alpha1.RestoreStateError
		restore.Status.Message = fmt.Errorf("failed to get instance %q: %w", restore.Spec.InstanceName, err).Error()
		return ctrl.Result{}, err
	}

	// Ensure payload secret.
	if err := r.ensurePayloadSecret(ctx, restore, instance, bc); err != nil {
		restore.Status.State = backupv1alpha1.RestoreStateError
		restore.Status.Message = fmt.Errorf("failed to ensure payload secret: %w", err).Error()
		return ctrl.Result{}, err
	}

	// Create restore job.
	if err := r.ensureRestoreJob(ctx, requiresRbac, restore, bc); err != nil {
		restore.Status.State = backupv1alpha1.RestoreStateError
		restore.Status.Message = fmt.Errorf("failed to create restore job: %w", err).Error()
		return ctrl.Result{}, err
	}

	// Observe the restore job status.
	if err := r.observeJobStatus(ctx, restore); err != nil {
		restore.Status.State = backupv1alpha1.RestoreStateError
		restore.Status.Message = fmt.Errorf("failed to observe job status: %w", err).Error()
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// ensureInstanceNameLabel ensures that the Restore resource has the instance name
// label, used for filtering restores by instance using label selectors.
func (r *RestoreReconciler) ensureInstanceNameLabel(ctx context.Context, restore *backupv1alpha1.Restore) error {
	labels := restore.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	if labels[common.InstanceNameLabel] == restore.Spec.InstanceName {
		return nil
	}

	labels[common.InstanceNameLabel] = restore.Spec.InstanceName
	restore.SetLabels(labels)
	if err := r.Client.Update(ctx, restore); err != nil {
		return fmt.Errorf("failed to update instance name label: %w", err)
	}

	return nil
}

// resolveBackupClass determines the BackupClass to use based on the Restore's DataSource.
func (r *RestoreReconciler) resolveBackupClass(
	ctx context.Context,
	restore *backupv1alpha1.Restore,
) (*backupv1alpha1.BackupClass, error) {
	var backupClassName string

	ds := restore.Spec.DataSource
	switch {
	case ds.BackupName != "":
		// Resolve from the referenced Backup CR.
		backup := &backupv1alpha1.Backup{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      ds.BackupName,
			Namespace: restore.GetNamespace(),
		}, backup); err != nil {
			return nil, fmt.Errorf("failed to get backup %q: %w", ds.BackupName, err)
		}
		backupClassName = backup.Spec.BackupClassName

	case ds.External != nil:
		backupClassName = ds.External.BackupClassName

	default:
		return nil, fmt.Errorf("dataSource must specify either backupName or external")
	}

	bc := &backupv1alpha1.BackupClass{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name: backupClassName,
	}, bc); err != nil {
		return nil, fmt.Errorf("failed to get backup class %q: %w", backupClassName, err)
	}

	return bc, nil
}

func (r *RestoreReconciler) ensureRestoreJob(
	ctx context.Context,
	useServiceAccount bool,
	restore *backupv1alpha1.Restore,
	bc *backupv1alpha1.BackupClass,
) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreJobName(restore),
			Namespace: restore.GetNamespace(),
		},
	}

	restore.Status.JobName = job.GetName()

	// Check if the job already exists.
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(job), job); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get restore job: %w", err)
		}
	} else {
		return nil
	}

	serviceAccount := ""
	if useServiceAccount {
		serviceAccount = r.getServiceAccountName(restore)
	}
	job.Spec = r.getJobSpec(restore, bc, serviceAccount)
	if err := controllerutil.SetControllerReference(restore, job, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}
	if err := r.Client.Create(ctx, job); err != nil {
		return err
	}
	restore.Status.StartedAt = pointer.To(metav1.Now())
	return nil
}

func restoreToolRequestSecretName(restore *backupv1alpha1.Restore) string {
	return restore.GetName() + "-restore-request"
}

func (r *RestoreReconciler) observeJobStatus(ctx context.Context, restore *backupv1alpha1.Restore) error {
	jobName := restore.Status.JobName
	if jobName == "" {
		restore.Status.State = backupv1alpha1.RestoreStatePending
		return nil
	}

	job := &batchv1.Job{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      jobName,
		Namespace: restore.GetNamespace(),
	}, job); err != nil {
		return fmt.Errorf("failed to get restore job: %w", err)
	}

	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			// Job is complete, delete the payload secret.
			if err := r.Client.Delete(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      restoreToolRequestSecretName(restore),
					Namespace: restore.GetNamespace(),
				},
			}); client.IgnoreNotFound(err) != nil {
				return fmt.Errorf("failed to delete payload secret: %w", err)
			}
			restore.Status.State = backupv1alpha1.RestoreStateSucceeded
			restore.Status.CompletedAt = job.Status.CompletionTime
			return nil
		}

		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			restore.Status.State = backupv1alpha1.RestoreStateFailed
			restore.Status.Message = c.Message
			return nil
		}
	}
	restore.Status.State = backupv1alpha1.RestoreStateRunning
	return nil
}

func (r *RestoreReconciler) ensurePayloadSecret(
	ctx context.Context,
	restore *backupv1alpha1.Restore,
	instance *corev1alpha1.Instance,
	bc *backupv1alpha1.BackupClass,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreToolRequestSecretName(restore),
			Namespace: restore.GetNamespace(),
		},
	}

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

	// Resolve storage details from the DataSource. The DataSource may either
	// reference an existing Backup CR (in which case we read its storageName)
	// or describe an external backup (which carries its own storageName).
	var storageName string
	ds := restore.Spec.DataSource
	switch {
	case ds.BackupName != "":
		backup := &backupv1alpha1.Backup{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      ds.BackupName,
			Namespace: restore.GetNamespace(),
		}, backup); err != nil {
			return fmt.Errorf("failed to get backup %q: %w", ds.BackupName, err)
		}
		storageName = backup.Spec.StorageName
	case ds.External != nil:
		storageName = ds.External.StorageName
	}

	if storageName != "" {
		storage := &backupv1alpha1.BackupStorage{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      storageName,
			Namespace: restore.GetNamespace(),
		}, storage); err != nil {
			return fmt.Errorf("failed to get BackupStorage %q: %w", storageName, err)
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

			if s3Dest.CredentialsSecretName != "" {
				credSecret := &corev1.Secret{}
				if err := r.Client.Get(ctx, client.ObjectKey{
					Name:      s3Dest.CredentialsSecretName,
					Namespace: restore.GetNamespace(),
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

	// Populate PITR details.
	if ds.PITR != nil {
		spec.PITR = &jobspec.PITRDetails{
			Type: string(ds.PITR.Type),
		}
		if ds.PITR.Date != nil {
			spec.PITR.Date = ds.PITR.Date.Format("2006-01-02T15:04:05Z")
		}
	}

	// Pass through the config.
	if cfg := restore.Spec.Config; cfg != nil {
		cfgMap := map[string]any{}
		if err := json.Unmarshal(cfg.Raw, &cfgMap); err != nil {
			return fmt.Errorf("failed to unmarshal restore config: %w", err)
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
		return controllerutil.SetControllerReference(restore, secret, r.Scheme)
	}); err != nil {
		return fmt.Errorf("failed to create or update payload secret: %w", err)
	}
	return nil
}

func restoreJobName(restore *backupv1alpha1.Restore) string {
	uuid := restore.GetUID()
	hash := md5.Sum([]byte(uuid)) //nolint:gosec
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("%s-%s", restore.GetName(), hashStr[:6])
}

func (r *RestoreReconciler) getJobSpec(
	restore *backupv1alpha1.Restore,
	bc *backupv1alpha1.BackupClass,
	serviceAccountName string,
) batchv1.JobSpec {
	spec := batchv1.JobSpec{
		BackoffLimit: pointer.ToInt32(0),
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: pointer.ToInt64(30), //nolint:mnd
				ServiceAccountName:            serviceAccountName,
				RestartPolicy:                 corev1.RestartPolicyNever,
				Containers: []corev1.Container{{
					Name:    "restorer",
					Image:   bc.Spec.RestoreJob.JobSpec.Image,
					Command: bc.Spec.RestoreJob.JobSpec.Command,
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
								SecretName: restoreToolRequestSecretName(restore),
							},
						},
					},
				},
			},
		},
	}
	return spec
}

func (r *RestoreReconciler) ensureRBACResources(
	ctx context.Context,
	restore *backupv1alpha1.Restore,
	permissions, clusterPermissions []rbacv1.PolicyRule,
) error {
	if len(permissions) > 0 {
		if err := r.ensureRole(ctx, permissions, restore); err != nil {
			return fmt.Errorf("failed to ensure role: %w", err)
		}
		if err := r.ensureRoleBinding(ctx, restore); err != nil {
			return fmt.Errorf("failed to ensure role binding: %w", err)
		}
	}

	if len(clusterPermissions) > 0 {
		if err := r.ensureClusterRole(ctx, clusterPermissions, restore); err != nil {
			return fmt.Errorf("failed to ensure cluster role: %w", err)
		}
		if err := r.ensureClusterRoleBinding(ctx, restore); err != nil {
			return fmt.Errorf("failed to ensure cluster role binding: %w", err)
		}
	}
	return nil
}

func (r *RestoreReconciler) handleFinalizers(
	ctx context.Context,
	restore *backupv1alpha1.Restore,
) (bool, error) {
	if controllerutil.ContainsFinalizer(restore, restoreRBACCleanupFinalizer) {
		return r.deleteResourcesInOrder(ctx, restore)
	}
	return true, nil
}

func (r *RestoreReconciler) deleteJob(ctx context.Context, restore *backupv1alpha1.Restore) (bool, error) {
	jobName := restore.Status.JobName
	if jobName == "" {
		return true, nil
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: restore.GetNamespace(),
		},
	}
	if err := r.Client.Delete(ctx, job, &client.DeleteOptions{
		PropagationPolicy: pointer.To(metav1.DeletePropagationForeground),
	}); client.IgnoreNotFound(err) != nil {
		return false, fmt.Errorf("failed to delete job %s: %w", jobName, err)
	}

	const jobNameLabel = "job-name"
	pods := &corev1.PodList{}
	if err := r.Client.List(ctx, pods, client.InNamespace(restore.GetNamespace()), client.MatchingLabels{
		jobNameLabel: jobName,
	}); err != nil {
		return false, fmt.Errorf("failed to list pods for job %s: %w", jobName, err)
	}

	return len(pods.Items) == 0, nil
}

// Returns: [done(bool), error] .
func (r *RestoreReconciler) deleteRBAC(ctx context.Context, restore *backupv1alpha1.Restore) (bool, error) {
	resources := []client.Object{
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.getRoleBindingName(restore),
				Namespace: restore.GetNamespace(),
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.getRoleName(restore),
				Namespace: restore.GetNamespace(),
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: r.getClusterRoleBindingName(restore),
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.getClusterRoleName(restore),
				Namespace: restore.GetNamespace(),
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.getServiceAccountName(restore),
				Namespace: restore.GetNamespace(),
			},
		},
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

func (r *RestoreReconciler) deleteResourcesInOrder(ctx context.Context, restore *backupv1alpha1.Restore) (bool, error) {
	ok, err := r.deleteJob(ctx, restore)
	if err != nil {
		return false, fmt.Errorf("failed to delete job: %w", err)
	}

	if !ok {
		return false, nil
	}

	ok, err = r.deleteRBAC(ctx, restore)
	if err != nil {
		return false, fmt.Errorf("failed to delete RBAC resources: %w", err)
	}

	if !ok {
		return false, nil
	}

	if controllerutil.RemoveFinalizer(restore, restoreRBACCleanupFinalizer) {
		if err := r.Client.Update(ctx, restore); err != nil {
			return false, fmt.Errorf("failed to remove ordered cleanup finalizer: %w", err)
		}
	}

	return true, nil
}

func (r *RestoreReconciler) getClusterRoleBindingName(restore *backupv1alpha1.Restore) string {
	return restore.GetName() + "-restore-clusterrolebinding"
}

func (r *RestoreReconciler) ensureClusterRoleBinding(
	ctx context.Context,
	restore *backupv1alpha1.Restore,
) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getClusterRoleBindingName(restore),
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, clusterRoleBinding, func() error {
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     kindClusterRole,
			Name:     r.getClusterRoleName(restore),
		}
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      r.getServiceAccountName(restore),
				Namespace: restore.GetNamespace(),
			},
		}
		clusterRoleBinding.SetLabels(map[string]string{
			restoreRefNameLabel:      restore.GetName(),
			restoreRefNamespaceLabel: restore.GetNamespace(),
		})
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure cluster role binding: %w", err)
	}
	return nil
}

func (r *RestoreReconciler) getClusterRoleName(restore *backupv1alpha1.Restore) string {
	return restore.GetName() + "-restore-clusterrole"
}

func (r *RestoreReconciler) ensureClusterRole(
	ctx context.Context,
	permissions []rbacv1.PolicyRule,
	restore *backupv1alpha1.Restore,
) error {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getClusterRoleName(restore),
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, clusterRole, func() error {
		clusterRole.SetLabels(map[string]string{
			restoreRefNameLabel:      restore.GetName(),
			restoreRefNamespaceLabel: restore.GetNamespace(),
		})
		clusterRole.Rules = permissions
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure cluster role: %w", err)
	}
	return nil
}

func (r *RestoreReconciler) getRoleBindingName(restore *backupv1alpha1.Restore) string {
	return restore.GetName() + "-restore-rolebinding"
}

func (r *RestoreReconciler) ensureRoleBinding(
	ctx context.Context,
	restore *backupv1alpha1.Restore,
) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getRoleBindingName(restore),
			Namespace: restore.GetNamespace(),
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, roleBinding, func() error {
		roleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     kindRole,
			Name:     r.getRoleName(restore),
		}
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      r.getServiceAccountName(restore),
				Namespace: restore.GetNamespace(),
			},
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure role binding: %w", err)
	}
	return nil
}

func (r *RestoreReconciler) getRoleName(restore *backupv1alpha1.Restore) string {
	return restore.GetName() + "-restore-role"
}

func (r *RestoreReconciler) ensureRole(
	ctx context.Context,
	permissions []rbacv1.PolicyRule,
	restore *backupv1alpha1.Restore,
) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getRoleName(restore),
			Namespace: restore.GetNamespace(),
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

func (r *RestoreReconciler) getServiceAccountName(restore *backupv1alpha1.Restore) string {
	return restore.GetName() + "-restore-sa"
}

func (r *RestoreReconciler) ensureServiceAccount(
	ctx context.Context,
	restore *backupv1alpha1.Restore,
) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getServiceAccountName(restore),
			Namespace: restore.GetNamespace(),
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, serviceAccount, func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("failed to ensure service account: %w", err)
	}
	return nil
}
