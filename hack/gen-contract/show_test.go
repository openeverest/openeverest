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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// show runs gen-contract show in the repository at root and returns what
// it prints, its notices and its error.
func show(t *testing.T, root string, args ...string) (string, string, error) {
	t.Helper()

	var notices bytes.Buffer
	msg, err := execute(append([]string{"-root", root, "show"}, args...), io.Discard, &notices)
	return msg, notices.String(), err
}

func TestShow(t *testing.T) {
	t.Parallel()

	// A release 2.1.0, tagged, whose record holds an older scheme too; the
	// working tree then moves on to the development value after it.
	files := testRepo()
	files[recordPath("2.1.0")] = recordWith("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]",
		map[string]map[string]string{otherScheme: digests(fpA), "c2": digests(fpB), scheme: testFingerprints(t, files)},
	)
	root := writeRepo(t, files)
	require.NoError(t, run(root))
	runGit(t, root, "init", "-q", "-b", "main")
	commit(t, root, "release", "v2.1.0")
	writeFile(t, root, changesFile, testChanges+"- #7: a change.\n")
	require.NoError(t, run(root))

	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"version"}, want: "2.2.0-alpha.0"},
		{args: []string{"version", "-ref", "v2.1.0"}, want: "2.1.0"},
		{args: []string{"min-version"}, want: "2.0.0"},
		{args: []string{"min-version", "-ref", "refs/tags/v2.1.0"}, want: "2.0.0"},
		{args: []string{"scheme"}, want: scheme},
		{args: []string{"development"}, want: "true"},
		{args: []string{"development", "-ref", "v2.1.0"}, want: "false"},
		{args: []string{"development", "-version", "2.0.0-alpha.1.next"}, want: "true"},
		{args: []string{"development", "-version", "2.1.0-alpha.0"}, want: "true"},
		{args: []string{"development", "-version", "2.0.0-dev.4"}, want: "false"},
		{args: []string{"development", "-version", "2.1.0"}, want: "false"},
		{args: []string{"record-schemes", "-version", "2.1.0"}, want: "c1,c2,c99"},
		{args: []string{"record-schemes", "-version", "2.1.0", "-ref", "v2.1.0"}, want: "c1,c2,c99"},
		{args: []string{"record-schemes", "-version", seedVersion}, want: ""},
	}
	for _, tt := range tests {
		got, _, err := show(t, root, tt.args...)
		require.NoError(t, err, tt.args)
		assert.Equal(t, tt.want, got, tt.args)
	}

	for _, tt := range []struct {
		args    []string
		wantErr string
	}{
		{args: nil, wantErr: "show: name one of version, min-version, scheme, development, record-schemes"},
		{args: []string{"versions"}, wantErr: "show: name one of"},
		{args: []string{"version", "x"}, wantErr: `show version: unexpected argument "x"`},
		{args: []string{"version", "-version", "2.1.0"}, wantErr: "show version: takes no -version"},
		{args: []string{"version", "-ref", "v9.9.9"}, wantErr: "show version: v9.9.9 is not here; fetch it first"},
		{args: []string{"scheme", "-ref", "v2.1.0"}, wantErr: "show scheme: takes neither -version nor -ref"},
		{args: []string{"development", "-version", "v2.1.0"}, wantErr: `show development: "v2.1.0" is not a SemVer 2.0.0 version`},
		{args: []string{"development", "-version", "2.1.0", "-ref", "v2.1.0"}, wantErr: "takes -version or -ref, not both"},
		{args: []string{"record-schemes"}, wantErr: "show record-schemes: -version is required"},
		{args: []string{"record-schemes", "-version", "2.1.1"}, wantErr: "the working tree holds no record of release 2.1.1, contract/releases/2.1.1.yaml"},
		{args: []string{"record-schemes", "-version", "2.1.1", "-ref", "v2.1.0"}, wantErr: "v2.1.0 holds no record of release 2.1.1"},
		{args: []string{"record-schemes", "-version", "2.2.0-alpha.0"}, wantErr: "is a development value"},
	} {
		_, _, err := show(t, root, tt.args...)
		require.ErrorContains(t, err, tt.wantErr, tt.args)
	}
}

// TestShowWithoutTheContract checks the rule for a tree that lacks the
// contract: only v1 predates it, so on any other ref a missing file is an
// error that says how a branch gets the contract.
func TestShowWithoutTheContract(t *testing.T) {
	t.Parallel()

	root := gitRepo(t, map[string]string{"README.md": "Before the contract.\n"}, "v1.16.0", "v2.0.0-alpha.1")
	for _, branch := range []string{"v1.x", "release-1.16", "release-2.0"} {
		runGit(t, root, "branch", branch)
	}

	// v1 refs print nothing, and say why.
	for _, args := range [][]string{
		{"version", "-ref", "v1.x"},
		{"min-version", "-ref", "refs/heads/release-1.16"},
		{"development", "-ref", "v1.16.0"},
		{"record-schemes", "-version", "2.1.0", "-ref", "release-1.16"},
		{"record-schemes", "-version", "1.16.0"},
	} {
		got, notices, err := show(t, root, args...)
		require.NoError(t, err, args)
		assert.Empty(t, got, args)
		assert.Contains(t, notices, "is of v1, which predates the contract, so it has no contract/", args)
	}

	// v2 refs fail, and say what to do.
	for _, tt := range []struct {
		args    []string
		wantErr string
	}{
		{
			args: []string{"version", "-ref", "release-2.0"},
			wantErr: "release-2.0 has no contract/VERSION, though only v1 predates the contract. " +
				"Merge main into release-2.0 when preparing its next pre-release, which brings the contract to a release branch cut before it. " +
				"See contract/MAINTAINING.md",
		},
		{args: []string{"record-schemes", "-version", "2.0.0-alpha.2", "-ref", "release-2.0"}, wantErr: "release-2.0 has no contract/releases"},
		{args: []string{"min-version", "-ref", "main"}, wantErr: "main has no contract/MIN_VERSION, though only v1 predates the contract. Every v2 branch carries it"},
		{args: []string{"version", "-ref", "v2.0.0-alpha.1"}, wantErr: "v2.0.0-alpha.1 has no contract/VERSION"},
		{args: []string{"version"}, wantErr: "the working tree has no contract/VERSION"},
		{args: []string{"development"}, wantErr: "the working tree has no contract/VERSION"},
		{args: []string{"record-schemes", "-version", "2.1.0"}, wantErr: "the working tree has no contract/releases"},
	} {
		_, _, err := show(t, root, tt.args...)
		require.ErrorContains(t, err, tt.wantErr, tt.args)
	}
}

func TestIsV1Ref(t *testing.T) {
	t.Parallel()

	for _, ref := range []string{
		"v1.x", "v0.x", "release-1.16", "release-0.9", "origin/release-1.17", "refs/heads/v1.x",
		"refs/remotes/upstream/release-1.2", "v1.16.0", "refs/tags/v1.16.1", "1.16.0", "v1.0.0-rc.1",
	} {
		assert.True(t, isV1Ref(ref), ref)
	}
	for _, ref := range []string{
		"", "main", "HEAD", "v2.x", "release-2.0", "origin/release-2.0", "v2.0.0-alpha.1", "2.1.0",
		"release-e2e-debug", "v1.16", "397bf4a83ae24bf25a769e2d29654ed39b9101ce", "feat/release-1.16",
	} {
		assert.False(t, isV1Ref(ref), ref)
	}
}
