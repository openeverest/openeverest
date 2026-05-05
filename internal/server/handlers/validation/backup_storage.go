package validation

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest-operator/utils"
	operatorUtils "github.com/percona/everest-operator/utils"
	"github.com/percona/everest/api"
	"github.com/percona/everest/cmd/config"
)

const (
	timeoutS3AccessSec = 2
)

var (
	errDuplicatedBackupStorage = func(namespace string) error {
		return fmt.Errorf("backup storage with the same url, bucket and region already exists in namespace='%s'", namespace)
	}
	// Used backup storage errors
	errEditInUseBackupStorage = func(namespace, name string) error {
		return fmt.Errorf("backup storage='%s' in namespace='%s' is used and cannot be updated", name, namespace)
	}
	errDeleteInUseBackupStorage = func(namespace, name string) error {
		return fmt.Errorf("backup storage='%s' in namespace='%s' is used and cannot be deleted", name, namespace)
	}
)

func (h *validateHandler) ListBackupStorages(ctx context.Context, namespace string) (*everestv1alpha1.BackupStorageList, error) {
	return h.next.ListBackupStorages(ctx, namespace)
}

func (h *validateHandler) GetBackupStorage(ctx context.Context, namespace, name string) (*everestv1alpha1.BackupStorage, error) {
	return h.next.GetBackupStorage(ctx, namespace, name)
}

func (h *validateHandler) CreateBackupStorage(ctx context.Context, namespace string, req *api.CreateBackupStorageParams) (*everestv1alpha1.BackupStorage, error) {
	bsList, err := h.kubeConnector.ListBackupStorages(ctx, ctrlclient.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to ListBackupStorages: %w", err)
	}
	if err := validateCreateBackupStorageRequest(ctx, h.log, req, bsList); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	return h.next.CreateBackupStorage(ctx, namespace, req)
}

func (h *validateHandler) UpdateBackupStorage(ctx context.Context, namespace, name string, req *api.UpdateBackupStorageParams) (*everestv1alpha1.BackupStorage, error) {
	bs, err := h.kubeConnector.GetBackupStorage(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to GetBackupStorage: %w", err)
	}
	secret, err := h.kubeConnector.GetSecret(ctx, types.NamespacedName{Namespace: namespace, Name: bs.Spec.CredentialsSecretName})
	if err != nil {
		return nil, fmt.Errorf("failed to GetSecret: %w", err)
	}
	if err := h.validateUpdateBackupStorageRequest(ctx, h.log, req, bs, secret); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	return h.next.UpdateBackupStorage(ctx, namespace, name, req)
}

func (h *validateHandler) DeleteBackupStorage(ctx context.Context, namespace, name string) error {
	var bs *metav1.PartialObjectMetadata
	var err error
	if bs, err = h.kubeConnector.GetBackupStorageMeta(ctx, types.NamespacedName{Namespace: namespace, Name: name}); err != nil {
		return err
	}

	if operatorUtils.IsEverestObjectInUse(bs) {
		// backup storage is used by some DB cluster
		return errors.Join(ErrInvalidRequest, errDeleteInUseBackupStorage(namespace, name))
	}
	return h.next.DeleteBackupStorage(ctx, namespace, name)
}

func validateCreateBackupStorageRequest(
	ctx context.Context,
	l *zap.SugaredLogger,
	params *api.CreateBackupStorageParams,
	existingStorages *everestv1alpha1.BackupStorageList,
) error {
	if slices.ContainsFunc(existingStorages.Items, func(storage everestv1alpha1.BackupStorage) bool {
		// Check if the storage already exists with the same region, endpoint URL, and bucket name
		return storage.Spec.Region == params.Region &&
			storage.Spec.EndpointURL == pointer.GetString(params.Url) &&
			storage.Spec.Bucket == params.BucketName
	}) {
		return errDuplicatedBackupStorage(existingStorages.Items[0].GetNamespace())
	}

	if err := utils.ValidateEverestResourceName(params.Name, "name"); err != nil {
		return err
	}

	if err := validateBucketName(params.BucketName); err != nil {
		return err
	}

	if params.Url != nil {
		if ok := utils.ValidateURL(*params.Url); !ok {
			err := ErrInvalidURL("url")
			return err
		}
	}

	if params.Type == api.CreateBackupStorageParamsTypeS3 {
		if params.Region == "" {
			return errors.New("region is required when using S3 storage type")
		}
	}

	// check data access
	if err := validateStorageAccessByCreate(ctx, params, l); err != nil {
		l.Error(err)
		return err
	}

	return nil
}

