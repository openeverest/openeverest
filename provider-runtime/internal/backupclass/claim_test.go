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

package backupclass

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// shippedManifest is a class as provider-sdk generates it.
const shippedManifest = `
apiVersion: backup.openeverest.io/v1alpha1
kind: BackupClass
metadata:
  name: pbm
  annotations:
    openeverest.io/provider-version: 1.2.0
spec:
  displayName: Percona Backup for MongoDB
  description: Native backups
  executionMode: ProviderManaged
  providerManaged:
    providerRef: {name: mongo}
    supportsPITR: true
    limits: {maxPITREnabledStorages: 1}
    pitrParametersSchema:
      openAPIV3Schema:
        type: object
        properties:
          oplogSpanMin: {type: integer, minimum: 1}
  backupStorageClasses:
  - name: s3
    authModes: [AccessKey]
    parametersSchema:
      openAPIV3Schema:
        type: object
        additionalProperties: false
        properties:
          region: {type: string}
          implementation: {type: string, enum: [aws, minio]}
    tls: {customCA: Supported}
  supportsImport: true
  parametersSchema:
    openAPIV3Schema:
      type: object
      properties:
        compression: {type: string, default: s2, enum: [s2, gzip]}
  uiSchema: {backup: {sections: []}}
`

func shippedClass(t *testing.T) backupv1alpha1.BackupClass {
	t.Helper()
	var c backupv1alpha1.BackupClass
	require.NoError(t, yaml.Unmarshal([]byte(shippedManifest), &c))
	return c
}

// live returns the shipped class as a cluster would hold it at generation 4,
// changed by mutate.
func live(t *testing.T, mutate func(*backupv1alpha1.BackupClass)) *backupv1alpha1.BackupClass {
	t.Helper()
	c := shippedClass(t)
	c.Generation = 4
	if mutate != nil {
		mutate(&c)
	}
	return &c
}

func withStamp(stamp string) func(*backupv1alpha1.BackupClass) {
	return func(c *backupv1alpha1.BackupClass) {
		if stamp == "" {
			delete(c.Annotations, backupv1alpha1.ProviderVersionAnnotation)
			return
		}
		c.Annotations[backupv1alpha1.ProviderVersionAnnotation] = stamp
	}
}

