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
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/contract"
)

const (
	fpA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fpB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// digests returns the digests of every input, each d.
func digests(d string) map[string]string {
	out := make(map[string]string, len(inputs))
	for _, name := range inputNames() {
		out[name] = d
	}
	return out
}

// with returns digests with name set to d, or removed for "".
func with(digests map[string]string, name, d string) map[string]string {
	out := maps.Clone(digests)
	if d == "" {
		delete(out, name)
	} else {
		out[name] = d
	}
	return out
}

// record returns the content of a release record holding the digests fps
// under the current scheme, or no fingerprints for nil.
func record(version, contractVersion, minVersion, typ, upgradableFrom string, fps map[string]string) string {
	return recordWith(version, contractVersion, minVersion, typ, upgradableFrom, map[string]map[string]string{scheme: fps})
}

// recordWith returns the content of a release record with fingerprints by
// scheme; none when every scheme is nil.
func recordWith(version, contractVersion, minVersion, typ, upgradableFrom string, fps map[string]map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: %s\nkind: %s\nversion: %s\ncontractVersion: %s\nminContractVersion: %s\ntype: %s\nupgradableFrom: %s\n",
		recordAPIVersion, recordKind, version, contractVersion, minVersion, typ, upgradableFrom,
	)
	opened := false
	for _, s := range slices.Sorted(maps.Keys(fps)) {
		if fps[s] == nil {
			continue
		}
		if !opened {
			b.WriteString("fingerprints:\n")
			opened = true
		}
		fmt.Fprintf(&b, "  %s:\n", s)
		for _, name := range sortedInputs(fps[s]) {
			fmt.Fprintf(&b, "    %s: %s\n", name, fps[s][name])
		}
	}
	return b.String()
}

// writeReleases writes release records, by file name, to a new directory.
func writeReleases(t *testing.T, records map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range records {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	return dir
}

func TestDevelopmentAfter(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"2.0.0-alpha.1":   "2.0.0-alpha.1.next",
		"2.0.0-alpha.1.1": "2.0.0-alpha.1.1.next",
		"2.0.0-alpha.10":  "2.0.0-alpha.10.next",
		"2.0.0-rc.1":      "2.0.0-rc.1.next",
		"2.0.0":           "2.1.0-alpha.0",
		"2.1.3":           "2.2.0-alpha.0",
		"2.9.0":           "2.10.0-alpha.0",
	}
	for release, want := range tests {
		t.Run(release, func(t *testing.T) {
			t.Parallel()

			got := developmentAfter(release)
			assert.Equal(t, want, got)
			assert.True(t, contract.IsDevelopment(got))
			assert.Positive(t, contract.Compare(got, release), "a development value sorts after its release")
		})
	}
}

// TestDevelopmentValuesSortBeforeTheNextRelease checks that the development
// value after a release sorts before every release main can build next.
func TestDevelopmentValuesSortBeforeTheNextRelease(t *testing.T) {
	t.Parallel()

	tests := []struct{ release, next string }{
		{"2.0.0-alpha.1", "2.0.0-alpha.2"},
		{"2.0.0-alpha.1", "2.0.0-beta.1"},
		{"2.0.0-alpha.1", "2.0.0-dev.4"},
		{"2.0.0-alpha.1", "2.0.0"},
		{"2.0.0-rc.1", "2.0.0"},
		{"2.0.0", "2.1.0-alpha.1"},
		{"2.0.0", "2.1.0"},
		{"2.0.0", "3.0.0-alpha.1"},
		{"2.0.1", "2.1.0-alpha.1"},
	}
	for _, tt := range tests {
		assert.Negative(t, contract.Compare(developmentAfter(tt.release), tt.next), "%s → %s", tt.release, tt.next)
	}
}

// TestDevelopmentValueAfterAReleaseSortsAboveItsPatches checks that main,
// which builds the next minor release, sorts above every patch release of
// the release it follows, which release branches build.
func TestDevelopmentValueAfterAReleaseSortsAboveItsPatches(t *testing.T) {
	t.Parallel()

	for _, patch := range []string{"2.0.1", "2.0.9", "2.0.10-rc.1"} {
		assert.Positive(t, contract.Compare(developmentAfter("2.0.0"), patch), patch)
	}
	assert.Equal(t, developmentAfter("2.0.0"), developmentAfter("2.0.1"), "a patch release leaves main's value as it was")
}

