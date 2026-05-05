package k8s

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/cenkalti/backoff"
	goversion "github.com/hashicorp/go-version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/common"
)

const (
	// PXC default upload interval
	// https://github.com/percona/percona-xtradb-cluster-operator/blob/25ad952931b3760ba22f082aa827fecb0e48162e/pkg/apis/pxc/v1/pxc_types.go#L938
	pxcDefaultUploadInterval = 60
	// PSMDB default upload interval
	// https://github.com/percona/percona-server-mongodb-operator/blob/98b72fac893eeb8a96e366d49a70d3aaaa4e9ed4/pkg/apis/psmdb/v1/psmdb_defaults.go#L514
	psmdbDefaultUploadInterval = 600
	// PG default upload interval
	// https://github.com/percona/percona-postgresql-operator/blob/82673d4d80aa329b5bd985889121280caad064fb/internal/pgbackrest/postgres.go#L58
	pgDefaultUploadInterval = 60
)

func (h *k8sHandler) CreateDatabaseCluster(ctx context.Context, db *everestv1alpha1.DatabaseCluster) (*everestv1alpha1.DatabaseCluster, error) {
	return h.kubeConnector.CreateDatabaseCluster(ctx, db)
}

func (h *k8sHandler) ListDatabaseClusters(ctx context.Context, namespace string) (*everestv1alpha1.DatabaseClusterList, error) {
	return h.kubeConnector.ListDatabaseClusters(ctx, ctrlclient.InNamespace(namespace))
}

func (h *k8sHandler) DeleteDatabaseCluster(ctx context.Context, namespace, name string, req *api.DeleteDatabaseClusterParams) error {
	cleanupStorage := pointer.Get(req.CleanupBackupStorage)

	backups, err := h.kubeConnector.ListDatabaseClusterBackups(ctx, ctrlclient.InNamespace(namespace))
	if err != nil {
		return err
	}

	if !cleanupStorage {
		for _, backup := range backups.Items {
			// Doesn't belong to this cluster, skip.
			if backup.Spec.DBClusterName != name || !backup.GetDeletionTimestamp().IsZero() {
				continue
			}
			if err := h.ensureBackupStorageProtection(ctx, &backup); err != nil {
				return errors.Join(err, errors.New("could not ensure backup storage protection"))
			}
		}
	}

	// Ensure foreground deletion on the Backups,
	// so that they're visible on the UI while getting deleted.
	for _, backup := range backups.Items {
		// Doesn't belong to this cluster, skip.
		if backup.Spec.DBClusterName != name || !backup.GetDeletionTimestamp().IsZero() {
			continue
		}
		if err := h.ensureBackupForegroundDeletion(ctx, &backup); err != nil {
			return errors.Join(err, errors.New("could not ensure foreground deletion"))
		}
	}

	delObj := &everestv1alpha1.DatabaseCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	return h.kubeConnector.DeleteDatabaseCluster(ctx, delObj)
}

func (h *k8sHandler) UpdateDatabaseCluster(ctx context.Context, db *everestv1alpha1.DatabaseCluster) (*everestv1alpha1.DatabaseCluster, error) {
	return h.kubeConnector.UpdateDatabaseCluster(ctx, db)
}

