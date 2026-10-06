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

package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func (p *testProvider) SyncBackup(c *controller.Context, backup *backupv1alpha1.Backup) (controller.BackupExecutionStatus, error) {
	if backup.Status.State == backupv1alpha1.BackupStateSucceeded {
		return controller.BackupExecutionStatus{
			State: backup.Status.State, StartedAt: backup.Status.StartedAt,
			CompletedAt: backup.Status.CompletedAt, Size: backup.Status.Size,
		}, nil
	}
	switch backup.Annotations[controlAnnotation] {
	case controlFail:
		return controller.BackupExecutionStatus{State: backupv1alpha1.BackupStateFailed, Message: "test failure requested"}, nil
	case controlWait:
		return controller.BackupExecutionStatus{State: backupv1alpha1.BackupStateRunning, Message: "test is waiting"}, nil
	}
	storage, err := c.BackupStorage(backup.Spec.StorageRef.Name)
	if err != nil {
		return controller.BackupExecutionStatus{}, err
	}
	s3Client, err := controller.NewS3Client(c.Context(), c.Client(), storage)
	if err != nil {
		return controller.BackupExecutionStatus{}, err
	}
	key, err := backupObjectKey(backup)
	if err != nil {
		return controller.BackupExecutionStatus{}, err
	}
	if backup.Spec.Origin.InstanceRef == nil {
		return controller.BackupExecutionStatus{}, fmt.Errorf("backup %q has no source instance", backup.Name)
	}
	marker := []byte(fmt.Sprintf("%s/%s/%s\n", backup.Namespace, backup.Spec.Origin.InstanceRef.Name, backup.Name))
	_, err = s3Client.PutObject(c.Context(), &s3.PutObjectInput{
		Bucket: aws.String(storage.Spec.S3.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(marker),
	})
	if err != nil {
		return controller.BackupExecutionStatus{}, fmt.Errorf("write test backup: %w", err)
	}
	now := metav1.Now()
	size := strconv.Itoa(len(marker))
	return controller.BackupExecutionStatus{
		State: backupv1alpha1.BackupStateSucceeded, StartedAt: &now, CompletedAt: &now, Size: &size,
	}, nil
}

func (p *testProvider) SyncRestore(c *controller.Context, restore *backupv1alpha1.Restore) (controller.RestoreExecutionStatus, error) {
	if restore.Status.State == backupv1alpha1.RestoreStateSucceeded {
		return controller.RestoreExecutionStatus{
			State: restore.Status.State, StartedAt: restore.Status.StartedAt, CompletedAt: restore.Status.CompletedAt,
		}, nil
	}
	switch restore.Annotations[controlAnnotation] {
	case controlFail:
		return controller.RestoreExecutionStatus{State: backupv1alpha1.RestoreStateFailed, Message: "test failure requested"}, nil
	case controlWait:
		return controller.RestoreExecutionStatus{State: backupv1alpha1.RestoreStateRunning, Message: "test is waiting"}, nil
	}
	storageName, key, err := restoreObject(c, restore)
	if err != nil {
		return controller.RestoreExecutionStatus{}, err
	}
	storage, err := c.BackupStorage(storageName)
	if err != nil {
		return controller.RestoreExecutionStatus{}, err
	}
	s3Client, err := controller.NewS3Client(c.Context(), c.Client(), storage)
	if err != nil {
		return controller.RestoreExecutionStatus{}, err
	}
	if key == "" {
		source := restore.Spec.DataSource.PointInTime
		instanceName := restore.Spec.InstanceRef.Name
		if source.Source.InstanceRef != nil {
			instanceName = source.Source.InstanceRef.Name
		}
		prefix := fmt.Sprintf("%s/%s/%s/", providerName, restore.Namespace, instanceName)
		objects := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
			Bucket: aws.String(storage.Spec.S3.Bucket), Prefix: aws.String(prefix),
		})
		var latestTime time.Time
		for objects.HasMorePages() {
			page, listErr := objects.NextPage(c.Context())
			if listErr != nil {
				return controller.RestoreExecutionStatus{}, fmt.Errorf("list test backups: %w", listErr)
			}
			for _, object := range page.Contents {
				if object.LastModified == nil {
					continue
				}
				if source.RecoveryTarget == backupv1alpha1.RecoveryTargetDate &&
					(source.Date == nil || object.LastModified.After(source.Date.Time)) {
					continue
				}
				if key == "" || object.LastModified.After(latestTime) {
					key = aws.ToString(object.Key)
					latestTime = *object.LastModified
				}
			}
		}
		if key == "" {
			return controller.RestoreExecutionStatus{State: backupv1alpha1.RestoreStateFailed, Message: "no test backup at the requested point"}, nil
		}
	}
	object, err := s3Client.GetObject(c.Context(), &s3.GetObjectInput{
		Bucket: aws.String(storage.Spec.S3.Bucket), Key: aws.String(key),
	})
	if err != nil {
		return controller.RestoreExecutionStatus{}, fmt.Errorf("read test backup: %w", err)
	}
	defer object.Body.Close() //nolint:errcheck
	if _, err := io.Copy(io.Discard, object.Body); err != nil {
		return controller.RestoreExecutionStatus{}, fmt.Errorf("read test backup body: %w", err)
	}
	now := metav1.Now()
	return controller.RestoreExecutionStatus{
		State: backupv1alpha1.RestoreStateSucceeded, StartedAt: &now, CompletedAt: &now,
	}, nil
}

