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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/contract"
)

func TestAnnotationKeysAreTheContractKeys(t *testing.T) {
	t.Parallel()

	assert.Equal(t, contract.VersionAnnotation, versionAnnotation)
	assert.Equal(t, contract.MinVersionAnnotation, minVersionAnnotation)
	assert.Equal(t, contract.UpgradableFromAnnotation, upgradableFromAnnotation)
}

// Annotation lines of stamped manifests.
const (
	windowLines = "    openeverest.io/contract-version: 2.1.0\n" +
		"    openeverest.io/min-contract-version: 2.0.0\n" +
		"    openeverest.io/upgradable-from: 2.0.0,2.0.1\n"
	freshInstallLines = "    openeverest.io/contract-version: 2.1.1-dev\n" +
		"    openeverest.io/min-contract-version: 2.0.0\n" +
		"    openeverest.io/upgradable-from: \"\"\n"
	staleLines = "    openeverest.io/contract-version: 2.0.0\n" +
		"    openeverest.io/min-contract-version: 2.0.0\n" +
		"    openeverest.io/upgradable-from: \"\"\n"
	classMetadata        = "metadata:\n  name: s3\n"
	stampedClassMetadata = "metadata:\n  annotations:\n    openeverest.io/contract-version: 2.1.0\n  name: s3\n"
)

// stampedCRD returns testCRD with annotation lines after controller-gen's.
func stampedCRD(lines string) string {
	const anchor = "    controller-gen.kubebuilder.io/version: v0.18.0\n"
	return strings.Replace(testCRD, anchor, anchor+lines, 1)
}

func TestStampManifest(t *testing.T) {
	t.Parallel()

	window := map[string]string{
		versionAnnotation:        "2.1.0",
		minVersionAnnotation:     "2.0.0",
		upgradableFromAnnotation: "2.0.0,2.0.1",
	}
	freshInstall := map[string]string{
		versionAnnotation:        "2.1.1-dev",
		minVersionAnnotation:     "2.0.0",
		upgradableFromAnnotation: "",
	}
	version := map[string]string{versionAnnotation: "2.1.0"}
	tests := []struct {
		name        string
		manifest    string
		annotations map[string]string
		want        string
	}{
		{name: "CRD", manifest: testCRD, annotations: window, want: stampedCRD(windowLines)},
		{name: "fresh install only", manifest: testCRD, annotations: freshInstall, want: stampedCRD(freshInstallLines)},
		{name: "stale stamp", manifest: stampedCRD(staleLines), annotations: window, want: stampedCRD(windowLines)},
		{
			name:        "class without annotations",
			manifest:    testBackupStorageClass,
			annotations: version,
			want:        strings.Replace(testBackupStorageClass, classMetadata, stampedClassMetadata, 1),
		},
		{
			name:        "metadata last",
			manifest:    "kind: BackupStorageClass\nspec: {}\n" + classMetadata + "\n",
			annotations: version,
			want:        "kind: BackupStorageClass\nspec: {}\n" + stampedClassMetadata + "\n",
		},
		{
			name:        "comment after metadata",
			manifest:    classMetadata + "\n# The spec.\nspec: {}\n",
			annotations: version,
			want:        stampedClassMetadata + "\n# The spec.\nspec: {}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := stampManifest([]byte(tt.manifest), tt.annotations)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))

			again, err := stampManifest(got, tt.annotations)
			require.NoError(t, err)
			assert.Equal(t, string(got), string(again), "stamping is idempotent")
		})
	}
}

func TestStampManifestErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest string
		wantErr  string
	}{
		{name: "no metadata", manifest: "kind: X\nspec: {}\n", wantErr: "no metadata"},
		{name: "flow style", manifest: "kind: X\nmetadata: {name: x}\n", wantErr: "no top-level metadata block in block style"},
		{name: "two documents", manifest: "metadata:\n  name: a\n---\nmetadata:\n  name: b\n", wantErr: "more than one top-level metadata block"},
		{name: "not YAML", manifest: "metadata: [\n", wantErr: "yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := stampManifest([]byte(tt.manifest), map[string]string{versionAnnotation: "2.1.0"})
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