// TestIsDevelopment checks the rule against the contract package's, which
// this command does not import.
func TestIsDevelopment(t *testing.T) {
	t.Parallel()

	var vectors struct {
		Development []struct {
			Version     string `json:"version"`
			Development bool   `json:"development"`
		} `json:"development"`
	}
	require.NoError(t, json.Unmarshal(contract.OrderVectors, &vectors))
	require.NotEmpty(t, vectors.Development)
	for _, tc := range vectors.Development {
		parsed, err := semver.Parse(tc.Version)
		if err != nil {
			continue
		}
		assert.Equal(t, tc.Development, isDevelopment(parsed), tc.Version)
	}
}

func TestDecide(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		records            map[string]string
		fps                map[string]string
		wantVersion        string
		wantMinVersion     string
		wantUpgradableFrom []string
		wantDiffer         []string
		wantSkipped        []string
	}{
		{
			name:           "seed, exempt by name",
			records:        map[string]string{"2.0.0-alpha.1.yaml": testSeedRelease},
			fps:            digests(fpA),
			wantVersion:    "2.0.0-alpha.1.next",
			wantMinVersion: "2.0.0-alpha.1",
		},
		{
			name: "unchanged since a pre-release",
			records: map[string]string{
				"2.0.0-alpha.1.yaml": testSeedRelease,
				"2.0.0-alpha.2.yaml": record("2.0.0-alpha.2", "2.0.0-alpha.2", "2.0.0-alpha.2", "fresh-install", "[]", digests(fpA)),
			},
			fps:            digests(fpA),
			wantVersion:    "2.0.0-alpha.2",
			wantMinVersion: "2.0.0-alpha.2",
		},
		{
			name: "changed since a pre-release",
			records: map[string]string{
				"2.0.0-alpha.1.yaml": testSeedRelease,
				"2.0.0-alpha.2.yaml": record("2.0.0-alpha.2", "2.0.0-alpha.2", "2.0.0-alpha.1", "upgradable", "[2.0.0-alpha.1]", digests(fpA)),
			},
			fps:            with(with(digests(fpA), "crds", fpB), "changes", fpB),
			wantVersion:    "2.0.0-alpha.2.next",
			wantMinVersion: "2.0.0-alpha.1",
			wantDiffer:     []string{"crds", "changes"},
		},
		{
			name: "an input the record lacks is not compared",
			records: map[string]string{
				"2.0.0-alpha.2.yaml": record("2.0.0-alpha.2", "2.0.0-alpha.2", "2.0.0-alpha.2", "fresh-install", "[]", with(digests(fpA), "go-api", "")),
			},
			fps:            with(digests(fpA), "go-api", fpB),
			wantVersion:    "2.0.0-alpha.2",
			wantMinVersion: "2.0.0-alpha.2",
			wantSkipped:    []string{"go-api"},
		},
		{
			name: "an input the record holds and the tree lacks differs",
			records: map[string]string{
				"2.0.0-alpha.2.yaml": record("2.0.0-alpha.2", "2.0.0-alpha.2", "2.0.0-alpha.2", "fresh-install", "[]", digests(fpA)),
			},
			fps:            with(digests(fpA), "storage-classes", ""),
			wantVersion:    "2.0.0-alpha.2.next",
			wantMinVersion: "2.0.0-alpha.2",
			wantDiffer:     []string{"storage-classes"},
		},
		{
			name: "schemes other than the current one are ignored",
			records: map[string]string{
				"2.0.0-alpha.2.yaml": recordWith("2.0.0-alpha.2", "2.0.0-alpha.2", "2.0.0-alpha.2", "fresh-install", "[]",
					map[string]map[string]string{scheme: digests(fpA), "c99": {"crds": fpB, "ui-schema": fpB}},
				),
			},
			fps:            digests(fpA),
			wantVersion:    "2.0.0-alpha.2",
			wantMinVersion: "2.0.0-alpha.2",
		},
		{
			name: "unchanged since an upgradable release",
			records: map[string]string{
				"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", digests(fpB)),
				"2.1.0.yaml": record("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0, 2.0.1]", digests(fpA)),
			},
			fps:                digests(fpA),
			wantVersion:        "2.1.0",
			wantMinVersion:     "2.0.0",
			wantUpgradableFrom: []string{"2.0.0", "2.0.1"},
		},
		{
			name: "changed since a release",
			records: map[string]string{
				"2.1.0.yaml": record("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]", digests(fpA)),
			},
			fps:            digests(fpB),
			wantVersion:    "2.2.0-alpha.0",
			wantMinVersion: "2.0.0",
			wantDiffer:     inputNames(),
		},
		{
			name: "unchanged since a patch release",
			records: map[string]string{
				"2.1.0.yaml": record("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]", digests(fpA)),
				"2.1.1.yaml": record("2.1.1", "2.1.0", "2.0.0", "upgradable", "[2.1.0]", digests(fpA)),
			},
			fps:                digests(fpA),
			wantVersion:        "2.1.0",
			wantMinVersion:     "2.0.0",
			wantUpgradableFrom: []string{"2.1.0"},
		},
		{
			name: "changed since a patch release",
			records: map[string]string{
				"2.1.0.yaml": record("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]", digests(fpA)),
				"2.1.1.yaml": record("2.1.1", "2.1.0", "2.0.0", "upgradable", "[2.1.0]", digests(fpA)),
			},
			fps:            with(digests(fpA), "http-api", fpB),
			wantVersion:    "2.2.0-alpha.0",
			wantMinVersion: "2.0.0",
			wantDiffer:     []string{"http-api"},
		},
		{
			name: "latest by precedence, not by name",
			records: map[string]string{
				"2.0.0-alpha.9.yaml":  record("2.0.0-alpha.9", "2.0.0-alpha.9", "2.0.0-alpha.9", "fresh-install", "[]", digests(fpB)),
				"2.0.0-alpha.10.yaml": record("2.0.0-alpha.10", "2.0.0-alpha.10", "2.0.0-alpha.10", "fresh-install", "[]", digests(fpA)),
			},
			fps:            digests(fpA),
			wantVersion:    "2.0.0-alpha.10",
			wantMinVersion: "2.0.0-alpha.10",
		},
		{
			name: "the minimum may go down at a later release",
			records: map[string]string{
				"2.1.0.yaml": record("2.1.0", "2.1.0", "2.1.0", "fresh-install", "[]", digests(fpB)),
				"2.2.0.yaml": record("2.2.0", "2.2.0", "2.0.0", "upgradable", "[2.1.0]", digests(fpA)),
			},
			fps:                digests(fpA),
			wantVersion:        "2.2.0",
			wantMinVersion:     "2.0.0",
			wantUpgradableFrom: []string{"2.1.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			releases, err := readReleases(writeReleases(t, tt.records))
			require.NoError(t, err)
			w, c, err := decide(releases, tt.fps)
			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, w.version)
			assert.Equal(t, tt.wantMinVersion, w.minVersion)
			assert.Equal(t, tt.fps, w.fingerprints)
			if tt.wantUpgradableFrom == nil {
				tt.wantUpgradableFrom = []string{}
			}
			assert.Equal(t, tt.wantUpgradableFrom, w.upgradableFrom)
			assert.Equal(t, tt.wantDiffer, c.differ)
			assert.Equal(t, tt.wantSkipped, c.skipped)
			assert.Equal(t, contract.ReasonSupported, contract.Check(w.version, w.minVersion, w.version).Reason,
				"a core must serve its own version",
			)
		})
	}
}

