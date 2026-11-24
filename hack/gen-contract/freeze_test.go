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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFileChanges(t *testing.T) {
	t.Parallel()

	got, err := parseFileChanges(strings.NewReader(
		"A\tcontract/releases/2.1.0.yaml\n" +
			"R100\tcontract/releases/a.yaml\tcontract/releases/b.yaml\n" +
			"modified\tcontract/releases/2.0.0.yaml\n" +
			"renamed\tcontract/releases/c.yaml\tcontract/releases/d.yaml\n" +
			"unchanged\tREADME.md\n\n" +
			"removed\tcontract/releases/2.0.1.yaml\n",
	))
	require.NoError(t, err)
	assert.Equal(t, []fileChange{
		{status: 'A', path: "contract/releases/2.1.0.yaml"},
		{status: 'R', from: "contract/releases/a.yaml", path: "contract/releases/b.yaml"},
		{status: 'M', path: "contract/releases/2.0.0.yaml"},
		{status: 'R', from: "contract/releases/c.yaml", path: "contract/releases/d.yaml"},
		{status: 'D', path: "contract/releases/2.0.1.yaml"},
	}, got,
	)

	for _, bad := range []string{"X\ta\n", "A\n", "R\ta\n", "M\ta\tb\n", "Ax\ta\n"} {
		_, err := parseFileChanges(strings.NewReader(bad))
		require.Error(t, err, bad)
	}
}

// checkRecords runs check-records on the repository at root with changes,
// as git diff --name-status lists them.
func checkRecords(t *testing.T, root string, changes ...string) (string, string, error) {
	t.Helper()

	file := filepath.Join(t.TempDir(), "changes.txt")
	require.NoError(t, os.WriteFile(file, []byte(strings.Join(changes, "\n")+"\n"), 0o600))
	var notices bytes.Buffer
	msg, err := execute([]string{"-root", root, "check-records", "-changes", file}, io.Discard, &notices)
	return msg, notices.String(), err
}

// TestCheckRecords checks that release records are frozen: a record is
// added only as the tag of its release carries it, and changes only by
// fingerprints of a new scheme computed from that tag.
func TestCheckRecords(t *testing.T) {
	t.Parallel()

	fps := testFingerprints(t, testRepo())
	tagged := olderRecord()
	rebaselined := tagged + "  " + scheme + ":\n"
	for _, name := range inputNames() {
		rebaselined += "    " + name + ": " + fps[name] + "\n"
	}
	added := "A\t" + recordPath("2.1.0")
	modified := "M\t" + recordPath("2.1.0")

	tests := []struct {
		name string
		// record is the record of 2.1.0 in the working tree; the tag
		// carries tagged.
		record  string
		changes []string
		wantErr string
	}{
		{name: "added as the tag carries it", record: tagged, changes: []string{added}},
		{name: "added with a new scheme", record: rebaselined, changes: []string{added}},
		{name: "rebaselined", record: rebaselined, changes: []string{modified}},
		{
			name:    "new scheme with other digests",
			record:  strings.Replace(rebaselined, "crds: "+fps["crds"], "crds: "+fpB, 1),
			changes: []string{modified},
			wantErr: "fingerprints." + scheme + " is not what rebaseline computes from v2.1.0, for crds",
		},
		{
			name:    "a scheme this generator does not compute",
			record:  tagged + "  c98:\n    crds: " + fpA + "\n",
			changes: []string{modified},
			wantErr: "fingerprints.c98 was added, but this generator computes only scheme " + scheme,
		},
		{
			name:    "a digest edited",
			record:  strings.Replace(tagged, "crds: "+fpA, "crds: "+fpB, 1),
			changes: []string{modified},
			wantErr: "fingerprints." + otherScheme + " differs from the one v2.1.0 carries; digests are never changed or removed",
		},
		{
			name:    "a scheme removed",
			record:  recordWith("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]", map[string]map[string]string{scheme: fps}),
			changes: []string{modified},
			wantErr: "fingerprints." + otherScheme + " differs from the one v2.1.0 carries",
		},
		{
			name:    "a decided field edited",
			record:  strings.Replace(strings.Replace(tagged, "minContractVersion: 2.0.0", "minContractVersion: 2.1.0", 1), "type: upgradable\nupgradableFrom: [2.0.0]", "type: fresh-install\nupgradableFrom: []", 1),
			changes: []string{modified},
			wantErr: "minContractVersion, type, upgradableFrom differ from the record v2.1.0 carries",
		},
		{name: "comments edited", record: "# Comments are not fields.\n" + tagged, changes: []string{modified}},
		{name: "deleted", record: tagged, changes: []string{"D\t" + recordPath("2.0.0")}, wantErr: "contract/releases/2.0.0.yaml: deleted; a release record is never deleted"},
		{
			name:    "renamed",
			record:  tagged,
			changes: []string{"R098\t" + recordPath("2.0.0") + "\t" + recordPath("2.1.0")},
			wantErr: "contract/releases/2.0.0.yaml: renamed to contract/releases/2.1.0.yaml; a release record is never renamed",
		},
		{name: "type changed", record: tagged, changes: []string{"T\t" + recordPath("2.1.0")}, wantErr: "its type changed"},
		{name: "not a record", record: tagged, changes: []string{"A\tcontract/releases/README.md"}, wantErr: "contract/releases holds only release records"},
		{name: "outside the records", record: tagged, changes: []string{"M\tcontract/VERSION", "D\tcontract/CHANGES"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := releaseRepo(t, tagged)
			writeFile(t, root, recordPath("2.1.0"), tt.record)
			msg, _, err := checkRecords(t, root, tt.changes...)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, "release records are frozen")
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, msg)
		})
	}
}

