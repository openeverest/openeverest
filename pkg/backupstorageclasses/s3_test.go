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

package backupstorageclasses_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openeverest/openeverest/v2/pkg/backupstorageclasses"
)

func TestEffectiveImplementation(t *testing.T) {
	t.Parallel()

	for in, want := range map[backupstorageclasses.S3Implementation]backupstorageclasses.S3Implementation{
		"":       backupstorageclasses.S3ImplementationAWS,
		"aws":    backupstorageclasses.S3ImplementationAWS,
		"minio":  backupstorageclasses.S3ImplementationMinIO,
		"ceph":   backupstorageclasses.S3ImplementationCeph,
		"gcs":    backupstorageclasses.S3ImplementationGCS,
		"other":  backupstorageclasses.S3ImplementationOther,
		"wasabi": backupstorageclasses.S3ImplementationOther,
	} {
		assert.Equal(t, want, backupstorageclasses.S3{Implementation: in}.EffectiveImplementation(), "implementation %q", in)
	}
}