func (h *k8sHandler) GetDatabaseCluster(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseCluster, error) {
	return h.kubeConnector.GetDatabaseCluster(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

func (h *k8sHandler) GetDatabaseClusterCredentials(ctx context.Context, namespace, name string) (*api.DatabaseClusterCredential, error) {
	databaseCluster, err := h.kubeConnector.GetDatabaseCluster(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to get database cluster %s/%s: %w", namespace, name, err)
	}
	secret, err := h.kubeConnector.GetSecret(ctx, types.NamespacedName{Namespace: namespace, Name: databaseCluster.Spec.Engine.UserSecretsName})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", namespace, databaseCluster.Spec.Engine.UserSecretsName, err)
	}
	response := &api.DatabaseClusterCredential{}
	switch databaseCluster.Spec.Engine.Type {
	case everestv1alpha1.DatabaseEnginePXC:
		response.Username = pointer.ToString("root")
		response.Password = pointer.ToString(string(secret.Data["root"]))
	case everestv1alpha1.DatabaseEnginePSMDB:
		response.Username = pointer.ToString(string(secret.Data["MONGODB_DATABASE_ADMIN_USER"]))
		response.Password = pointer.ToString(string(secret.Data["MONGODB_DATABASE_ADMIN_PASSWORD"]))
		response.ConnectionUrl = h.connectionURL(ctx, databaseCluster, *response.Username, *response.Password)
	case everestv1alpha1.DatabaseEnginePostgresql:
		response.Username = pointer.ToString("postgres")
		response.Password = pointer.ToString(string(secret.Data["password"]))
		response.ConnectionUrl = h.connectionURL(ctx, databaseCluster, *response.Username, *response.Password)
	default:
		return nil, fmt.Errorf("unsupported database engine type: %s", databaseCluster.Spec.Engine.Type)
	}
	return response, nil
}

//nolint:funlen
func (h *k8sHandler) GetDatabaseClusterComponents(ctx context.Context, namespace, name string) ([]api.DatabaseClusterComponent, error) {
	pods, err := h.kubeConnector.ListPods(ctx, ctrlclient.InNamespace(namespace), ctrlclient.MatchingLabels{"app.kubernetes.io/instance": name})
	if err != nil {
		return nil, fmt.Errorf("failed to get pods for database cluster %s/%s: %w", namespace, name, err)
	}

	res := make([]api.DatabaseClusterComponent, 0, len(pods.Items))
	for _, pod := range pods.Items {
		component := pod.Labels["app.kubernetes.io/component"]
		if component == "" {
			continue
		}

		restarts := 0
		ready := 0
		containers := make([]api.DatabaseClusterComponentContainer, 0, len(pod.Status.ContainerStatuses))
		for _, c := range pod.Status.ContainerStatuses {
			restarts += int(c.RestartCount)
			if c.Ready {
				ready++
			}
			var started time.Time
			if c.State.Running != nil {
				started = c.State.Running.StartedAt.Time
			}

			var status string
			switch {
			case c.State.Running != nil:
				status = "Running"
			case c.State.Waiting != nil:
				status = "Waiting"
			case c.State.Terminated != nil:
				status = "Terminated"
			}

			var startedString *string
			if !started.IsZero() {
				startedString = pointer.ToString(started.Format(time.RFC3339))
			}
			containers = append(containers, api.DatabaseClusterComponentContainer{
				Name:     &c.Name, //nolint:exportloopref
				Started:  startedString,
				Ready:    &c.Ready, //nolint:exportloopref
				Restarts: pointer.ToInt(int(c.RestartCount)),
				Status:   &status,
			})
		}

		var started *string
		if startTime := pod.Status.StartTime; startTime != nil && !startTime.Time.IsZero() {
			started = pointer.ToString(pod.Status.StartTime.Time.Format(time.RFC3339))
		}
		res = append(res, api.DatabaseClusterComponent{
			Status:     pointer.ToString(string(pod.Status.Phase)),
			Name:       &pod.Name, //nolint:exportloopref
			Type:       &component,
			Started:    started,
			Restarts:   pointer.ToInt(restarts),
			Ready:      pointer.ToString(fmt.Sprintf("%d/%d", ready, len(pod.Status.ContainerStatuses))),
			Containers: &containers,
		})
	}
	return res, nil
}

func (h *k8sHandler) GetDatabaseClusterComponentLogs(ctx context.Context, namespace, clusterName, componentName string, params api.GetDatabaseClusterComponentLogsParams, stream handlers.StreamFunc) error {
	// after all the previous handlers allowed the request we can start streaming logs.
	return stream(ctx, namespace, clusterName, componentName, params)
}

func (h *k8sHandler) GetDatabaseClusterPitr(ctx context.Context, namespace, name string) (*api.DatabaseClusterPitr, error) {
	databaseCluster, err := h.kubeConnector.GetDatabaseCluster(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to get database cluster %s/%s: %w", namespace, name, err)
	}

	response := &api.DatabaseClusterPitr{}
	// for PG there is no such thing as enabling PITR, it is always enabled
	if !databaseCluster.Spec.Backup.PITR.Enabled && databaseCluster.Spec.Engine.Type != everestv1alpha1.DatabaseEnginePostgresql {
		return response, nil
	}

	backups, err := h.kubeConnector.ListDatabaseClusterBackups(ctx,
		ctrlclient.InNamespace(namespace),
		ctrlclient.MatchingLabels{common.DatabaseClusterNameLabel: name},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list database cluster backups: %w", err)
	}
	if len(backups.Items) == 0 {
		return response, nil
	}

	latestBackup := latestSuccessfulBackup(backups.Items)
	if latestBackup == nil || latestBackup.Status.CreatedAt == nil {
		return response, nil
	}

	backupTime := latestBackup.Status.CompletedAt.UTC()
	var latest *time.Time
	// if there is the LatestRestorableTime set in the CR, use it
	// except of psmdb which has a bug https://perconadev.atlassian.net/browse/K8SPSMDB-1186
	if latestBackup.Status.LatestRestorableTime != nil && databaseCluster.Spec.Engine.Type != everestv1alpha1.DatabaseEnginePSMDB {
		t := &latestBackup.Status.LatestRestorableTime.Time
		if t.After(backupTime) {
			latest = t
		}
	} else {
		// otherwise use heuristics based on the UploadInterval
		heuristicsInterval := getDefaultUploadInterval(databaseCluster.Spec.Engine, databaseCluster.Spec.Backup.PITR.UploadIntervalSec)
		latest = latestRestorableDate(time.Now(), backupTime, heuristicsInterval)
	}

	response.LatestDate = latest
	if response.LatestDate != nil {
		response.EarliestDate = &backupTime
	}
	response.LatestBackupName = &latestBackup.Name
	response.Gaps = &latestBackup.Status.Gaps
	return response, nil
}

//nolint:gochecknoglobals
var everestAPIConstantBackoff = backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 10) //nolint:mnd

