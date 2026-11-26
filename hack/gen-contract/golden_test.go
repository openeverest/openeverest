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
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

//nolint:gochecknoglobals // a test flag
var updateGolden = flag.Bool("update-golden", false, "write the golden files of a new fingerprint scheme")

// goldenDir holds a frozen tree, with an artifact of every input, and for
// each scheme the canonical stream of every input of that tree and its
// digests.
const goldenDir = "testdata/golden"

// TestGolden pins the canonical streams of the current scheme. Records
// hold digests computed by every generator that ever ran under a scheme,
// so a change to what an input covers, strips or how it is written down
// would make digests of an unchanged contract differ, and the contract
// version move for no reason, or differ from the record of a release
// branch whose generator did not change. Such a change needs a new scheme:
// name it in scheme, then run
//
//	go test ./hack/gen-contract -run TestGolden -update-golden
//
// which writes the golden files of the new scheme and refuses to rewrite
// those of an existing one. Delete the directory of the old scheme.
func TestGolden(t *testing.T) {
	t.Parallel()

	tree := filepath.Join(goldenDir, "tree")
	dir := filepath.Join(goldenDir, scheme)
	got, err := streams(tree)
	require.NoError(t, err)
	require.Len(t, got, len(inputs), "the golden tree holds an artifact of every input")
	fps, err := fingerprints(tree)
	require.NoError(t, err)

	if *updateGolden {
		_, err := os.Stat(dir)
		require.ErrorIs(t, err, os.ErrNotExist, "%s exists: the golden files of a scheme never change; name a new scheme", dir)
		require.NoError(t, os.MkdirAll(dir, 0o750))
		for name, data := range got {
			require.NoError(t, os.WriteFile(filepath.Join(dir, name+".stream"), data, 0o600))
		}
		data, err := yaml.Marshal(fps)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "digests.yaml"), data, 0o600))
	}

	for _, name := range inputNames() {
		want, err := os.ReadFile(filepath.Join(dir, name+".stream")) //nolint:gosec // a golden file of this test
		require.NoError(t, err, "no golden files for scheme %s: run go test ./hack/gen-contract -run TestGolden -update-golden", scheme)
		assert.Equal(t, string(want), string(got[name]),
			"the canonical stream of %s changed under scheme %s: name a new scheme and write its golden files (see TestGolden)", name, scheme,
		)
	}
	data, err := os.ReadFile(filepath.Join(dir, "digests.yaml")) //nolint:gosec // a golden file of this test
	require.NoError(t, err)
	var want map[string]string
	require.NoError(t, yaml.UnmarshalStrict(data, &want))
	assert.Equal(t, want, fps, "the digests of scheme %s changed: name a new scheme (see TestGolden)", scheme)
}