func (p *testProvider) CleanupBackup(c *controller.Context, backup *backupv1alpha1.Backup) (bool, error) {
	if c.ShouldRetainBackupData(backup) {
		return true, nil
	}
	storage, err := c.BackupStorage(backup.Spec.StorageRef.Name)
	if err != nil {
		return false, err
	}
	s3Client, err := controller.NewS3Client(c.Context(), c.Client(), storage)
	if err != nil {
		return false, err
	}
	key, err := backupObjectKey(backup)
	if err != nil {
		return false, err
	}
	_, err = s3Client.DeleteObject(c.Context(), &s3.DeleteObjectInput{
		Bucket: aws.String(storage.Spec.S3.Bucket), Key: aws.String(key),
	})
	if err != nil {
		return false, fmt.Errorf("delete test backup: %w", err)
	}
	return true, nil
}

func (p *testProvider) CleanupRestore(_ *controller.Context, _ *backupv1alpha1.Restore) (bool, error) {
	return true, nil
}

func backupObjectKey(backup *backupv1alpha1.Backup) (string, error) {
	if backup.Spec.Origin.Type == backupv1alpha1.BackupOriginTypeExternal {
		if backup.Spec.Origin.External == nil {
			return "", fmt.Errorf("external backup %q has no path", backup.Name)
		}
		return backup.Spec.Origin.External.Path, nil
	}
	if backup.Spec.Origin.InstanceRef == nil {
		return "", fmt.Errorf("backup %q has no source instance", backup.Name)
	}
	return strings.Join([]string{providerName, backup.Namespace, backup.Spec.Origin.InstanceRef.Name, string(backup.UID)}, "/"), nil
}

func restoreObject(c *controller.Context, restore *backupv1alpha1.Restore) (string, string, error) {
	source := restore.Spec.DataSource
	switch source.Type {
	case backupv1alpha1.DataSourceTypeBackup:
		if source.Backup == nil {
			return "", "", fmt.Errorf("restore %q has no backup reference", restore.Name)
		}
		backup := &backupv1alpha1.Backup{}
		name := client.ObjectKey{Namespace: restore.Namespace, Name: source.Backup.BackupRef.Name}
		if err := c.Client().Get(c.Context(), name, backup); err != nil {
			return "", "", fmt.Errorf("get source backup: %w", err)
		}
		key, err := backupObjectKey(backup)
		return backup.Spec.StorageRef.Name, key, err
	case backupv1alpha1.DataSourceTypePointInTime:
		if source.PointInTime == nil {
			return "", "", fmt.Errorf("restore %q has no point-in-time source", restore.Name)
		}
		return source.PointInTime.Source.StorageRef.Name, "", nil
	default:
		return "", "", fmt.Errorf("unsupported restore source %q", source.Type)
	}
}