func TestClaim(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		live        func(*backupv1alpha1.BackupClass)
		wantStatus  metav1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{
			name:        "the shipped class",
			wantStatus:  metav1.ConditionTrue,
			wantReason:  backupv1alpha1.ReasonAccepted,
			wantMessage: "provider mongo executes BackupClass pbm",
		},
		{
			name: "relabelled",
			live: func(c *backupv1alpha1.BackupClass) {
				c.Spec.DisplayName, c.Spec.Description = "Nightly", "What the platform team calls it"
				c.Spec.UISchema = &runtime.RawExtension{Raw: []byte(`{"backup":{"order":["compression"]}}`)}
			},
			wantStatus: metav1.ConditionTrue,
			wantReason: backupv1alpha1.ReasonAccepted,
		},
		{
			name:        "a class the provider does not ship",
			live:        func(c *backupv1alpha1.BackupClass) { c.Name = "pbm-physical" },
			wantStatus:  metav1.ConditionFalse,
			wantReason:  backupv1alpha1.ReasonUnknownClass,
			wantMessage: "provider mongo does not ship a BackupClass named pbm-physical",
		},
		{
			name: "edited capabilities",
			live: func(c *backupv1alpha1.BackupClass) {
				c.Spec.SupportsImport = false
				c.Spec.BackupStorageClasses[0].TLS.InsecureSkipVerify = backupv1alpha1.TLSOptionSupported
				c.Spec.ProviderManaged.Limits.MaxStorages = new(int32(3))
				c.Spec.ParametersSchema.OpenAPIV3Schema.Properties["level"] = apiextensionsv1.JSONSchemaProps{Type: "integer"}
			},
			wantStatus: metav1.ConditionFalse,
			wantReason: backupv1alpha1.ReasonSpecDrift,
			wantMessage: "the capabilities of BackupClass pbm differ from those of the class provider mongo ships: " +
				"spec.backupStorageClasses[s3].tls.insecureSkipVerify, spec.parametersSchema, " +
				"spec.providerManaged.limits.maxStorages, spec.supportsImport",
		},
		{
			name: "a storage class added",
			live: func(c *backupv1alpha1.BackupClass) {
				c.Spec.BackupStorageClasses = append(c.Spec.BackupStorageClasses,
					backupv1alpha1.BackupStorageClassSupport{Name: "gcs", AuthModes: []string{"AccessKey"}},
				)
			},
			wantStatus: metav1.ConditionFalse,
			wantReason: backupv1alpha1.ReasonSpecDrift,
			wantMessage: "the capabilities of BackupClass pbm differ from those of the class provider mongo ships: " +
				"spec.backupStorageClasses[gcs]",
		},
		{
			name:       "an unstamped class with the shipped capabilities",
			live:       withStamp(""),
			wantStatus: metav1.ConditionTrue,
			wantReason: backupv1alpha1.ReasonAccepted,
		},
		{
			name: "an unstamped class with other capabilities",
			live: func(c *backupv1alpha1.BackupClass) {
				withStamp("")(c)
				c.Spec.SupportsImport = false
			},
			wantStatus: metav1.ConditionFalse,
			wantReason: backupv1alpha1.ReasonSpecDrift,
		},
		{
			name: "the definition of a newer release",
			live: func(c *backupv1alpha1.BackupClass) {
				withStamp("1.3.0")(c)
				c.Spec.SupportsImport = false
			},
			wantStatus: metav1.ConditionUnknown,
			wantReason: backupv1alpha1.ReasonNewerDefinition,
			wantMessage: "BackupClass pbm is the definition of provider mongo release 1.3.0, newer than release 1.2.0, " +
				"which this provider runs: waiting for the provider to be upgraded",
		},
		{
			name: "the definition of an older release",
			live: func(c *backupv1alpha1.BackupClass) {
				withStamp("1.2.0-rc.1")(c)
				c.Spec.SupportsImport = false
			},
			wantStatus: metav1.ConditionUnknown,
			wantReason: backupv1alpha1.ReasonOlderDefinition,
			wantMessage: "BackupClass pbm is the definition of provider mongo release 1.2.0-rc.1, older than release 1.2.0, " +
				"which this provider runs: waiting for the class to be upgraded",
		},
		{
			name:       "a stamp that is not a version",
			live:       withStamp("nightly"),
			wantStatus: metav1.ConditionUnknown,
			wantReason: backupv1alpha1.ReasonNewerDefinition,
		},
		{
			name:       "the same release spelled differently",
			live:       withStamp("v1.2.0"),
			wantStatus: metav1.ConditionTrue,
			wantReason: backupv1alpha1.ReasonAccepted,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Claim(live(t, tc.live), []backupv1alpha1.BackupClass{shippedClass(t)}, "mongo")
			require.NoError(t, err)
			assert.Equal(t, backupv1alpha1.BackupClassConditionAccepted, got.Type)
			assert.Equal(t, int64(4), got.ObservedGeneration)
			assert.Equal(t, tc.wantStatus, got.Status)
			assert.Equal(t, tc.wantReason, got.Reason)
			if tc.wantMessage != "" {
				assert.Equal(t, tc.wantMessage, got.Message)
			}
		})
	}
}

func TestClaim_ShippedClassWithoutStamp(t *testing.T) {
	t.Parallel()
	shipped := shippedClass(t)
	delete(shipped.Annotations, backupv1alpha1.ProviderVersionAnnotation)

	got, err := Claim(live(t, nil), []backupv1alpha1.BackupClass{shipped}, "mongo")
	require.NoError(t, err)
	assert.Equal(t, metav1.ConditionUnknown, got.Status)
	assert.Equal(t, backupv1alpha1.ReasonNewerDefinition, got.Reason)
	assert.Contains(t, got.Message, "newer than an unstamped release")
}

