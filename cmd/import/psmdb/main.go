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

// Command psmdb-import is the Job-mode discovery image for Percona Backup for
// MongoDB (PBM) backups. The BackupImport controller spawns it with a payload
// file (jobspec.ImportSpec) mounted at the path given as the first argument.
//
// It lists the referenced BackupStorage, decodes each PBM metadata document
// (*.pbm.json), keeps the backups PBM marks "done", and creates one external
// Backup CR (origin.type=External) per discovered backup. Each Backup uses a
// deterministic name derived from (storageRef, path) and is labeled with the
// originating BackupImport name, so re-running discovery is idempotent and the
// controller can count what this import produced. Deduplication is enforced by
// the API server's name uniqueness: an AlreadyExists error is treated as a
// successful no-op.
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/jobspec"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	pkgcommon "github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	// pbmMetadataSuffix is the object-key suffix of the per-backup metadata
	// documents written by Percona Backup for MongoDB.
	pbmMetadataSuffix = ".pbm.json"
	// pbmStatusDone is the PBM backup status that marks a backup as complete
	// and restorable. Only backups in this state are surfaced as importable.
	pbmStatusDone = "done"
	// maxPBMMetaSize caps the size of a *.pbm.json document read from storage.
	maxPBMMetaSize = 1 << 20 // 1 MiB
)

// pbmBackupMeta is Percona Backup for MongoDB's per-backup metadata document
// (<name>.pbm.json).
type pbmBackupMeta struct {
	// Status is the backup lifecycle state; only "done" is restorable.
	Status string `json:"status"`
	// StartTS is the backup start time in Unix seconds.
	StartTS int64 `json:"start_ts"`
	// LastTransitionTS is the Unix-seconds time of the last status transition;
	// for a "done" backup this is its completion time.
	LastTransitionTS int64 `json:"last_transition_ts"`
}

func main() {
	log.SetLogger(zap.New())
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <payload-path>\n", os.Args[0])
		os.Exit(2)
	}
	if err := run(context.Background(), os.Args[1]); err != nil {
		log.Log.Error(err, "discovery failed")
		os.Exit(1)
	}
}

func run(ctx context.Context, payloadPath string) error {
	cfg, err := readPayload(payloadPath)
	if err != nil {
		return err
	}
	if cfg.Storage == nil || cfg.Storage.S3 == nil {
		return fmt.Errorf("payload has no S3 storage details; only S3 is supported")
	}

	logger := log.FromContext(ctx).WithValues(
		"backupImport", cfg.ImportName,
		"namespace", cfg.Namespace,
		"storage", cfg.StorageRef,
	)

	s3c := newS3Client(cfg.Storage.S3)
	k8sClient, err := newK8sClient()
	if err != nil {
		return err
	}

	bucket := cfg.Storage.S3.Bucket
	paginator := s3.NewListObjectsV2Paginator(s3c, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})

	discovered, created := 0, 0
	for paginator.HasMorePages() {
		page, perr := paginator.NextPage(ctx)
		if perr != nil {
			return fmt.Errorf("list objects in bucket %q: %w", bucket, perr)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if !strings.HasSuffix(key, pbmMetadataSuffix) {
				continue
			}

			data, err := getObject(ctx, s3c, bucket, key)
			if err != nil {
				return fmt.Errorf("read %q: %w", key, err)
			}

			backup, err := buildBackup(cfg, key, data)
			if err != nil {
				logger.Error(err, "skipping unparseable PBM metadata", "key", key)
				continue
			}
			if backup == nil {
				continue // not a completed backup
			}
			discovered++

			if err := k8sClient.Create(ctx, backup); err != nil {
				if apierrors.IsAlreadyExists(err) {
					logger.Info("backup already imported, skipping", "name", backup.Name)
					continue
				}
				return fmt.Errorf("create Backup %q: %w", backup.Name, err)
			}
			created++
			logger.Info("created external backup", "name", backup.Name, "path", backup.Spec.Origin.External.Path)
		}
	}

	logger.Info("discovery complete", "discovered", discovered, "created", created)
	return nil
}