// TestDecideWithoutScheme checks that a latest record without fingerprints
// under the current scheme stops the generator with the commands that add
// them, rather than let it guess either way.
func TestDecideWithoutScheme(t *testing.T) {
	t.Parallel()

	releases, err := readReleases(writeReleases(t, map[string]string{
		"2.0.0-alpha.2.yaml": recordWith("2.0.0-alpha.2", "2.0.0-alpha.2", "2.0.0-alpha.2", "fresh-install", "[]",
			map[string]map[string]string{"c99": digests(fpA)},
		),
	}))
	require.NoError(t, err)
	for _, fps := range []map[string]string{digests(fpA), digests(fpB)} {
		_, _, err = decide(releases, fps)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "the record of release 2.0.0-alpha.2, contract/releases/2.0.0-alpha.2.yaml, has no fingerprints under scheme "+scheme)
		assert.Contains(t, err.Error(), "\tgit fetch upstream --tags")
		assert.Contains(t, err.Error(), "\tgo run ./hack/gen-contract rebaseline -version 2.0.0-alpha.2\n")

		_, _, err = checkRelease(releases, fps, "2.0")
		require.ErrorContains(t, err, "has no fingerprints under scheme "+scheme)
	}
}

func TestComparisonMessages(t *testing.T) {
	t.Parallel()

	r := release{Version: "2.1.0", Fingerprints: map[string]map[string]string{scheme: with(digests(fpA), "go-api", "")}}
	c, err := compareWith(r, with(with(digests(fpA), "go-api", fpB), "crds", ""))
	require.NoError(t, err)
	assert.Equal(t, "the contract changed since release 2.1.0: crds differs from its record (fingerprint scheme "+scheme+"); crds is absent here", c.summary())
	assert.Equal(t, "not compared: go-api, which the record of release 2.1.0 does not hold under fingerprint scheme "+scheme+", since it was absent from its tag", c.skippedNotice())

	c, err = compareWith(r, with(digests(fpA), "go-api", ""))
	require.NoError(t, err)
	assert.Empty(t, c.skippedNotice())
	assert.Equal(t, "the contract is the one release 2.1.0 was cut with (crds, http-api, storage-classes, changes equal under fingerprint scheme "+scheme+")", c.summary())
}