// TestCapabilities_Normalization checks that what the API server does to a
// class it stores is no drift: it fills the CRD's defaults, keeps empty
// optional fields as written and re-encodes schemas.
func TestCapabilities_Normalization(t *testing.T) {
	t.Parallel()

	azure := func(c *backupv1alpha1.BackupClass) {
		c.Spec.BackupStorageClasses = append(c.Spec.BackupStorageClasses, backupv1alpha1.BackupStorageClassSupport{
			Name: "azure", AuthModes: []string{"AccessKey"},
		})
	}
	azureTLS := func(tls *backupv1alpha1.BackupStorageTLSSupport) func(*backupv1alpha1.BackupClass) {
		return func(c *backupv1alpha1.BackupClass) { c.Spec.BackupStorageClasses[1].TLS = tls }
	}

	tests := []struct {
		name    string
		shipped func(*backupv1alpha1.BackupClass)
		live    func(*backupv1alpha1.BackupClass)
	}{
		{
			name: "defaulted tls settings",
			live: func(c *backupv1alpha1.BackupClass) {
				c.Spec.BackupStorageClasses[0].TLS.InsecureSkipVerify = backupv1alpha1.TLSOptionNone
			},
		},
		{
			name:    "a tls block of defaults where the shipped class has none",
			shipped: azure,
			live: azureTLS(&backupv1alpha1.BackupStorageTLSSupport{
				CustomCA: backupv1alpha1.TLSOptionNone, InsecureSkipVerify: backupv1alpha1.TLSOptionNone,
			}),
		},
		{
			name:    "an empty tls block where the shipped class has none",
			shipped: azure,
			live:    azureTLS(&backupv1alpha1.BackupStorageTLSSupport{}),
		},
		{
			name:    "empty limits where the shipped class has none",
			shipped: func(c *backupv1alpha1.BackupClass) { c.Spec.ProviderManaged.Limits = nil },
			live: func(c *backupv1alpha1.BackupClass) {
				c.Spec.ProviderManaged.Limits = &backupv1alpha1.BackupClassLimits{}
			},
		},
		{
			name: "empty lists",
			live: func(c *backupv1alpha1.BackupClass) {
				c.Spec.InstanceConstraints.RequiredFields = []string{}
				c.Spec.ProviderManaged.PITRParametersSchema.OpenAPIV3Schema.Required = []string{}
			},
		},
		{
			name: "schema values re-encoded",
			live: func(c *backupv1alpha1.BackupClass) {
				p := c.Spec.ParametersSchema.OpenAPIV3Schema.Properties["compression"]
				p.Default = &apiextensionsv1.JSON{Raw: []byte(` "s2" `)}
				p.Enum = []apiextensionsv1.JSON{{Raw: []byte(`"s2"`)}, {Raw: []byte(` "gzip"`)}}
				c.Spec.ParametersSchema.OpenAPIV3Schema.Properties["compression"] = p
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			shipped := shippedClass(t)
			if tc.shipped != nil {
				tc.shipped(&shipped)
			}
			c := live(t, func(c *backupv1alpha1.BackupClass) {
				if tc.shipped != nil {
					tc.shipped(c)
				}
				tc.live(c)
			})
			drift, err := CapabilityDiff(&c.Spec, &shipped.Spec)
			require.NoError(t, err)
			assert.Empty(t, drift)
		})
	}
}

// TestCapabilities_KeepsSchemaContents checks that the contents of
// parameter schemas are compared as written: an empty schema accepts every
// payload, an absent one none, and additionalProperties: false restricts.
func TestCapabilities_KeepsSchemaContents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		live func(*backupv1alpha1.BackupClass)
		want []string
	}{
		{
			name: "additionalProperties false removed",
			live: func(c *backupv1alpha1.BackupClass) {
				c.Spec.BackupStorageClasses[0].ParametersSchema.OpenAPIV3Schema.AdditionalProperties = nil
			},
			want: []string{"spec.backupStorageClasses[s3].parametersSchema"},
		},
		{
			name: "an empty schema where the shipped class has none",
			live: func(c *backupv1alpha1.BackupClass) {
				c.Spec.RestoreParametersSchema.OpenAPIV3Schema = &apiextensionsv1.JSONSchemaProps{}
			},
			want: []string{"spec.restoreParametersSchema"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			shipped := shippedClass(t)
			drift, err := CapabilityDiff(&live(t, tc.live).Spec, &shipped.Spec)
			require.NoError(t, err)
			assert.Equal(t, tc.want, drift)
		})
	}
}

func TestSpecSchemaLoads(t *testing.T) {
	t.Parallel()
	s, err := specSchema()
	require.NoError(t, err)
	assert.Contains(t, s.Properties, "backupStorageClasses")
}
