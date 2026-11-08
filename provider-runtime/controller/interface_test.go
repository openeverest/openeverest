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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

func TestSetShippedBackupClasses(t *testing.T) {
	t.Parallel()

	var p BaseProvider
	assert.Empty(t, p.ShippedBackupClasses())

	classes := []backupv1alpha1.BackupClass{{
		ObjectMeta: metav1.ObjectMeta{Name: "pbm"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:   backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{ProviderRef: common.ObjectRef{Name: "mongo"}},
		},
	}}
	p.SetShippedBackupClasses(classes)
	classes[0].Spec.ProviderManaged.ProviderRef.Name = "changed"

	got := p.ShippedBackupClasses()
	assert.Len(t, got, 1)
	assert.Equal(t, "mongo", got[0].Spec.ExecutorProvider(), "the provider keeps its own copy")
}