func TestReadReleasesErrors(t *testing.T) {
	t.Parallel()

	valid := record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", digests(fpA))
	tests := []struct {
		name    string
		records map[string]string
		wantErr string
	}{
		{name: "none", records: make(map[string]string), wantErr: "holds no release record"},
		{
			name:    "misnamed",
			records: map[string]string{"2.0.0.yaml": record("2.0.1", "2.0.1", "2.0.1", "fresh-install", "[]", digests(fpA))},
			wantErr: "must be named 2.0.1.yaml",
		},
		{
			name:    "unknown field",
			records: map[string]string{"2.0.0.yaml": valid + "minVersion: 2.0.0\n"},
			wantErr: `unknown field "minVersion"`,
		},
		{
			name:    "the format before records had a header",
			records: map[string]string{"2.0.0.yaml": "version: 2.0.0\ncontractVersion: 2.0.0\nminContractVersion: 2.0.0\ntype: fresh-install\nupgradableFrom: []\nfingerprint: \"\"\n"},
			wantErr: `kind "" is not ReleaseRecord`,
		},
		{
			name:    "unknown apiVersion",
			records: map[string]string{"2.0.0.yaml": strings.Replace(valid, recordAPIVersion, "contract.openeverest.io/v2", 1)},
			wantErr: `apiVersion "contract.openeverest.io/v2" is not contract.openeverest.io/v1, the only format this command reads`,
		},
		{
			name:    "unknown kind",
			records: map[string]string{"2.0.0.yaml": strings.Replace(valid, "kind: ReleaseRecord", "kind: Release", 1)},
			wantErr: `kind "Release" is not ReleaseRecord`,
		},
		{
			name:    "v prefix",
			records: map[string]string{"v2.0.0.yaml": record("v2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", digests(fpA))},
			wantErr: `version: "v2.0.0" is not a SemVer 2.0.0 version`,
		},
		{
			name:    "build metadata",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0+abc", "2.0.0", "fresh-install", "[]", digests(fpA))},
			wantErr: "contractVersion: \"2.0.0+abc\" carries build metadata",
		},
		{
			name:    "development value",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0-alpha.1.next", "fresh-install", "[]", digests(fpA))},
			wantErr: "minContractVersion: \"2.0.0-alpha.1.next\" is a development value",
		},
		{
			name:    "contract after release",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.1", "2.0.0", "fresh-install", "[]", digests(fpA))},
			wantErr: "contractVersion 2.0.1 is above version 2.0.0",
		},
		{
			name:    "minimum above contract",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0-rc.1", "2.0.0", "fresh-install", "[]", digests(fpA))},
			wantErr: "minContractVersion 2.0.0 is above contractVersion 2.0.0-rc.1",
		},
		{
			name:    "unknown type",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "in-place", "[]", digests(fpA))},
			wantErr: `type "in-place" is neither fresh-install nor upgradable`,
		},
		{
			name:    "fresh install upgradable from",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[2.0.0-rc.1]", digests(fpA))},
			wantErr: "upgradableFrom must be empty for a fresh-install release",
		},
		{
			name:    "upgradable from nothing",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "upgradable", "[]", digests(fpA))},
			wantErr: "upgradableFrom must list the releases",
		},
		{
			name:    "upgradable from a later release",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "upgradable", "[2.0.1]", digests(fpA))},
			wantErr: "upgradableFrom: 2.0.1 is not older than 2.0.0",
		},
		{
			name:    "upgradable from twice",
			records: map[string]string{"2.0.1.yaml": record("2.0.1", "2.0.0", "2.0.0", "upgradable", "[2.0.0, 2.0.0]", digests(fpA))},
			wantErr: "upgradableFrom: 2.0.0 is listed twice",
		},
		{
			name:    "no fingerprints",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", nil)},
			wantErr: "fingerprints: missing; only the record of release 2.0.0-alpha.1 may have none. Restore the record from its tag: git show v2.0.0:contract/releases/2.0.0.yaml",
		},
		{
			name:    "empty fingerprints",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", nil) + "fingerprints: {}\n"},
			wantErr: "fingerprints: missing",
		},
		{
			name:    "seed with fingerprints",
			records: map[string]string{"2.0.0-alpha.1.yaml": record("2.0.0-alpha.1", "2.0.0-alpha.1", "2.0.0-alpha.1", "fresh-install", "[]", digests(fpA))},
			wantErr: "fingerprints: release 2.0.0-alpha.1 predates fingerprinting and is exempt by name, so its record has none",
		},
		{
			name:    "malformed scheme",
			records: map[string]string{"2.0.0.yaml": recordWith("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", map[string]map[string]string{"v1": digests(fpA)})},
			wantErr: `fingerprints: "v1" is not a scheme such as c1`,
		},
		{
			name:    "empty scheme",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", make(map[string]string))},
			wantErr: "fingerprints." + scheme + ": holds no digest",
		},
		{
			name:    "blank digest",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", with(digests(fpA), "crds", `""`))},
			wantErr: "fingerprints." + scheme + `.crds: "" is not sha256:<64 lowercase hex digits>`,
		},
		{
			name:    "unknown input",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", with(digests(fpA), "ui-schema", fpA))},
			wantErr: "fingerprints." + scheme + ": ui-schema is not an input of the scheme",
		},
		{
			name:    "no changes",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", with(digests(fpA), "changes", ""))},
			wantErr: "fingerprints." + scheme + ": changes is missing, though no tree lacks it",
		},
		{
			name: "contract version goes back",
			records: map[string]string{
				"2.0.0.yaml": valid,
				"2.0.1.yaml": record("2.0.1", "2.0.0-rc.1", "2.0.0-rc.1", "fresh-install", "[]", digests(fpA)),
			},
			wantErr: "release 2.0.1: its contract version 2.0.0-rc.1 is below 2.0.0, the contract version of the earlier release 2.0.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := readReleases(writeReleases(t, tt.records))
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestIsPatchOf checks which releases must keep the contract of the latest
// one: the patch releases of a final release.
func TestIsPatchOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		latest, v string
		want      bool
	}{
		{"2.1.0", "2.1.1", true},
		{"2.1.0", "2.1", true},
		{"2.1.3", "2.1.4", true},
		{"2.1.0", "2.1.1-rc.1", true},
		{"2.1.0", "2.1.0", true},
		{"2.1.0", "2.2.0", false},
		{"2.1.0", "2.2", false},
		{"2.1.0", "2.2.0-rc.1", false},
		{"2.1.0", "3.1.0", false},
		{"2.1.0", "2.0.5", false},
		{"2.1.0-rc.1", "2.1.0", false},
		{"2.1.0-rc.1", "2.1", false},
		{"2.0.0-alpha.1", "2.0.0-alpha.2", false},
		{"2.0.0-alpha.1", "2.0", false},
		{"2.1.0", "release-2.1", false},
		{"2.1.0", "", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, isPatchOf(tt.latest, tt.v), "isPatchOf(%s, %s)", tt.latest, tt.v)
	}
}