func TestCheckRecordsTags(t *testing.T) {
	t.Parallel()

	// A record added without its tag fails, whatever it holds.
	files := testRepo()
	files[recordPath("2.1.0")] = olderRecord()
	root := gitRepo(t, files)
	_, _, err := checkRecords(t, root, "A\t"+recordPath("2.1.0"))
	require.ErrorContains(t, err, "tag v2.1.0 is not here; fetch it")
	require.ErrorContains(t, err, "a record is added only from the tag of its release, so a record without one fails")

	// A tag that carries no record.
	delete(files, recordPath("2.1.0"))
	root = gitRepo(t, files, "v2.1.0")
	writeFile(t, root, recordPath("2.1.0"), olderRecord())
	_, _, err = checkRecords(t, root, "A\t"+recordPath("2.1.0"))
	require.ErrorContains(t, err, "v2.1.0 carries no record of its release, so this one did not come from it")

	// A record that does not read.
	root = releaseRepo(t, olderRecord())
	writeFile(t, root, recordPath("2.1.0"), "kind: ReleaseRecord\n")
	_, _, err = checkRecords(t, root, "M\t"+recordPath("2.1.0"))
	require.ErrorContains(t, err, `apiVersion "" is not contract.openeverest.io/v1`)
}

// TestCheckRecordsSeed checks the seed, which no tag carries: it may be
// added, and never changes after.
func TestCheckRecordsSeed(t *testing.T) {
	t.Parallel()

	root := gitRepo(t, testRepo())
	seed := recordPath(seedVersion)
	_, notices, err := checkRecords(t, root, "A\t"+seed)
	require.NoError(t, err)
	assert.Contains(t, notices, "no tag carries the record of the seed release, so only later changes to it are checked")

	_, _, err = checkRecords(t, root, "M\t"+seed)
	require.ErrorContains(t, err, "contract/releases/2.0.0-alpha.1.yaml: modified; the record of the seed release is never modified once merged")
	_, _, err = checkRecords(t, root, "D\t"+seed)
	require.ErrorContains(t, err, "deleted")
}

// TestCheckAllRecords checks -all, which checks every record as it is, for
// a branch whose changes are unknown.
func TestCheckAllRecords(t *testing.T) {
	t.Parallel()

	root := releaseRepo(t, olderRecord())
	msg, err := execute([]string{"-root", root, "check-records", "-all"}, io.Discard, io.Discard)
	require.NoError(t, err)
	assert.Equal(t, "2 release record changes checked against the tags of their releases", msg)

	writeFile(t, root, recordPath("2.1.0"), strings.Replace(olderRecord(), "type: upgradable", "type: upgradable # still", 1)+"# Comments only.\n")
	_, err = execute([]string{"-root", root, "check-records", "-all"}, io.Discard, io.Discard)
	require.NoError(t, err)

	writeFile(t, root, recordPath("2.1.0"), strings.Replace(olderRecord(), "minContractVersion: 2.0.0", "minContractVersion: 2.0.1", 1))
	_, err = execute([]string{"-root", root, "check-records", "-all"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "minContractVersion differs from the record v2.1.0 carries")

	for _, args := range [][]string{{}, {"-all", "-changes", "x"}, {"-all", "x"}} {
		_, err = execute(append([]string{"-root", root, "check-records"}, args...), io.Discard, io.Discard)
		require.Error(t, err, args)
	}
}
