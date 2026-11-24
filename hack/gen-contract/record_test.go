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

	msg, err = execute([]string{
		"-root", root, "release", "-version", "2.0.0-alpha.3", "-type", "upgradable",
		"-min", "2.0.0-alpha.2", "-upgradable-from", " 2.0.0-alpha.2 ",
	}, io.Discard, io.Discard)
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
// installed fresh and of a 2.0.1 patch release that can be upgraded to
// from it, with the window [2.0.0, 2.0.1].
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
			name: "fresh install of an unchanged contract",
			fp:   fpA,
			req:  releaseRequest{version: "2.0.2", typ: typeFreshInstall},
			want: newRecord("2.0.2", "2.0.2", "2.0.2", typeFreshInstall, []string{}, fpA),
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
			name: "upgradable from an older release only",
			fp:   fpA,
			req:  releaseRequest{version: "2.0.2", typ: typeUpgradable, minVersion: "2.0.0", upgradableFrom: []string{"2.0.0"}},
			want: newRecord("2.0.2", "2.0.2", "2.0.0", typeUpgradable, []string{"2.0.0"}, fpA),
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
		name     string
		req      releaseRequest
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

			_, err := newRelease(testReleases(t), digests(fpA), tt.req)
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

func TestWriteNew(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "2.1.0.yaml")
	require.NoError(t, writeNew(path, []byte("a\n")))
	require.ErrorIs(t, writeNew(path, []byte("b\n")), os.ErrExist)
	assert.Equal(t, "a\n", readFile(t, filepath.Dir(path), filepath.Base(path)))
}
