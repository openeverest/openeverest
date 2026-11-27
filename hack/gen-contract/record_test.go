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
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/contract"
)

// TestReleaseCommand records a fresh-install release, then an upgradable
// one, and checks the window make gen-contract then publishes.
func TestReleaseCommand(t *testing.T) {
	t.Parallel()

	files := testRepo()
	fps := testFingerprints(t, files)
	root := writeRepo(t, files)

	msg, err := execute([]string{"-root", root, "release", "-version", "2.0.0-alpha.2", "-type", "fresh-install"}, io.Discard, io.Discard)
	require.NoError(t, err)
	wantMsg := "recorded release 2.0.0-alpha.2 in contract/releases/2.0.0-alpha.2.yaml: fresh-install, " +
		"serving contract versions [2.0.0-alpha.2, 2.0.0-alpha.2]; run make gen-contract to publish it"
	assert.Equal(t, wantMsg, msg)
	wantRecord := record("2.0.0-alpha.2", "2.0.0-alpha.2", "2.0.0-alpha.2", "fresh-install", "[]", fps)
	assertRecord(t, wantRecord, readFile(t, root, releasesDir+"/2.0.0-alpha.2.yaml"))

	require.NoError(t, run(root))
	assert.Equal(t, "2.0.0-alpha.2\n", readFile(t, root, versionFile))
	assert.Equal(t, "2.0.0-alpha.2\n", readFile(t, root, minVersionFile))
	assert.Contains(t, readFile(t, root, crdPath), "    openeverest.io/contract-version: 2.0.0-alpha.2\n"+
		"    openeverest.io/min-contract-version: 2.0.0-alpha.2\n"+
		"    openeverest.io/upgradable-from: \"\"\n")
	_, err = execute([]string{"-root", root, "-check-release", "-version", "2.0.0-alpha.2"}, io.Discard, io.Discard)
	require.NoError(t, err, "the release branch holds the contract it just recorded")

	args := []string{
		"-root", root, "release", "-version", "2.0.0-alpha.3", "-type", "upgradable",
		"-min", "2.0.0-alpha.2", "-upgradable-from", " 2.0.0-alpha.2 ",
	}
	msg, err = execute(args, io.Discard, io.Discard)
	require.NoError(t, err)
	wantMsg = "recorded release 2.0.0-alpha.3 in contract/releases/2.0.0-alpha.3.yaml: upgradable, " +
		"serving contract versions [2.0.0-alpha.2, 2.0.0-alpha.3], upgradable from 2.0.0-alpha.2; run make gen-contract to publish it"
	assert.Equal(t, wantMsg, msg)
	wantRecord = record("2.0.0-alpha.3", "2.0.0-alpha.3", "2.0.0-alpha.2", "upgradable", "[2.0.0-alpha.2]", fps)
	assertRecord(t, wantRecord, readFile(t, root, releasesDir+"/2.0.0-alpha.3.yaml"))

	require.NoError(t, run(root))
	assert.Equal(t, "2.0.0-alpha.3\n", readFile(t, root, versionFile))
	assert.Equal(t, "2.0.0-alpha.2\n", readFile(t, root, minVersionFile))
	assert.Contains(t, readFile(t, root, crdPath), "    openeverest.io/contract-version: 2.0.0-alpha.3\n"+
		"    openeverest.io/min-contract-version: 2.0.0-alpha.2\n"+
		"    openeverest.io/upgradable-from: 2.0.0-alpha.2\n")
	assert.Equal(t, contract.ReasonSupported, contract.Check("2.0.0-alpha.2", "2.0.0-alpha.2", "2.0.0-alpha.3").Reason,
		"extensions built for the release it is upgraded from keep working",
	)

	_, err = execute([]string{"-root", root, "release", "-version", "2.0.0-alpha.3", "-type", "fresh-install"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "release 2.0.0-alpha.3 is recorded already, in contract/releases/2.0.0-alpha.3.yaml")
}

// assertRecord checks that a record file reads as the record want.
func assertRecord(t *testing.T, want, got string) {
	t.Helper()

	w, err := decodeRecord([]byte(want))
	require.NoError(t, err)
	g, err := decodeRecord([]byte(got))
	require.NoError(t, err)
	assert.Equal(t, w, g)
}

// testReleases are the records of a 2.0.0 release that can only be
// installed fresh and of a 2.0.1 release that can be upgraded to from it,
// with the window [2.0.0, 2.0.1].
func testReleases(t *testing.T) []release {
	t.Helper()

	releases, err := readReleases(writeReleases(t, map[string]string{
		"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", digests(fpB)),
		"2.0.1.yaml": record("2.0.1", "2.0.1", "2.0.0", "upgradable", "[2.0.0]", digests(fpA)),
	}))
	require.NoError(t, err)
	return releases
}