func TestLineOfBranch(t *testing.T) {
	t.Parallel()

	for branch, want := range map[string]string{"release-2.1": "2.1", "release-10.0": "10.0", "main": "", "release-2.1.0": "", "release-02.1": "", "v2": ""} {
		got, ok := lineOfBranch(branch)
		assert.Equal(t, want != "", ok, branch)
		assert.Equal(t, want, got, branch)
	}
}

func TestCheckRelease(t *testing.T) {
	t.Parallel()

	final := []release{{
		Version: "2.1.0", ContractVersion: "2.1.0", MinContractVersion: "2.1.0", Type: typeFreshInstall,
		Fingerprints: map[string]map[string]string{scheme: digests(fpA)},
	}}

	msg, _, err := checkRelease(final, digests(fpA), "2.1")
	require.NoError(t, err)
	assert.Contains(t, msg, "the contract is the one release 2.1.0 was cut with")

	_, c, err := checkRelease(final, with(digests(fpA), "go-api", fpB), "2.1.1")
	require.ErrorContains(t, err, "the contract changed since release 2.1.0: go-api differs from its record")
	require.ErrorContains(t, err, "A patch release must not change what core and extensions share")
	assert.Equal(t, []string{"go-api"}, c.differ)

	msg, _, err = checkRelease(final, digests(fpB), "2.2")
	require.NoError(t, err, "a minor release may change the contract")
	assert.Contains(t, msg, "is not a patch release of 2.1.0")

	pre := []release{{
		Version: "2.1.0-rc.1", ContractVersion: "2.1.0-rc.1", MinContractVersion: "2.1.0-rc.1", Type: typeFreshInstall,
		Fingerprints: map[string]map[string]string{scheme: digests(fpA)},
	}}
	msg, _, err = checkRelease(pre, digests(fpB), "2.1")
	require.NoError(t, err, "a release after a pre-release may change the contract")
	assert.Contains(t, msg, "is not a patch release of 2.1.0-rc.1")

	seed := []release{{Version: seedVersion, ContractVersion: seedVersion, MinContractVersion: seedVersion, Type: typeFreshInstall}}
	msg, _, err = checkRelease(seed, digests(fpA), "2.0")
	require.NoError(t, err, "the seed is a pre-release")
	assert.Contains(t, msg, "is not a patch release of 2.0.0-alpha.1")

	// A record whose fingerprints were blanked does not read, and one
	// without the current scheme stops the check: neither passes.
	_, err = readReleases(writeReleases(t, map[string]string{"2.1.0.yaml": record("2.1.0", "2.1.0", "2.1.0", "fresh-install", "[]", nil)}))
	require.ErrorContains(t, err, "fingerprints: missing")
	other := []release{{
		Version: "2.1.0", ContractVersion: "2.1.0", MinContractVersion: "2.1.0", Type: typeFreshInstall,
		Fingerprints: map[string]map[string]string{"c99": digests(fpA)},
	}}
	_, _, err = checkRelease(other, digests(fpA), "2.1")
	require.ErrorContains(t, err, "has no fingerprints under scheme")
}

// TestSeedRelease checks the record of the first release, which predates
// fingerprinting and is exempt by name.
func TestSeedRelease(t *testing.T) {
	t.Parallel()

	releases, err := readReleases(filepath.Join("..", "..", filepath.FromSlash(releasesDir)))
	require.NoError(t, err)
	want := release{
		APIVersion:         recordAPIVersion,
		Kind:               recordKind,
		Version:            seedVersion,
		ContractVersion:    seedVersion,
		MinContractVersion: seedVersion,
		Type:               typeFreshInstall,
		UpgradableFrom:     []string{},
	}
	assert.Equal(t, want, releases[0])
	assert.True(t, releases[0].isSeed())
}
