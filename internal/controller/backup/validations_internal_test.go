// everest
// Copyright (C) 2023 Percona LLC
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

package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// azureClass is a third-party class with its own validations.
func azureClass(validations ...common.ClassValidation) *backupv1alpha1.BackupStorageClass {
	return &backupv1alpha1.BackupStorageClass{
		ObjectMeta: metav1.ObjectMeta{Name: "azure-blob.example.com", UID: "uid-azure", Generation: 1},
		Spec: backupv1alpha1.BackupStorageClassSpec{
			ParametersSchema: common.ParametersSchema{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
				Type:       "object",
				Properties: map[string]apiextensionsv1.JSONSchemaProps{"account": {Type: "string"}},
			}},
			AuthModes:   []common.AuthMode{{Name: common.AuthModeWorkloadIdentity}},
			Validations: validations,
		},
	}
}

func azureInput(container, account string) AcceptInput {
	return AcceptInput{
		Location:       backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: container}},
		Parameters:     &runtime.RawExtension{Raw: []byte(`{"account":"` + account + `"}`)},
		Authentication: common.Authentication{Mode: common.AuthModeWorkloadIdentity},
	}
}

func TestAcceptValidations(t *testing.T) {
	t.Parallel()

	rules := []common.ClassValidation{
		{Expression: "self.location.objectStore.bucket.size() <= 24", Message: "a container name is at most 24 characters", Reason: backupv1alpha1.ReasonInvalidLocation},
		{Expression: "self.parameters.account.startsWith('acct')", Message: "the account must start with acct", Reason: backupv1alpha1.ReasonInvalidParameters},
	}
	tests := []struct {
		name        string
		class       *backupv1alpha1.BackupStorageClass
		in          AcceptInput
		wantReason  string
		wantMessage string
	}{
		{name: "valid", class: azureClass(rules...), in: azureInput("backups", "acct1"), wantReason: backupv1alpha1.ReasonAccepted},
		{
			name: "location rule", class: azureClass(rules...), in: azureInput("a-container-name-that-is-too-long", "acct1"),
			wantReason: backupv1alpha1.ReasonInvalidLocation, wantMessage: "a container name is at most 24 characters",
		},
		{
			name: "parameters rule", class: azureClass(rules...), in: azureInput("backups", "other"),
			wantReason: backupv1alpha1.ReasonInvalidParameters, wantMessage: "the account must start with acct",
		},
		{name: "no validations", class: azureClass(), in: azureInput("a-container-name-that-is-too-long", "other"), wantReason: backupv1alpha1.ReasonAccepted},
		{
			name:       "validations that do not compile",
			class:      azureClass(common.ClassValidation{Expression: "self.parameters.region == 'x'", Message: "m", Reason: backupv1alpha1.ReasonInvalidParameters}),
			in:         azureInput("backups", "acct1"),
			wantReason: backupv1alpha1.ReasonInvalidClass, wantMessage: "has invalid validations",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Accept(tt.class, tt.in)
			assert.Equal(t, tt.wantReason, got.Reason, got.Message)
			assert.Contains(t, got.Message, tt.wantMessage)
		})
	}
}

func TestValidationCache(t *testing.T) {
	t.Parallel()

	var cache validationCache
	class := azureClass(common.ClassValidation{Expression: "self.parameters.account != ''", Message: "m", Reason: backupv1alpha1.ReasonInvalidParameters})
	first, err := cache.rules(class)
	require.NoError(t, err)
	again, err := cache.rules(class)
	require.NoError(t, err)
	assert.Same(t, first, again, "compiled once per generation")

	changed := class.DeepCopy()
	changed.Generation = 2
	changed.Spec.Validations[0].Expression = "self.parameters.region == 'x'"
	_, err = cache.rules(changed)
	require.Error(t, err, "a new generation is compiled again")

	_, err = cache.rules(azureClass())
	require.NoError(t, err)
	noUID := azureClass()
	noUID.UID = ""
	_, err = cache.rules(noUID)
	require.ErrorIs(t, err, errNoUID)
}
