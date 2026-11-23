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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/contract/definitions"
	"github.com/openeverest/openeverest/v2/contract/referents"
)

// A provider still matching the deprecated sentinels must match the errors
// core and the runtime now return, and the other way around.
func TestDeprecatedSentinelsMatchTheirReplacements(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ deprecated, replacement error }{
		{ErrInvalidReference, referents.ErrInvalidReference},
		{ErrBackupNotFound, referents.ErrBackupNotFound},
		{ErrBackupClassNotFound, referents.ErrBackupClassNotFound},
		{ErrInstanceNotFound, referents.ErrInstanceNotFound},
		{ErrBackupStorageNotFound, referents.ErrBackupStorageNotFound},
		{ErrBackupNotSucceeded, referents.ErrBackupNotSucceeded},
		{ErrProviderUnsupported, referents.ErrProviderUnsupported},
		{ErrRestorePITRUnsupported, referents.ErrRestorePITRUnsupported},
		{ErrRestorePITRDateRequired, referents.ErrRestorePITRDateRequired},
		{ErrBackupImportUnsupported, referents.ErrBackupImportUnsupported},
		{ErrBackupClassIncompatible, referents.ErrBackupClassIncompatible},
		{ErrBackupStorageIncompatible, referents.ErrBackupStorageIncompatible},
	} {
		t.Run(tc.replacement.Error(), func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, fmt.Errorf("wrapped: %w", tc.replacement), tc.deprecated)
			require.ErrorIs(t, fmt.Errorf("wrapped: %w", tc.deprecated), tc.replacement)
			require.ErrorIs(t, fmt.Errorf("wrapped: %w", tc.deprecated), referents.ErrInvalidReference)
			require.ErrorIs(t, fmt.Errorf("wrapped: %w", tc.replacement), ErrInvalidReference)
		})
	}
}

func TestDeprecatedSchemaErrorsMatchTheirReplacements(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, fmt.Errorf("wrapped: %w", definitions.ErrSecretSchemaValidation), ErrSecretSchemaValidation)
	require.ErrorIs(t, fmt.Errorf("wrapped: %w", definitions.ErrConfigMapSchemaValidation), ErrConfigMapSchemaValidation)
}

func TestDeprecatedStatusFieldOwner(t *testing.T) {
	t.Parallel()
	assert.Equal(t, contract.ProviderStatusFieldOwner("psmdb"), StatusFieldOwner("psmdb"))
}