// newRecord returns the record of a release whose every digest is fp.
func newRecord(version, contractVersion, minVersion, typ string, from []string, fp string) release {
	return release{
		APIVersion: recordAPIVersion, Kind: recordKind,
		Version: version, ContractVersion: contractVersion, MinContractVersion: minVersion, Type: typ, UpgradableFrom: from,
		Fingerprints: map[string]map[string]string{scheme: digests(fp)},
	}
}

func TestNewRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fp   string
		req  releaseRequest
		want release
	}{
		{
			name: "fresh install",
			fp:   fpB,
			req:  releaseRequest{version: "2.1.0", typ: typeFreshInstall},
			want: newRecord("2.1.0", "2.1.0", "2.1.0", typeFreshInstall, []string{}, fpB),
		},
		{
			name: "minor release of an unchanged contract records itself",
			fp:   fpA,
			req:  releaseRequest{version: "2.1.0", typ: typeFreshInstall},
			want: newRecord("2.1.0", "2.1.0", "2.1.0", typeFreshInstall, []string{}, fpA),
		},
		{
			name: "fresh-install patch release keeps the contract version of its line",
			fp:   fpA,
			req:  releaseRequest{version: "2.0.2", typ: typeFreshInstall},
			want: newRecord("2.0.2", "2.0.1", "2.0.1", typeFreshInstall, []string{}, fpA),
		},
		{
			name: "upgradable from the latest release, serving its extensions only",
			fp:   fpB,
			req:  releaseRequest{version: "2.1.0", typ: typeUpgradable, minVersion: "2.0.1", upgradableFrom: []string{"2.0.1"}},
			want: newRecord("2.1.0", "2.1.0", "2.0.1", typeUpgradable, []string{"2.0.1"}, fpB),
		},
		{
			name: "upgradable from every release, serving all their extensions",
			fp:   fpB,
			req:  releaseRequest{version: "2.1.0", typ: typeUpgradable, minVersion: "2.0.0", upgradableFrom: []string{"2.0.0", "2.0.1"}},
			want: newRecord("2.1.0", "2.1.0", "2.0.0", typeUpgradable, []string{"2.0.0", "2.0.1"}, fpB),
		},
		{
			name: "patch release upgradable from an older release only",
			fp:   fpA,
			req:  releaseRequest{version: "2.0.2", typ: typeUpgradable, minVersion: "2.0.0", upgradableFrom: []string{"2.0.0"}},
			want: newRecord("2.0.2", "2.0.1", "2.0.0", typeUpgradable, []string{"2.0.0"}, fpA),
		},
		{
			name: "minimum below the latest minimum",
			fp:   fpB,
			req:  releaseRequest{version: "2.1.0", typ: typeUpgradable, minVersion: "2.0.0-rc.1", upgradableFrom: []string{"2.0.1"}},
			want: newRecord("2.1.0", "2.1.0", "2.0.0-rc.1", typeUpgradable, []string{"2.0.1"}, fpB),
		},
		{
			name: "pre-release",
			fp:   fpB,
			req:  releaseRequest{version: "2.1.0-rc.1", typ: typeUpgradable, minVersion: "2.0.0", upgradableFrom: []string{"2.0.1"}},
			want: newRecord("2.1.0-rc.1", "2.1.0-rc.1", "2.0.0", typeUpgradable, []string{"2.0.1"}, fpB),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := newRelease(testReleases(t), digests(tt.fp), tt.req)
			require.NoError(t, err)
			assert.Equal(t, tt.want, r)
		})
	}
}

