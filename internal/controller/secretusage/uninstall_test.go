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

package secretusage

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/openeverest/openeverest/v2/pkg/inuse/uninstall"
)

// TestUninstallCheckCoversSecretFinalizer keeps the uninstall check in step
// with the finalizer the Reconciler holds: one it does not look for lets
// core be uninstalled while Secrets still carry it.
func TestUninstallCheckCoversSecretFinalizer(t *testing.T) {
	t.Parallel()

	kind := corev1.SchemeGroupVersion.WithKind("Secret")
	registry := uninstall.Registry()
	i := slices.IndexFunc(registry, func(p uninstall.Protected) bool { return p.Kind == kind })
	require.NotEqual(t, -1, i, "uninstall.Registry has no Secret entry")
	assert.Equal(t, finalizer, registry[i].Finalizer)
}
