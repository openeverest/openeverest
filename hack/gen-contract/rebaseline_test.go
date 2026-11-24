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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runGit runs git in the repository at root, without the configuration of
// the user.
func runGit(t *testing.T, root string, args ...string) {
	t.Helper()

	gitArgs := make([]string, 0, 10+len(args))
	gitArgs = append(gitArgs,
		"-C", root, "-c", "user.name=Test", "-c", "user.email=test@example.com",
		"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false",
	)
	cmd := exec.CommandContext(t.Context(), "git", append(gitArgs, args...)...) //nolint:gosec // git, with arguments of this test
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), out)
}

// gitRepo returns a new git repository holding files in one commit, tagged
// with tags.
func gitRepo(t *testing.T, files map[string]string, tags ...string) string {
	t.Helper()

	root := writeRepo(t, files)
	runGit(t, root, "init", "-q", "-b", "main")
	commit(t, root, "init", tags...)
	return root
}

// commit commits every change of the repository at root and tags the
// commit with tags.
func commit(t *testing.T, root, msg string, tags ...string) {
	t.Helper()

	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-q", "--allow-empty", "-m", msg)
	for _, tag := range tags {
		runGit(t, root, "tag", tag)
	}
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestExtractInputs(t *testing.T) {
	t.Parallel()

	files := testRepo()
	files["contract/releases/README.md"] = "not an input\n"
	files["ui/x.ts"] = "not an input\n"
	root := gitRepo(t, files, "v2.1.0")
	// The working tree moves on; the tag does not.
	replace(t, files, crdPath, "maximum: 10", "maximum: 9")
	writeFile(t, root, crdPath, files[crdPath])

	want := testFingerprints(t, testRepo())
	got, err := fingerprintsAt(root, "v2.1.0")
	require.NoError(t, err)
	assert.Equal(t, want, got, "the fingerprints of a tag are those of its tree")

	dir, cleanup, err := extractInputs(root, "v2.1.0")
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "ui"))
	require.ErrorIs(t, err, os.ErrNotExist, "only the files of inputs are extracted")
	cleanup()
	_, err = os.Stat(dir)
	require.ErrorIs(t, err, os.ErrNotExist)

	_, err = fingerprintsAt(root, "v9.9.9")
	require.ErrorContains(t, err, "tag v9.9.9 is not here; fetch it: git fetch --depth=1 origin refs/tags/v9.9.9:refs/tags/v9.9.9")
}

// TestExtractInputsInAShallowClone checks that a tag fetched alone at depth
// 1 is enough.
func TestExtractInputsInAShallowClone(t *testing.T) {
	t.Parallel()

	origin := gitRepo(t, testRepo(), "v2.1.0")
	writeFile(t, origin, changesFile, testChanges+"- #7: a change.\n")
	commit(t, origin, "later", "v2.2.0")

	clone := t.TempDir()
	runGit(t, clone, "init", "-q")
	runGit(t, clone, "fetch", "-q", "--depth=1", "file://"+origin, "refs/tags/v2.1.0:refs/tags/v2.1.0")
	got, err := fingerprintsAt(clone, "v2.1.0")
	require.NoError(t, err)
	assert.Equal(t, testFingerprints(t, testRepo()), got)
}

// releaseRepo returns a git repository holding testRepo and the record of
// release 2.1.0, as rec, tagged v2.1.0.
func releaseRepo(t *testing.T, rec string) string {
	t.Helper()

	files := testRepo()
	files[recordPath("2.1.0")] = rec
	return gitRepo(t, files, "v2.1.0")
}

// otherScheme is a scheme this generator does not compute, as the scheme
// of a record cut before a scheme bump is to the generator after it.
const otherScheme = "c99"

// olderRecord returns a record of 2.1.0 that holds only otherScheme.
func olderRecord() string {
	return recordWith("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]", map[string]map[string]string{otherScheme: digests(fpA)})
}

