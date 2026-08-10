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

package common

import (
	"testing"

	versionpb "github.com/Percona-Lab/percona-version-service/versionpb"
	goversion "github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int(0), CompareVersions("1.0.0", "1.0.0"))
	assert.Equal(t, int(0), CompareVersions("1.0.0", goversion.Must(goversion.NewVersion("1.0.0"))))
	assert.Equal(t, int(0), CompareVersions("1.0.0-rc1", goversion.Must(goversion.NewVersion("1.0.0"))))
	assert.Equal(t, int(1), CompareVersions("1.0.0", "0.9.0"))
	assert.Equal(t, int(1), CompareVersions("1.0.0", goversion.Must(goversion.NewVersion("0.9.0"))))
	assert.Equal(t, int(1), CompareVersions(goversion.Must(goversion.NewVersion("1.0.0")), "0.10.1"))
	assert.Equal(t, int(1), CompareVersions(goversion.Must(goversion.NewVersion("1.0.0-rc1")), "0.10.1"))
	assert.Equal(t, int(-1), CompareVersions("1.0.0", "1.1.0"))
	assert.Equal(t, int(-1), CompareVersions("1.0.0", goversion.Must(goversion.NewVersion("1.1.0"))))
	assert.Equal(t, int(-1), CompareVersions("1.0.0-rc1", goversion.Must(goversion.NewVersion("1.1.0"))))
}

func TestCheckConstraints(t *testing.T) {
	t.Parallel()

	assert.True(t, CheckConstraint("1.0.0", ">= 1.0.0"))
	assert.True(t, CheckConstraint(goversion.Must(goversion.NewVersion("1.0.0")), ">= 0.9.0"))
	assert.True(t, CheckConstraint("1.0.0", ">= 0.9.0, < 1.1.0"))
	assert.True(t, CheckConstraint(goversion.Must(goversion.NewVersion("1.0.0")), ">= 0.9.0, < 1.0.1"))
	assert.False(t, CheckConstraint("1.0.0", "< 1.0.0"))
	assert.True(t, CheckConstraint("1.1.0", "~> 1.1.0"))
	assert.False(t, CheckConstraint("1.2.0", "~> 1.1.0"))
	assert.True(t, CheckConstraint("1.2.0-rc1", "~> 1.2.0"))
	assert.True(t, CheckConstraint("1.2.1-rc1", "~> 1.2.0"))
	assert.False(t, CheckConstraint("1.3.1-rc1", "~> 1.2.0"))
	assert.True(t, CheckConstraint("1.3.1-rc1", "> 1.2.0"))
}

func TestNewSupportedVersion(t *testing.T) {
	t.Parallel()

	meta := &versionpb.MetadataVersion{
		Supported: map[string]string{
			"cli":           ">= 1.0.0",
			"olm":           ">= 0.20.0",
			"catalog":       ">= 1.0.0",
			"kubernetes":    ">= 1.22.0",
			"pgOperator":    ">= 2.4.0",
			"pxcOperator":   ">= 1.10.0",
			"psmdbOperator": ">= 1.15.0",
		},
	}

	supVer, err := NewSupportedVersion(meta)
	require.NoError(t, err)

	assert.True(t, supVer.Cli.Check(goversion.Must(goversion.NewVersion("1.0.0"))))
	assert.True(t, supVer.Olm.Check(goversion.Must(goversion.NewVersion("0.20.0"))))
	assert.True(t, supVer.Catalog.Check(goversion.Must(goversion.NewVersion("1.0.0"))))
	assert.True(t, supVer.Kubernetes.Check(goversion.Must(goversion.NewVersion("1.22.0"))))
	assert.True(t, supVer.PGOperator.Check(goversion.Must(goversion.NewVersion("2.4.0"))))
	assert.True(t, supVer.PXCOperator.Check(goversion.Must(goversion.NewVersion("1.10.0"))))
	assert.True(t, supVer.PSMDBOperator.Check(goversion.Must(goversion.NewVersion("1.15.0"))))
}
