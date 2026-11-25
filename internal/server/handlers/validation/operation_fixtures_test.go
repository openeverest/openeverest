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

package validation

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// s3StorageClass is the BackupStorageClass of every judgedStorage.
var s3StorageClass = &backupv1alpha1.BackupStorageClass{ //nolint:gochecknoglobals // shared test fixture
	ObjectMeta: metav1.ObjectMeta{Name: "s3", UID: "s3-uid", Generation: 1},
}

// judgedClass returns a class executed by executor with the verdict core
// publishes for it at the class's generation.
func judgedClass(name, executor string, compatibility backupv1alpha1.Compatibility, reason string) *backupv1alpha1.BackupClass {
	return &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(name + "-uid"), Generation: 1},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				ProviderRef:  common.ObjectRef{Name: executor},
				SupportsPITR: true,
			},
			SupportsImport: true,
		},
		Status: backupv1alpha1.BackupClassStatus{
			Providers: []backupv1alpha1.BackupClassProviderCompatibility{{
				Name:            executor,
				Compatibility:   compatibility,
				Reason:          reason,
				Message:         "published by the test",
				ClassGeneration: 1,
			}},
		},
	}
}

// storageVerdict is a verdict core publishes on a storage for class.
type storageVerdict struct {
	class         *backupv1alpha1.BackupClass
	compatibility backupv1alpha1.Compatibility
	reason        string
}

// judgedStorage returns an s3 storage with the verdicts core publishes for
// the current objects.
func judgedStorage(name, namespace string, verdicts ...storageVerdict) *backupv1alpha1.BackupStorage {
	s := &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Generation: 1},
		Spec:       backupv1alpha1.BackupStorageSpec{BackupStorageClassRef: common.ObjectRef{Name: s3StorageClass.Name}},
	}
	for _, v := range verdicts {
		s.Status.BackupClasses = append(s.Status.BackupClasses, backupv1alpha1.BackupClassVerdict{
			Name:          v.class.Name,
			Compatibility: v.compatibility,
			Reason:        v.reason,
			Message:       "published by the test",
			Observed: backupv1alpha1.BackupClassVerdictInputs{
				StorageGeneration:      s.Generation,
				ClassUID:               v.class.UID,
				ClassGeneration:        v.class.Generation,
				StorageClassUID:        s3StorageClass.UID,
				StorageClassGeneration: s3StorageClass.Generation,
			},
		})
	}
	return s
}

// assertWarnings asserts that got holds one warning containing each of want,
// and nothing else.
func assertWarnings(t *testing.T, want, got []string) {
	t.Helper()
	assert.Len(t, got, len(want), "warnings: %q", got)
	for _, w := range want {
		assert.True(t, slices.ContainsFunc(got, func(g string) bool { return strings.Contains(g, w) }),
			"no warning contains %q: %q", w, got,
		)
	}
}