func (h *k8sHandler) ensureBackupStorageProtection(ctx context.Context, backup *everestv1alpha1.DatabaseClusterBackup) error {
	// We wrap this logic in a retry loop to reduce the chances of resource conflicts.
	return backoff.Retry(func() error {
		backup, err := h.kubeConnector.GetDatabaseClusterBackup(ctx, types.NamespacedName{Namespace: backup.GetNamespace(), Name: backup.GetName()})
		if err != nil {
			return err
		}
		controllerutil.AddFinalizer(backup, everestv1alpha1.DBBackupStorageProtectionFinalizer)
		controllerutil.AddFinalizer(backup, common.ForegroundDeletionFinalizer)
		_, err = h.kubeConnector.UpdateDatabaseClusterBackup(ctx, backup)
		return err
	},
		backoff.WithContext(everestAPIConstantBackoff, ctx),
	)
}

func (h *k8sHandler) ensureBackupForegroundDeletion(ctx context.Context, backup *everestv1alpha1.DatabaseClusterBackup) error {
	// We wrap this logic in a retry loop to reduce the chances of resource conflicts.
	return backoff.Retry(func() error {
		backup, err := h.kubeConnector.GetDatabaseClusterBackup(ctx, types.NamespacedName{Namespace: backup.GetNamespace(), Name: backup.GetName()})
		if err != nil {
			return err
		}
		controllerutil.AddFinalizer(backup, common.ForegroundDeletionFinalizer)
		_, err = h.kubeConnector.UpdateDatabaseClusterBackup(ctx, backup)
		return err
	},
		backoff.WithContext(everestAPIConstantBackoff, ctx),
	)
}

func (h *k8sHandler) connectionURL(ctx context.Context, db *everestv1alpha1.DatabaseCluster, user, password string) *string {
	if db.Status.Hostname == "" {
		return nil
	}
	var url string
	defaultHost := net.JoinHostPort(db.Status.Hostname, fmt.Sprint(db.Status.Port))
	switch db.Spec.Engine.Type {
	case everestv1alpha1.DatabaseEnginePXC:
		url = queryEscapedURL("jdbc:mysql", user, password, defaultHost)
	case everestv1alpha1.DatabaseEnginePSMDB:
		url = queryEscapedURL("mongodb", user, password, psmdbHosts(ctx, db, h.kubeConnector.ListPods))
	case everestv1alpha1.DatabaseEnginePostgresql:
		url = queryEscapedURL("postgres", user, password, defaultHost)
	}
	return pointer.ToString(url)
}

