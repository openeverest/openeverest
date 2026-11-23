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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/contract"
)

const (
	fpA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fpB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// record returns the content of a release record.
func record(version, contractVersion, minVersion, typ, upgradableFrom, fp string) string {
	return fmt.Sprintf("version: %s\ncontractVersion: %s\nminContractVersion: %s\ntype: %s\nupgradableFrom: %s\nfingerprint: %q\n",
		version, contractVersion, minVersion, typ, upgradableFrom, fp,
	)
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
		"2.0.0-alpha.1":  "2.0.0-alpha.1.dev",
		"2.0.0-alpha.10": "2.0.0-alpha.10.dev",
		"2.0.0-rc.1":     "2.0.0-rc.1.dev",
		"2.0.0":          "2.0.1-dev",
		"2.1.3":          "2.1.4-dev",
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
// value after a release sorts before every release that can follow it.
func TestDevelopmentValuesSortBeforeTheNextRelease(t *testing.T) {
	t.Parallel()

	tests := []struct{ release, next string }{
		{"2.0.0-alpha.1", "2.0.0-alpha.2"},
		{"2.0.0-alpha.1", "2.0.0-beta.1"},
		{"2.0.0-alpha.1", "2.0.0"},
		{"2.0.0-rc.1", "2.0.0"},
		{"2.0.0", "2.0.1"},
		{"2.0.0", "2.1.0-alpha.1"},
		{"2.0.0", "3.0.0-alpha.1"},
	}
	for _, tt := range tests {
		assert.Negative(t, contract.Compare(developmentAfter(tt.release), tt.next), "%s → %s", tt.release, tt.next)
	}
}

func TestDecide(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		records            map[string]string
		fp                 string
		wantVersion        string
		wantMinVersion     string
		wantUpgradableFrom []string
	}{
		{
			name:           "seed without fingerprint",
			records:        map[string]string{"2.0.0-alpha.1.yaml": testSeedRelease},
			fp:             fpA,
			wantVersion:    "2.0.0-alpha.1.dev",
			wantMinVersion: "2.0.0-alpha.1",
		},
		{
			name: "unchanged since a pre-release",
			records: map[string]string{
				"2.0.0-alpha.1.yaml": testSeedRelease,
				"2.0.0-alpha.2.yaml": record("2.0.0-alpha.2", "2.0.0-alpha.2", "2.0.0-alpha.2", "fresh-install", "[]", fpA),
			},
			fp:             fpA,
			wantVersion:    "2.0.0-alpha.2",
			wantMinVersion: "2.0.0-alpha.2",
		},
		{
			name: "changed since a pre-release",
			records: map[string]string{
				"2.0.0-alpha.1.yaml": testSeedRelease,
				"2.0.0-alpha.2.yaml": record("2.0.0-alpha.2", "2.0.0-alpha.2", "2.0.0-alpha.1", "upgradable", "[2.0.0-alpha.1]", fpA),
			},
			fp:             fpB,
			wantVersion:    "2.0.0-alpha.2.dev",
			wantMinVersion: "2.0.0-alpha.1",
		},
		{
			name: "unchanged since an upgradable release",
			records: map[string]string{
				"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", fpB),
				"2.1.0.yaml": record("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0, 2.0.1]", fpA),
			},
			fp:                 fpA,
			wantVersion:        "2.1.0",
			wantMinVersion:     "2.0.0",
			wantUpgradableFrom: []string{"2.0.0", "2.0.1"},
		},
		{
			name: "changed since a release",
			records: map[string]string{
				"2.1.0.yaml": record("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]", fpA),
			},
			fp:             fpB,
			wantVersion:    "2.1.1-dev",
			wantMinVersion: "2.0.0",
		},
		{
			name: "unchanged since a patch release",
			records: map[string]string{
				"2.1.0.yaml": record("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]", fpA),
				"2.1.1.yaml": record("2.1.1", "2.1.0", "2.0.0", "upgradable", "[2.1.0]", fpA),
			},
			fp:                 fpA,
			wantVersion:        "2.1.0",
			wantMinVersion:     "2.0.0",
			wantUpgradableFrom: []string{"2.1.0"},
		},
		{
			name: "changed since a patch release",
			records: map[string]string{
				"2.1.0.yaml": record("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]", fpA),
				"2.1.1.yaml": record("2.1.1", "2.1.0", "2.0.0", "upgradable", "[2.1.0]", fpA),
			},
			fp:             fpB,
			wantVersion:    "2.1.2-dev",
			wantMinVersion: "2.0.0",
		},
		{
			name: "latest by precedence, not by name",
			records: map[string]string{
				"2.0.0-alpha.9.yaml":  record("2.0.0-alpha.9", "2.0.0-alpha.9", "2.0.0-alpha.9", "fresh-install", "[]", fpB),
				"2.0.0-alpha.10.yaml": record("2.0.0-alpha.10", "2.0.0-alpha.10", "2.0.0-alpha.10", "fresh-install", "[]", fpA),
			},
			fp:             fpA,
			wantVersion:    "2.0.0-alpha.10",
			wantMinVersion: "2.0.0-alpha.10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			releases, err := readReleases(writeReleases(t, tt.records))
			require.NoError(t, err)
			w := decide(releases, tt.fp)
			assert.Equal(t, tt.wantVersion, w.version)
			assert.Equal(t, tt.wantMinVersion, w.minVersion)
			assert.Equal(t, tt.fp, w.fingerprint)
			if tt.wantUpgradableFrom == nil {
				tt.wantUpgradableFrom = []string{}
			}
			assert.Equal(t, tt.wantUpgradableFrom, w.upgradableFrom)
			assert.Equal(t, contract.ReasonSupported, contract.Check(w.version, w.minVersion, w.version).Reason,
				"a core must serve its own version",
			)
		})
	}
}

func TestReadReleasesErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		records map[string]string
		wantErr string
	}{
		{name: "none", records: make(map[string]string), wantErr: "holds no release record"},
		{
			name:    "misnamed",
			records: map[string]string{"2.0.0.yaml": record("2.0.1", "2.0.1", "2.0.1", "fresh-install", "[]", "")},
			wantErr: "must be named 2.0.1.yaml",
		},
		{
			name:    "unknown field",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", "") + "minVersion: 2.0.0\n"},
			wantErr: `unknown field "minVersion"`,
		},
		{
			name:    "v prefix",
			records: map[string]string{"v2.0.0.yaml": record("v2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", "")},
			wantErr: `version: "v2.0.0" is not a SemVer 2.0.0 version`,
		},
		{
			name:    "build metadata",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0+abc", "2.0.0", "fresh-install", "[]", "")},
			wantErr: "contractVersion: \"2.0.0+abc\" carries build metadata",
		},
		{
			name:    "development value",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0-alpha.1.dev", "fresh-install", "[]", "")},
			wantErr: "minContractVersion: \"2.0.0-alpha.1.dev\" is a development value",
		},
		{
			name:    "contract after release",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.1", "2.0.0", "fresh-install", "[]", "")},
			wantErr: "contractVersion 2.0.1 is above version 2.0.0",
		},
		{
			name:    "minimum above contract",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0-rc.1", "2.0.0", "fresh-install", "[]", "")},
			wantErr: "minContractVersion 2.0.0 is above contractVersion 2.0.0-rc.1",
		},
		{
			name:    "unknown type",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "in-place", "[]", "")},
			wantErr: `type "in-place" is neither fresh-install nor upgradable`,
		},
		{
			name:    "fresh install upgradable from",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[2.0.0-rc.1]", "")},
			wantErr: "upgradableFrom must be empty for a fresh-install release",
		},
		{
			name:    "upgradable from nothing",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "upgradable", "[]", "")},
			wantErr: "upgradableFrom must list the releases",
		},
		{
			name:    "upgradable from a later release",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "upgradable", "[2.0.1]", "")},
			wantErr: "upgradableFrom: 2.0.1 is not older than 2.0.0",
		},
		{
			name:    "upgradable from twice",
			records: map[string]string{"2.0.1.yaml": record("2.0.1", "2.0.0", "2.0.0", "upgradable", "[2.0.0, 2.0.0]", "")},
			wantErr: "upgradableFrom: 2.0.0 is listed twice",
		},
		{
			name:    "malformed fingerprint",
			records: map[string]string{"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", "abc")},
			wantErr: `fingerprint "abc" is neither empty nor sha256:`,
		},
		{
			name: "window goes back",
			records: map[string]string{
				"2.0.0.yaml": record("2.0.0", "2.0.0", "2.0.0", "fresh-install", "[]", ""),
				"2.0.1.yaml": record("2.0.1", "2.0.0-rc.1", "2.0.0-rc.1", "fresh-install", "[]", ""),
			},
			wantErr: "release 2.0.1: its contract window [2.0.0-rc.1, 2.0.0-rc.1] goes below the window [2.0.0, 2.0.0] of the earlier release 2.0.0",
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

func TestCheckRelease(t *testing.T) {
	t.Parallel()

	released := []release{{Version: "2.0.0", ContractVersion: "2.0.0", MinContractVersion: "2.0.0", Type: typeFreshInstall, Fingerprint: fpA}}
	require.NoError(t, checkRelease(released, fpA))

	err := checkRelease(released, fpB)
	require.ErrorContains(t, err, "the contract changed since release 2.0.0")
	require.ErrorContains(t, err, fpA)
	require.ErrorContains(t, err, fpB)

	seed := []release{{Version: "2.0.0-alpha.1", ContractVersion: "2.0.0-alpha.1", MinContractVersion: "2.0.0-alpha.1", Type: typeFreshInstall}}
	require.ErrorContains(t, checkRelease(seed, fpA), "the record of release 2.0.0-alpha.1 has no fingerprint")
}

// TestSeedRelease checks the record of the first release, which predates
// fingerprinting.
func TestSeedRelease(t *testing.T) {
	t.Parallel()

	releases, err := readReleases(filepath.Join("..", "..", filepath.FromSlash(releasesDir)))
	require.NoError(t, err)
	want := release{
		Version:            "2.0.0-alpha.1",
		ContractVersion:    "2.0.0-alpha.1",
		MinContractVersion: "2.0.0-alpha.1",
		Type:               typeFreshInstall,
		UpgradableFrom:     []string{},
		Fingerprint:        "",
	}
	assert.Equal(t, want, releases[0])
}