func validateBucketName(s string) error {
	// sanitize: accept only lowercase letters, numbers, dots and hyphens.
	// can be applied to both s3 bucket name and azure container name.
	bucketRegex := `^[a-z0-9\.\-]{3,63}$`
	re := regexp.MustCompile(bucketRegex)
	if !re.MatchString(s) {
		return errInvalidBucketName
	}

	return nil
}

func validateStorageAccessByCreate(ctx context.Context, params *api.CreateBackupStorageParams, l *zap.SugaredLogger) error {
	switch params.Type {
	case api.CreateBackupStorageParamsTypeS3:
		return s3Access(l, params.Url, params.AccessKey, params.SecretKey, params.BucketName, params.Region, pointer.Get(params.VerifyTLS), pointer.Get(params.ForcePathStyle))
	case api.CreateBackupStorageParamsTypeAzure:
		return azureAccess(ctx, l, params.AccessKey, params.SecretKey, params.BucketName)
	default:
		return ErrCreateStorageNotSupported(string(params.Type))
	}
}

func validateBackupStorageAccess(
	ctx context.Context,
	sType string,
	url *string,
	bucketName, region, accessKey, secretKey string,
	verifyTLS bool,
	forcePathStyle bool,
	l *zap.SugaredLogger,
) error {
	switch sType {
	case string(api.BackupStorageTypeS3):
		if region == "" {
			return errors.New("region is required when using S3 storage type")
		}
		if err := s3Access(l, url, accessKey, secretKey, bucketName, region, verifyTLS, forcePathStyle); err != nil {
			return err
		}
	case string(api.BackupStorageTypeAzure):
		if err := azureAccess(ctx, l, accessKey, secretKey, bucketName); err != nil {
			return err
		}
	default:
		return ErrUpdateStorageNotSupported(sType)
	}

	return nil
}

//nolint:funlen
func s3Access(
	l *zap.SugaredLogger,
	endpoint *string,
	accessKey, secretKey, bucketName, region string,
	verifyTLS bool,
	forcePathStyle bool,
) error {
	if config.Debug {
		return nil
	}

	if endpoint != nil && *endpoint == "" {
		endpoint = nil
	}

	c := http.DefaultClient
	c.Timeout = timeoutS3AccessSec * time.Second
	c.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !verifyTLS}, //nolint:gosec
	}
	// Create a new session with the provided credentials
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         endpoint,
		Region:           aws.String(region),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		HTTPClient:       c,
		S3ForcePathStyle: aws.Bool(forcePathStyle),
	})
	if err != nil {
		l.Error(err)
		return errors.New("could not initialize S3 session")
	}

	// Create a new S3 client with the session
	svc := s3.New(sess)

	_, err = svc.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		l.Error(err)
		return errors.New("unable to connect to s3. Check your credentials")
	}

	testKey := "everest-write-test"
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Body:   bytes.NewReader([]byte{}),
		Key:    aws.String(testKey),
	})
	if err != nil {
		l.Error(err)
		return errors.New("could not write to S3 bucket")
	}

	_, err = svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
	})
	if err != nil {
		l.Error(err)
		return errors.New("could not read from S3 bucket")
	}

	_, err = svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return errors.New("could not list objects in S3 bucket")
	}

	_, err = svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
	})
	if err != nil {
		l.Error(err)
		return errors.New("could not delete an object from S3 bucket")
	}

	return nil
}

func azureAccess(ctx context.Context, l *zap.SugaredLogger, accountName, accountKey, containerName string) error {
	if config.Debug {
		return nil
	}

	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		l.Error(err)
		return errors.New("could not initialize Azure credentials")
	}

	client, err := azblob.NewClientWithSharedKeyCredential(fmt.Sprintf("https://%s.blob.core.windows.net/", url.PathEscape(accountName)), cred, nil)
	if err != nil {
		l.Error(err)
		return errors.New("could not initialize Azure client")
	}

	pager := client.NewListBlobsFlatPager(containerName, nil)
	if pager.More() {
		if _, err := pager.NextPage(ctx); err != nil {
			l.Error(err)
			return errors.New("could not list blobs in Azure container")
		}
	}

	blobName := "everest-test-blob"
	if _, err = client.UploadBuffer(ctx, containerName, blobName, []byte{}, nil); err != nil {
		l.Error(err)
		return errors.New("could not write to Azure container")
	}

	if _, err = client.DownloadBuffer(ctx, containerName, blobName, []byte{}, nil); err != nil {
		l.Error(err)
		return errors.New("could not read from Azure container")
	}

	if _, err = client.DeleteBlob(ctx, containerName, blobName, nil); err != nil {
		l.Error(err)
		return errors.New("could not delete a blob from Azure container")
	}

	return nil
}