// readPayload loads and decodes the mounted jobspec.ImportSpec payload.
func readPayload(path string) (*jobspec.ImportSpec, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is the controller-mounted payload
	if err != nil {
		return nil, fmt.Errorf("read payload %q: %w", path, err)
	}
	cfg := &jobspec.ImportSpec{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	if cfg.Namespace == "" || cfg.ImportName == "" || cfg.ClassRef == "" || cfg.StorageRef == "" {
		return nil, fmt.Errorf("payload missing required fields (namespace, importName, classRef, storageRef)")
	}
	return cfg, nil
}

// buildBackup decodes a single PBM metadata document and, when it describes a
// completed ("done") backup, builds the corresponding external Backup CR. It
// returns (nil, nil) for backups that are not yet complete.
func buildBackup(cfg *jobspec.ImportSpec, key string, data []byte) (*backupv1alpha1.Backup, error) {
	var meta pbmBackupMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal PBM metadata: %w", err)
	}
	if meta.Status != pbmStatusDone {
		return nil, nil
	}

	path := strings.TrimSuffix(key, pbmMetadataSuffix)
	return &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      importBackupName(cfg.StorageRef, path),
			Namespace: cfg.Namespace,
			Labels: map[string]string{
				pkgcommon.BackupImportNameLabel: cfg.ImportName,
			},
		},
		Spec: backupv1alpha1.BackupSpec{
			Origin: backupv1alpha1.BackupOrigin{
				Type: backupv1alpha1.BackupOriginTypeExternal,
				External: &backupv1alpha1.BackupOriginExternal{
					Path:        path,
					StartedAt:   metav1.Unix(meta.StartTS, 0),
					CompletedAt: metav1.Unix(meta.LastTransitionTS, 0),
				},
			},
			ClassRef:       common.ObjectRef{Name: cfg.ClassRef},
			StorageRef:     common.ObjectRef{Name: cfg.StorageRef},
			DeletionPolicy: backupv1alpha1.BackupDeletionPolicyRetain,
		},
	}, nil
}

// importBackupName derives a deterministic Backup CR name from the (storage,
// path) pair so re-running discovery never creates duplicates. The name is
// "import-" followed by a 64-bit FNV-1a hash rendered as 16 hex digits, which
// is stable, DNS-safe, and collision-resistant for practical bucket sizes.
func importBackupName(storageName, path string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(storageName))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(path))
	return fmt.Sprintf("import-%016x", h.Sum64())
}

// getObject reads the full contents of a single object from the bucket, capped
// at maxPBMMetaSize.
func getObject(ctx context.Context, s3c *s3.Client, bucket, key string) ([]byte, error) {
	resp, err := s3c.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body close
	return io.ReadAll(io.LimitReader(resp.Body, maxPBMMetaSize))
}

// newS3Client builds an S3 client from the inline storage details carried in
// the payload. Credentials are embedded by the controller from the
// BackupStorage's credentials Secret, so no Secret read happens in-job.
func newS3Client(s3Details *jobspec.S3Details) *s3.Client {
	httpClient := awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
		if !s3Details.VerifyTLS {
			if tr.TLSClientConfig == nil {
				tr.TLSClientConfig = &tls.Config{} //nolint:gosec // VerifyTLS=false is an explicit opt-out
			}
			tr.TLSClientConfig.InsecureSkipVerify = true
		}
	})

	awsCfg := aws.Config{
		Region: s3Details.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			s3Details.AccessKeyID, s3Details.SecretAccessKey, ""),
		HTTPClient: httpClient,
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if s3Details.EndpointURL != "" {
			o.BaseEndpoint = aws.String(s3Details.EndpointURL)
		}
		o.UsePathStyle = s3Details.ForcePathStyle
	})
}

// newK8sClient builds a controller-runtime client with the Backup scheme
// registered, using the in-cluster/ServiceAccount config.
func newK8sClient() (ctrlclient.Client, error) {
	scheme := runtime.NewScheme()
	if err := backupv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("add backup scheme: %w", err)
	}
	c, err := ctrlclient.New(config.GetConfigOrDie(), ctrlclient.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("build kubernetes client: %w", err)
	}
	return c, nil
}
