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

package secretdefinition

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

func TestLabelledAndTracked(t *testing.T) {
	t.Parallel()

	managed := func(labels map[string]string) map[string]string {
		labels = maps.Clone(labels)
		labels[common.OpenEverestManagedLabel] = "true"
		return labels
	}

	tests := []struct {
		name        string
		labels      map[string]string
		wantLabel   bool
		wantTracked bool
	}{
		{
			name:        "monitoring class credentials",
			labels:      classLabels("pmm", "ServiceAccountToken"),
			wantLabel:   true,
			wantTracked: true,
		},
		{
			name:        "monitoring class credentials created through the API",
			labels:      managed(classLabels("pmm", "ServiceAccountToken")),
			wantLabel:   true,
			wantTracked: true,
		},
		{
			name:        "CA bundle",
			labels:      map[string]string{common.OpenEverestDefinitionLabel: common.OpenEverestCABundleDefinition},
			wantLabel:   true,
			wantTracked: true,
		},
		{
			name:      "provider Secret is labelled but not tracked",
			labels:    providerLabels("psmdb", "user"),
			wantLabel: true,
		},
		{
			name:      "provider Secret created through the API is not tracked either",
			labels:    managed(providerLabels("psmdb", "user")),
			wantLabel: true,
		},
		{
			name: "no labels",
		},
		{
			name:   "managed alone is not labelled",
			labels: map[string]string{common.OpenEverestManagedLabel: "true"},
		},
		{
			name:   "a definition other than ca-bundle without a source",
			labels: map[string]string{common.OpenEverestDefinitionLabel: "user"},
		},
		{
			name:   "a source without a definition",
			labels: map[string]string{common.OpenEverestMonitoringClassLabel: "pmm"},
		},
		{
			name: "two sources",
			labels: map[string]string{
				common.OpenEverestProviderLabel:        "psmdb",
				common.OpenEverestMonitoringClassLabel: "pmm",
				common.OpenEverestDefinitionLabel:      "user",
			},
		},
		{
			name:   "an empty definition",
			labels: classLabels("pmm", ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantLabel, Labelled(tt.labels), "Labelled")
			assert.Equal(t, tt.wantTracked, Tracked(tt.labels), "Tracked")
		})
	}
}

func TestWritable(t *testing.T) {
	t.Parallel()

	labels := classLabels("pmm", "ServiceAccountToken")
	managedLabels := maps.Clone(labels)
	managedLabels[common.OpenEverestManagedLabel] = "true"
	externalSecret := metav1.OwnerReference{
		APIVersion: "external-secrets.io/v1",
		Kind:       "ExternalSecret",
		Name:       "pmm-credentials",
		UID:        "1111",
		Controller: new(true),
	}
	notController := metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Name:       "inventory",
		UID:        "2222",
	}
	secret := func(labels map[string]string, owners ...metav1.OwnerReference) *corev1.Secret {
		return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "pmm-credentials", Labels: labels, OwnerReferences: owners}}
	}

	tests := []struct {
		name   string
		secret *corev1.Secret
		// wantOwner is the controller owner Writable names, as Kind/name.
		wantOwner string
		wantErr   error
	}{
		{
			name:   "created through the API",
			secret: secret(managedLabels),
		},
		{
			name:   "created through the API with an owner that is not its controller",
			secret: secret(managedLabels, notController),
		},
		{
			name:    "applied with kubectl or by Git",
			secret:  secret(labels),
			wantErr: ErrManagedOutside,
		},
		{
			name:    "managed set to something other than true",
			secret:  secret(map[string]string{common.OpenEverestManagedLabel: "yes"}),
			wantErr: ErrManagedOutside,
		},
		{
			name:      "owned by a controller",
			secret:    secret(labels, notController, externalSecret),
			wantOwner: "ExternalSecret/pmm-credentials",
		},
		{
			name:      "owned by a controller although created through the API",
			secret:    secret(managedLabels, externalSecret),
			wantOwner: "ExternalSecret/pmm-credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Writable(tt.secret)
			switch {
			case tt.wantOwner != "":
				var byController *ManagedByControllerError
				require.ErrorAs(t, err, &byController)
				assert.Equal(t, tt.wantOwner, byController.Kind+"/"+byController.Name)
				require.EqualError(t, err, "managed by "+tt.wantOwner)
				require.NotErrorIs(t, err, ErrManagedOutside)
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
				require.EqualError(t, err, "managed outside OpenEverest; change it at its source")
			default:
				require.NoError(t, err)
			}
		})
	}
}