func TestNewReleaseErrors(t *testing.T) {
	t.Parallel()

	upgradable := func(minVersion string, from ...string) releaseRequest {
		return releaseRequest{version: "2.1.0", typ: typeUpgradable, minVersion: minVersion, upgradableFrom: from}
	}
	tests := []struct {
		name string
		req  releaseRequest
		// fp is every digest of the contract; fpA, those of the latest
		// release, by default.
		fp       string
		wantErrs []string
	}{
		{
			name:     "v prefix",
			req:      releaseRequest{version: "v2.1.0", typ: typeFreshInstall},
			wantErrs: []string{`-version: "v2.1.0" is not a SemVer 2.0.0 version`},
		},
		{
			name:     "not three components",
			req:      releaseRequest{version: "2.1", typ: typeFreshInstall},
			wantErrs: []string{`-version: "2.1" is not a SemVer 2.0.0 version`},
		},
		{
			name:     "no version",
			req:      releaseRequest{typ: typeFreshInstall},
			wantErrs: []string{`-version: "" is not a SemVer 2.0.0 version`},
		},
		{
			name:     "build metadata",
			req:      releaseRequest{version: "2.1.0+abc", typ: typeFreshInstall},
			wantErrs: []string{`-version: "2.1.0+abc" carries build metadata`},
		},
		{
			name:     "development value after a release",
			req:      releaseRequest{version: "2.1.0-alpha.0", typ: typeFreshInstall},
			wantErrs: []string{`-version: "2.1.0-alpha.0" is a development value`},
		},
		{
			name:     "development value after a pre-release",
			req:      releaseRequest{version: "2.1.0-rc.1.next", typ: typeFreshInstall},
			wantErrs: []string{`-version: "2.1.0-rc.1.next" is a development value`},
		},
		{
			name:     "latest release again",
			req:      releaseRequest{version: "2.0.1", typ: typeFreshInstall},
			wantErrs: []string{"release 2.0.1 is recorded already, in contract/releases/2.0.1.yaml"},
		},
		{
			name:     "earlier release again",
			req:      releaseRequest{version: "2.0.0", typ: typeFreshInstall},
			wantErrs: []string{"release 2.0.0 is recorded already"},
		},
		{
			name:     "older than the latest release",
			req:      releaseRequest{version: "2.0.1-rc.1", typ: typeFreshInstall},
			wantErrs: []string{"release 2.0.1-rc.1 is not newer than 2.0.1, the latest release recorded here"},
		},
		{
			name: "patch release that changes the contract",
			req:  releaseRequest{version: "2.0.2", typ: typeFreshInstall},
			fp:   fpB,
			wantErrs: []string{
				"release 2.0.2 is a patch release of 2.0.1, so it must keep its contract, " +
					"but the contract changed since release 2.0.1: crds, http-api, go-api, changes differ",
			},
		},
		{
			name:     "no type",
			req:      releaseRequest{version: "2.1.0"},
			wantErrs: []string{"-type is required: fresh-install or upgradable"},
		},
		{
			name:     "unknown type",
			req:      releaseRequest{version: "2.1.0", typ: "in-place"},
			wantErrs: []string{`-type "in-place" is neither fresh-install nor upgradable`},
		},
		{
			name:     "fresh install with a minimum",
			req:      releaseRequest{version: "2.1.0", typ: typeFreshInstall, minVersion: "2.0.0"},
			wantErrs: []string{"-min and -upgradable-from are for an upgradable release only"},
		},
		{
			name:     "fresh install upgradable from",
			req:      releaseRequest{version: "2.1.0", typ: typeFreshInstall, upgradableFrom: []string{"2.0.1"}},
			wantErrs: []string{"-min and -upgradable-from are for an upgradable release only"},
		},
		{
			name: "upgradable without a minimum or releases",
			req:  upgradable(""),
			wantErrs: []string{
				"-min is required for an upgradable release",
				"-upgradable-from is required for an upgradable release",
			},
		},
		{
			name:     "invalid minimum",
			req:      upgradable("2.0", "2.0.1"),
			wantErrs: []string{`-min: "2.0" is not a SemVer 2.0.0 version`},
		},
		{
			name:     "development minimum",
			req:      upgradable("2.0.1-rc.1.next", "2.0.1"),
			wantErrs: []string{`-min: "2.0.1-rc.1.next" is a development value`},
		},
		{
			name:     "upgradable from an invalid version",
			req:      upgradable("2.0.1", "v2.0.1"),
			wantErrs: []string{`-upgradable-from: "v2.0.1" is not a SemVer 2.0.0 version`},
		},
		{
			name:     "upgradable from a release without a record",
			req:      upgradable("2.0.1", "2.0.1", "2.0.2"),
			wantErrs: []string{"-upgradable-from: release 2.0.2 has no record in contract/releases"},
		},
		{
			name:     "upgradable from itself",
			req:      upgradable("2.0.1", "2.1.0"),
			wantErrs: []string{"-upgradable-from: release 2.1.0 has no record"},
		},
		{
			name:     "upgradable from a release twice",
			req:      upgradable("2.0.1", "2.0.1", "2.0.1"),
			wantErrs: []string{"upgradableFrom: 2.0.1 is listed twice"},
		},
		{
			name: "minimum above the latest contract version",
			req:  upgradable("2.0.2", "2.0.1"),
			wantErrs: []string{
				"-min 2.0.2 is above 2.0.1, the contract version of release 2.0.1: " +
					"extensions built for 2.0.1 would stop working as soon as core is upgraded",
			},
		},
		{
			name:     "minimum above the contract version of a release it is upgraded from",
			req:      upgradable("2.0.1", "2.0.0", "2.0.1"),
			wantErrs: []string{"-min 2.0.1 is above 2.0.0, the contract version of release 2.0.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.fp == "" {
				tt.fp = fpA
			}
			_, err := newRelease(testReleases(t), digests(tt.fp), tt.req)
			require.Error(t, err)
			for _, want := range tt.wantErrs {
				assert.ErrorContains(t, err, want)
			}
		})
	}
}

func TestReleaseCommandErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "unknown command", args: []string{"publish"}, wantErr: `unknown command "publish"`},
		{name: "check and record", args: []string{"-check-release", "release", "-version", "2.0.0-alpha.2"}, wantErr: "-check-release, -version and -branch do not go with the release command"},
		{name: "unknown flag", args: []string{"release", "-version", "2.0.0-alpha.2", "-force"}, wantErr: "flag provided but not defined: -force"},
		{name: "extra argument", args: []string{"release", "-version", "2.0.0-alpha.2", "-type", "fresh-install", "now"}, wantErr: `release: unexpected argument "now"`},
		{
			name:    "empty release in the list",
			args:    []string{"release", "-version", "2.0.0-alpha.2", "-type", "upgradable", "-min", "2.0.0-alpha.1", "-upgradable-from", "2.0.0-alpha.1,"},
			wantErr: `-upgradable-from "2.0.0-alpha.1," has an empty entry`,
		},
		{name: "invalid request", args: []string{"release", "-version", "2.0.0-alpha.2"}, wantErr: "-type is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := writeRepo(t, testRepo())
			_, err := execute(append([]string{"-root", root}, tt.args...), io.Discard, io.Discard)
			require.ErrorContains(t, err, tt.wantErr)
			entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(releasesDir)))
			require.NoError(t, err)
			assert.Len(t, entries, 1, "a refused release leaves no record behind")
		})
	}
}

