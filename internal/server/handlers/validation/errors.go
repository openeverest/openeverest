package validation

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/percona/everest-operator/api/everest/v1alpha1"
)

var (
	// ErrInvalidRequest is an error that appears when the request is invalid.
	ErrInvalidRequest = errors.New("invalid request")

	minStorageQuantity = resource.MustParse("1G")   //nolint:gochecknoglobals
	minCPUQuantity     = resource.MustParse("600m") //nolint:gochecknoglobals
	minMemQuantity     = resource.MustParse("512M") //nolint:gochecknoglobals

	errCannotShrinkStorageSize       = errors.New("cannot shrink storage size")
	errNotEnoughMemory               = fmt.Errorf("memory limits should be above %s", minMemQuantity.String())
	errNotEnoughCPU                  = fmt.Errorf("CPU limits should be above %s", minCPUQuantity.String())
	errNotEnoughDiskSize             = fmt.Errorf("storage size should be above %s", minStorageQuantity.String())
	errUnsupportedPXCProxy           = errors.New("you can use either HAProxy or Proxy SQL for PXC clusters")
	errUnsupportedPGProxy            = errors.New("you can use only PGBouncer as a proxy type for Postgres clusters")
	errUnsupportedPSMDBProxy         = errors.New("you can use only Mongos as a proxy type for MongoDB clusters")
	errNoNameInSchedule              = errors.New("'name' field for the backup schedules cannot be empty")
	errScheduleNoBackupStorageName   = errors.New("'backupStorageName' field cannot be empty when schedule is enabled")
	errPitrNoBackupStorageName       = errors.New("'backupStorageName' field cannot be empty when pitr is enabled")
	errNoResourceDefined             = errors.New("please specify resource limits for the cluster")
	errPitrUploadInterval            = errors.New("'uploadIntervalSec' should be more than 0")
	errPXCPitrS3Only                 = errors.New("point-in-time recovery only supported for s3 compatible storages")
	errPSMDBMultipleStorages         = errors.New("can't use more than one backup storage for PSMDB clusters")
	errPSMDBViolateActiveStorage     = errors.New("can't change the active storage for PSMDB clusters")
	errDataSourceConfig              = errors.New("either DBClusterBackupName, BackupSource or DataImport must be specified in the DataSource field")
	errDataSourceNoPitrDateSpecified = errors.New("pitr Date must be specified for type Date")
	errDataSourceWrongDateFormat     = errors.New("failed to parse .Spec.DataSource.Pitr.Date as 2006-01-02T15:04:05Z")
	errDataSourceNoBackupStorageName = errors.New("'backupStorageName' should be specified in .Spec.DataSource.BackupSource")
	errDataSourceNoPath              = errors.New("'path' should be specified in .Spec.DataSource.BackupSource")
	errUnsupportedPitrType           = errors.New("the given point-in-time recovery type is not supported")
	errTooManyPGStorages             = fmt.Errorf("only %d different storages are allowed in a PostgreSQL cluster", pgReposLimit)
	errInvalidBucketName             = fmt.Errorf("invalid bucketName")
	errInvalidVersion                = errors.New("invalid database engine version provided")
	errDBEngineMajorVersionUpgrade   = errors.New("database engine cannot be upgraded to a major version")
	errDBEngineMajorUpgradeNotSeq    = errors.New("database engine major version upgrade is not supported for non-sequential versions")
	errDBEngineDowngrade             = errors.New("database engine version cannot be downgraded")
	errDuplicatedSchedules           = errors.New("duplicated backup schedules are not allowed")
	errDuplicatedStoragePG           = errors.New("postgres clusters can't use the same storage for the different schedules")
	errStorageChangePG               = errors.New("the existing postgres schedules can't change their storage")
	errShardingIsNotSupported        = errors.New("sharding is not supported")
	errInsufficientShardsNumber      = errors.New("shards number should be greater than 0")
	errInsufficientCfgSrvNumber      = errors.New("sharding: minimum config servers number is 3")
	errInsufficientCfgSrvNumber1Node = errors.New("sharding: minimum config servers number for 1 node replsets is 1")
	errEvenServersNumber             = errors.New("sharding: config servers number should be odd")
	errDisableShardingNotSupported   = errors.New("sharding: disable sharding is not supported")
	errShardingEnablingNotSupported  = errors.New("sharding: enable sharding is not supported when editing db cluster")
	errShardingVersion               = errors.New("sharding is available starting PSMDB 1.17.0")
	errEvenEngineReplicas            = errors.New("engine replicas number should be odd")
	errMaxPXCEngineReplicas          = errors.New("max replicas number for MySQL is 5")
	errMinPXCProxyReplicas           = errors.New("min replicas number for Proxy is 2")
	errEmptyName                     = errors.New("name cannot be empty")
	errEmptyNamespace                = errors.New("namespace cannot be empty")
)

// ErrUpdateStorageNotSupported appears when trying to update a storage of a type that is not supported.
func ErrUpdateStorageNotSupported(storageType string) error {
	return fmt.Errorf("updating storage is not implemented for '%s'", storageType)
}

// ErrInvalidURL when the given fieldName contains invalid URL.
func ErrInvalidURL(fieldName string) error {
	return fmt.Errorf("'%s' is an invalid URL", fieldName)
}

// ErrCreateStorageNotSupported appears when trying to create a storage of a type that is not supported.
func ErrCreateStorageNotSupported(storageType string) error {
	return fmt.Errorf("creating storage is not implemented for '%s'", storageType)
}

// ErrDuplicateSourceRange appears when a duplicated source range is found.
func ErrDuplicateSourceRange(sourceRange v1alpha1.IPSourceRange) error {
	return fmt.Errorf("duplicate expose ranges for source range %s", sourceRange)
}