func basicStorageParamsAreChanged(bs *everestv1alpha1.BackupStorage, params *api.UpdateBackupStorageParams) bool {
	if params.BucketName != nil && bs.Spec.Bucket != pointer.GetString(params.BucketName) {
		return true
	}
	if params.Region != nil && bs.Spec.Region != pointer.GetString(params.Region) {
		return true
	}
	return false
}

//nolint:funlen
func (h *validateHandler) validateUpdateBackupStorageRequest(
	ctx context.Context,
	l *zap.SugaredLogger,
	params *api.UpdateBackupStorageParams,
	existing *everestv1alpha1.BackupStorage,
	secret *corev1.Secret,
) error {
	if operatorUtils.IsEverestObjectInUse(existing) && basicStorageParamsAreChanged(existing, params) {
		return errEditInUseBackupStorage(existing.GetNamespace(), existing.GetName())
	}

	existingStorages, err := h.kubeConnector.ListBackupStorages(ctx, ctrlclient.InNamespace(existing.GetNamespace()))
	if err != nil {
		return err
	}
	if duplicate := validateDuplicateStorageByUpdate(existing.GetName(), existing, existingStorages, params); duplicate {
		return errDuplicatedBackupStorage(existing.GetNamespace())
	}

	url := &existing.Spec.EndpointURL
	if params.Url != nil {
		if ok := operatorUtils.ValidateURL(*params.Url); !ok {
			err := ErrInvalidURL("url")
			return err
		}
		url = params.Url
	}

	if params.BucketName != nil {
		if err := validateBucketName(*params.BucketName); err != nil {
			return err
		}
	}

	accessKey := string(secret.Data["AWS_ACCESS_KEY_ID"])
	if params.AccessKey != nil {
		accessKey = *params.AccessKey
	}
	secretKey := string(secret.Data["AWS_SECRET_ACCESS_KEY"])
	if params.SecretKey != nil {
		secretKey = *params.SecretKey
	}

	bucketName := existing.Spec.Bucket
	if params.BucketName != nil {
		bucketName = *params.BucketName
	}

	region := existing.Spec.Region
	if params.Region != nil {
		region = *params.Region
	}

	err = validateBackupStorageAccess(ctx, string(existing.Spec.Type), url, bucketName, region, accessKey, secretKey, pointer.Get(params.VerifyTLS), pointer.Get(params.ForcePathStyle), l)
	if err != nil {
		return err
	}
	return nil
}

func validateDuplicateStorageByUpdate(
	currentStorageName string,
	currentStorage *everestv1alpha1.BackupStorage,
	existingStorages *everestv1alpha1.BackupStorageList,
	params *api.UpdateBackupStorageParams,
) bool {
	// Construct the combined key for comparison
	toCompare := regionOrDefault(params, currentStorage.Spec.Region) +
		bucketNameOrDefault(params, currentStorage.Spec.Bucket) +
		urlOrDefault(params, currentStorage.Spec.EndpointURL)

	for _, s := range existingStorages.Items {
		if s.Name == currentStorageName {
			continue
		}
		if s.Spec.Region+s.Spec.Bucket+s.Spec.EndpointURL == toCompare {
			return true
		}
	}
	return false
}

func regionOrDefault(params *api.UpdateBackupStorageParams, defaultRegion string) string {
	if params.Region != nil {
		return *params.Region
	}
	return defaultRegion
}

func bucketNameOrDefault(params *api.UpdateBackupStorageParams, defaultBucketName string) string {
	if params.BucketName != nil {
		return *params.BucketName
	}
	return defaultBucketName
}

func urlOrDefault(params *api.UpdateBackupStorageParams, defaultURL string) string {
	if params.Url != nil {
		return *params.Url
	}
	return defaultURL
}