func TestExecuteHelpAndLoadErrors(t *testing.T) {
	t.Parallel()

	_, err := execute([]string{"-h"}, io.Discard, io.Discard)
	require.ErrorIs(t, err, flag.ErrHelp)

	_, err = execute([]string{"-root", t.TempDir(), "release", "-version", "2.0.0-alpha.2", "-type", "fresh-install"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "holds no release record")
}

// TestMarshalRelease checks that a record reads back as the release it was
// written from.
func TestMarshalRelease(t *testing.T) {
	t.Parallel()

	for _, r := range testReleases(t) {
		got, err := decodeRecord(marshalRelease(r))
		require.NoError(t, err)
		assert.Equal(t, r, got)
	}
	want := "# OpenEverest release record: data for hack/gen-contract, not a Kubernetes resource.\n" +
		"apiVersion: contract.openeverest.io/v1\n" +
		"kind: ReleaseRecord\n" +
		"version: 2.0.1                 # computed\n" +
		"contractVersion: 2.0.1         # computed\n" +
		"minContractVersion: 2.0.0      # decided\n" +
		"type: upgradable               # decided\n" +
		"upgradableFrom: [2.0.0]        # decided\n" +
		"fingerprints:                  # computed; add-only; key = fingerprint scheme\n" +
		"  " + scheme + ":\n"
	for _, name := range inputNames() {
		want += "    " + name + ": " + fpA + "\n"
	}
	assert.Equal(t, want, string(marshalRelease(testReleases(t)[1])))
}

// TestReleaseFromTags checks that -tag-repo reads the records of the
// releases an upgradable release is upgraded from out of their tags, as
// the release workflow does: a release cut on another branch has no record
// here until a pull request brings it.
func TestReleaseFromTags(t *testing.T) {
	t.Parallel()

	files := testRepo()
	files[recordPath("2.0.0")] = record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", testFingerprints(t, files))
	root := gitRepo(t, files, "v2.0.0")
	// 2.0.1 was cut from another branch: its tag carries its record, this
	// branch does not.
	writeFile(t, root, recordPath("2.0.1"), record("2.0.1", "2.0.0", "2.0.0", "upgradable", "[2.0.0]", testFingerprints(t, files)))
	commit(t, root, "release 2.0.1", "v2.0.1")
	runGit(t, root, "rm", "-q", recordPath("2.0.1"))
	commit(t, root, "this branch")

	args := []string{"-root", root, "release", "-version", "2.1.0", "-type", "upgradable", "-min", "2.0.0", "-upgradable-from", "2.0.0,2.0.1"}
	_, err := execute(args, io.Discard, io.Discard)
	require.ErrorContains(t, err, "release 2.0.1 has no record in contract/releases; read the records of releases cut on another branch from their tags with -tag-repo")

	msg, err := execute(append(args, "-tag-repo", root), io.Discard, io.Discard)
	require.NoError(t, err)
	assert.Contains(t, msg, "upgradable from 2.0.0, 2.0.1")

	// The minimum must serve the extensions of 2.0.1, whose contract
	// version its tag says.
	files = testRepo()
	files[recordPath("2.0.0")] = record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", testFingerprints(t, files))
	root = gitRepo(t, files, "v2.0.0")
	args = []string{
		"-root", root, "release", "-version", "2.1.0", "-type", "upgradable", "-min", "2.0.0",
		"-upgradable-from", "2.0.0,2.0.1", "-tag-repo", root,
	}
	_, err = execute(args, io.Discard, io.Discard)
	require.ErrorContains(t, err, "-upgradable-from: tag v2.0.1 is not here; fetch it")

	runGit(t, root, "tag", "v2.0.1")
	args = []string{
		"-root", root, "release", "-version", "2.1.0", "-type", "upgradable", "-min", "2.0.0",
		"-upgradable-from", "2.0.1", "-tag-repo", root,
	}
	_, err = execute(args, io.Discard, io.Discard)
	require.ErrorContains(t, err, "-upgradable-from: v2.0.1 carries no record of release 2.0.1")

	// The seed, which no tag carries, comes from the records here.
	args = []string{
		"-root", root, "release", "-version", "2.1.0", "-type", "upgradable", "-min", seedVersion,
		"-upgradable-from", seedVersion, "-tag-repo", root,
	}
	_, err = execute(args, io.Discard, io.Discard)
	require.NoError(t, err)
}

// TestCheckReleaseOfTheLatestRelease checks -check-release -version for the
// release recorded last, which a re-run of the release workflow tags: the
// tree must be the one it was recorded from, whatever release it is.
func TestCheckReleaseOfTheLatestRelease(t *testing.T) {
	t.Parallel()

	files := testRepo()
	files[recordPath("2.1.0-rc.1")] = record("2.1.0-rc.1", "2.1.0-rc.1", "2.1.0-rc.1", "fresh-install", "[]", testFingerprints(t, files))
	root := writeRepo(t, files)
	msg, err := execute([]string{"-root", root, "-check-release", "-version", "2.1.0-rc.1"}, io.Discard, io.Discard)
	require.NoError(t, err)
	assert.Contains(t, msg, "the contract is the one release 2.1.0-rc.1 was cut with")

	writeFile(t, root, changesFile, testChanges+"- #9: a change.\n")
	_, err = execute([]string{"-root", root, "-check-release", "-version", "2.1.0-rc.1"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "changes differs from its record (fingerprint scheme "+scheme+"), so this tree is not the one release 2.1.0-rc.1 was recorded from")

	msg, err = execute([]string{"-root", root, "-check-release", "-version", "2.1.0-rc.2"}, io.Discard, io.Discard)
	require.NoError(t, err, "a later pre-release may change the contract")
	assert.Contains(t, msg, "is not a patch release of 2.1.0-rc.1")
}

func TestWriteNew(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "2.1.0.yaml")
	require.NoError(t, writeNew(path, []byte("a\n")))
	require.ErrorIs(t, writeNew(path, []byte("b\n")), os.ErrExist)
	assert.Equal(t, "a\n", readFile(t, filepath.Dir(path), filepath.Base(path)))
}