// Using own format instead of url.URL bc it uses the password encoding policy which does not encode char like ','
// however such char may appear in the db passwords.
func queryEscapedURL(scheme, user, password, hosts string) string {
	format := "%s://%s:%s@%s"
	return fmt.Sprintf(format, scheme, user, url.QueryEscape(password), hosts)
}

func psmdbHosts(
	ctx context.Context,
	db *everestv1alpha1.DatabaseCluster,
	getPods func(ctx context.Context, opts ...ctrlclient.ListOption) (*corev1.PodList, error),
) string {
	// for sharded clusters use a single entry point (mongos)
	if db.Spec.Sharding != nil && db.Spec.Sharding.Enabled {
		return net.JoinHostPort(db.Status.Hostname, fmt.Sprint(db.Status.Port))
	}

	return db.Status.Hostname
}

func latestSuccessfulBackup(backups []everestv1alpha1.DatabaseClusterBackup) *everestv1alpha1.DatabaseClusterBackup {
	slices.SortFunc(backups, sortFunc)
	for _, backup := range backups {
		if backup.Status.State == everestv1alpha1.BackupSucceeded {
			return &backup
		}
	}
	return nil
}

func sortFunc(a, b everestv1alpha1.DatabaseClusterBackup) int {
	if a.Status.CreatedAt == nil {
		return 1
	}
	if b.Status.CreatedAt == nil {
		return -1
	}
	if b.Status.CreatedAt.After(a.Status.CreatedAt.Time) {
		return 1
	}
	return -1
}

func getDefaultUploadInterval(engine everestv1alpha1.Engine, uploadInterval *int) int {
	version, err := goversion.NewVersion(engine.Version)
	if err != nil {
		return 0
	}
	switch engine.Type {
	case everestv1alpha1.DatabaseEnginePXC:
		// latest restorable time appeared in PXC 1.14.0
		if common.CheckConstraint(version, "<1.14.0") {
			return valueOrDefault(uploadInterval, pxcDefaultUploadInterval)
		}
	case everestv1alpha1.DatabaseEnginePSMDB:
		// latest restorable time appeared in PSMDB 1.16.0, however it's not reliable https://perconadev.atlassian.net/browse/K8SPSMDB-1186
		// so we still use heuristics
		return valueOrDefault(uploadInterval, psmdbDefaultUploadInterval)
	case everestv1alpha1.DatabaseEnginePostgresql:
		// latest restorable time appeared in PG 2.4.0, however it's not reliable https://perconadev.atlassian.net/browse/K8SPG-681
		// so we still use heuristics
		return valueOrDefault(uploadInterval, pgDefaultUploadInterval)
	}
	// for newer versions don't use the heuristics, so return 0 upload interval
	return 0
}

func latestRestorableDate(now, latestBackupTime time.Time, heuristicsInterval int) *time.Time {
	// if heuristicsInterval is not set, then no latest restorable date available
	if heuristicsInterval == 0 {
		return nil
	}
	// delete nanoseconds since they're not accepted by restoration
	now = now.Truncate(time.Duration(now.Nanosecond()) * time.Nanosecond)
	// heuristic: latest restorable date is now minus uploadInterval
	date := now.Add(-time.Duration(heuristicsInterval) * time.Second).UTC()
	// not able to restore if after the latest backup passed less than uploadInterval time,
	// so in that case return nil
	if latestBackupTime.After(date) {
		return nil
	}
	return &date
}

func valueOrDefault(value *int, defaultValue int) int {
	if value == nil {
		return defaultValue
	}
	return *value
}

func (h *k8sHandler) CreateDatabaseClusterSecret(ctx context.Context,
	namespace, dbName string,
	secret *corev1.Secret,
) (*corev1.Secret, error) {
	secretCpy := secret.DeepCopy()
	secretCpy.SetNamespace(namespace)
	secretCpy.SetLabels(map[string]string{
		common.DatabaseClusterNameLabel: dbName,
	})
	return h.kubeConnector.CreateSecret(ctx, secretCpy)
}