func TestRebaseline(t *testing.T) {
	t.Parallel()

	// A record holds only an older scheme, so the generator fails.
	rec := "# A comment that stays.\n" + olderRecord()
	root := releaseRepo(t, rec)
	require.ErrorContains(t, run(root), "rebaseline -version 2.1.0")

	msg, err := execute([]string{"-root", root, "rebaseline", "-version", "2.1.0"}, io.Discard, io.Discard)
	require.NoError(t, err)
	assert.Equal(t, "added the fingerprints of v2.1.0 under scheme "+scheme+" to contract/releases/2.1.0.yaml: crds, http-api, storage-classes, go-api, changes", msg)

	fps := testFingerprints(t, testRepo())
	var block strings.Builder
	block.WriteString("  " + scheme + ":\n")
	for _, name := range inputNames() {
		block.WriteString("    " + name + ": " + fps[name] + "\n")
	}
	assert.Equal(t, rec+block.String(), readFile(t, root, recordPath("2.1.0")), "rebaseline only adds lines")

	// Now the generator compares with the new scheme.
	require.NoError(t, run(root))
	assert.Equal(t, "2.1.0\n", readFile(t, root, versionFile))

	// Again: nothing changes.
	msg, err = execute([]string{"-root", root, "rebaseline", "-version", "2.1.0"}, io.Discard, io.Discard)
	require.NoError(t, err)
	assert.Contains(t, msg, "holds the fingerprints of v2.1.0 under scheme "+scheme+" already")

	// Digests that differ are never rewritten.
	writeFile(t, root, recordPath("2.1.0"), strings.Replace(readFile(t, root, recordPath("2.1.0")), fps["crds"], fpB, 1))
	_, err = execute([]string{"-root", root, "rebaseline", "-version", "2.1.0"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "holds fingerprints under scheme "+scheme+" that differ from those computed from v2.1.0, for crds: a record is never rewritten")
}

// TestRebaselineAbsentInputs checks that an input absent from the tag is
// not recorded, and that a missing CHANGES is recorded as the empty log.
func TestRebaselineAbsentInputs(t *testing.T) {
	t.Parallel()

	files := testRepo()
	for _, f := range []string{contractPath, "contract/check_test.go", "contract/testdata/order.json", "backupstorageclasses/s3/s3.go", changesFile} {
		delete(files, f)
	}
	files[recordPath("2.1.0")] = olderRecord()
	root := gitRepo(t, files, "v2.1.0")
	_, err := execute([]string{"-root", root, "rebaseline", "-version", "2.1.0"}, io.Discard, io.Discard)
	require.NoError(t, err)
	releases, err := readReleases(filepath.Join(root, filepath.FromSlash(releasesDir)))
	require.NoError(t, err)
	recorded := releases[len(releases)-1].Fingerprints[scheme]
	assert.NotContains(t, recorded, "go-api")
	assert.Equal(t, testFingerprints(t, map[string]string{changesFile: "# Only comments.\n"})["changes"], recorded["changes"],
		"a missing CHANGES is the empty log",
	)

	// The tree grows the Go API and the log after the release: the Go API
	// is not compared, and the message says so; the log is.
	for _, f := range []string{contractPath, "backupstorageclasses/s3/s3.go", changesFile} {
		writeFile(t, root, f, testRepo()[f])
	}
	var notices bytes.Buffer
	require.NoError(t, generate(root, &notices))
	assert.Equal(t, "2.1.0\n", readFile(t, root, versionFile))
	assert.Contains(t, notices.String(), "not compared: go-api, which the record of release 2.1.0 does not hold")
	writeFile(t, root, changesFile, testChanges+"- #7: a change.\n")
	require.NoError(t, generate(root, io.Discard))
	assert.Equal(t, "2.2.0-alpha.0\n", readFile(t, root, versionFile))
}

func TestRebaselineErrors(t *testing.T) {
	t.Parallel()

	files := testRepo()
	files[recordPath("2.1.0")] = olderRecord()
	files[recordPath("2.1.1")] = "# Fingerprints are not in block style.\n" +
		strings.Replace(recordWith("2.1.1", "2.1.0", "2.0.0", "upgradable", "[2.1.0]", nil), "upgradableFrom: [2.1.0]\n",
			"upgradableFrom: [2.1.0]\nfingerprints: {"+otherScheme+": {changes: "+fpA+"}}\n", 1,
		)
	root := gitRepo(t, files, "v2.1.0", "v2.1.1")
	tests := []struct {
		args    []string
		wantErr string
	}{
		{args: nil, wantErr: "-version is required"},
		{args: []string{"-version", "v2.1.0"}, wantErr: `-version: "v2.1.0" is not a SemVer 2.0.0 version`},
		{args: []string{"-version", "2.1.2"}, wantErr: "2.1.2.yaml: no such file or directory"},
		{args: []string{"-version", seedVersion}, wantErr: "release 2.0.0-alpha.1 is the seed, which predates fingerprinting and is exempt by name"},
		{args: []string{"-version", "2.1.0", "-tag", "v2.1.9"}, wantErr: "tag v2.1.9 is not here; fetch it"},
		{args: []string{"-version", "2.1.1"}, wantErr: "fingerprints is not in block style"},
		{args: []string{"-version", "2.1.0", "x"}, wantErr: `rebaseline: unexpected argument "x"`},
	}
	for _, tt := range tests {
		_, err := execute(append([]string{"-root", root, "rebaseline"}, tt.args...), io.Discard, io.Discard)
		require.ErrorContains(t, err, tt.wantErr, tt.args)
	}

	// -tag reads another tag.
	_, err := execute([]string{"-root", root, "rebaseline", "-version", "2.1.0", "-tag", "v2.1.1"}, io.Discard, io.Discard)
	require.NoError(t, err)
}

func TestAddSchemeKeepsTheRest(t *testing.T) {
	t.Parallel()

	rec := olderRecord() + "# A trailing comment.\n"
	r, err := decodeRecord([]byte(rec))
	require.NoError(t, err)
	got, err := addScheme([]byte(rec), r, scheme, map[string]string{"changes": fpB, "crds": fpB})
	require.NoError(t, err)
	assert.Equal(t, olderRecord()+"  "+scheme+":\n    crds: "+fpB+"\n    changes: "+fpB+"\n# A trailing comment.\n", string(got))
}
