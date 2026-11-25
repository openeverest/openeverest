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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnifiedDiff(t *testing.T) {
	t.Parallel()

	a := strings.Split("a b c d e f g h i j k l m n o p", " ")
	b := strings.Split("a b X d e f g h i j k l m n p q", " ")
	got := unifiedDiff(diffLines(a, b), "old", "new")
	want := []string{
		"--- old", "+++ new",
		"@@ -1,6 +1,6 @@", " a", " b", "-c", "+X", " d", " e", " f",
		"@@ -12,5 +12,5 @@", " l", " m", " n", "-o", " p", "+q",
	}
	assert.Equal(t, want, got)

	assert.Nil(t, unifiedDiff(diffLines(a, a), "old", "new"), "no change, no diff")

	// A hunk names the entry it is in.
	a = []string{"# one", "1", "2", "3", "4", "5", "6", "7", "8", "# two", "x", "y"}
	b = []string{"# one", "1", "2", "3", "4", "5", "6", "7", "8", "# two", "x", "z"}
	assert.Equal(t, []string{"--- old", "+++ new", "@@ -9,4 +9,4 @@ # two", " 8", " # two", " x", "-y", "+z"}, unifiedDiff(diffLines(a, b), "old", "new"))
	assert.Equal(t, []string{"--- old", "+++ new", "@@ -0,0 +1,2 @@", "+x", "+y"}, unifiedDiff(diffLines(nil, []string{"x", "y"}), "old", "new"))
	assert.Equal(t, []string{"--- old", "+++ new", "@@ -1,1 +0,0 @@", "-x"}, unifiedDiff(diffLines([]string{"x"}, nil), "old", "new"))
}

// TestDiffLinesRebuildsBothSides checks that an edit script, the shortest
// or the one past maxEditDistance, turns one side into the other.
func TestDiffLinesRebuildsBothSides(t *testing.T) {
	t.Parallel()

	lines := func(n, every int, tag string) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprint(i)
			if every > 0 && i%every == 0 {
				out[i] += tag
			}
		}
		return out
	}
	tests := []struct {
		name string
		a, b []string
	}{
		{name: "scattered", a: lines(200, 7, "a"), b: lines(220, 5, "b")},
		{name: "past the limit", a: lines(3000, 1, "a"), b: lines(2500, 1, "b")},
		{name: "empty", a: nil, b: nil},
	}
	for _, tt := range tests {
		edits := diffLines(tt.a, tt.b)
		var a, b []string
		for i, e := range edits {
			if e.kind != '+' {
				assert.Equal(t, len(a), e.a, "%s: edit %d", tt.name, i)
				a = append(a, e.line)
			}
			if e.kind != '-' {
				assert.Equal(t, len(b), e.b, "%s: edit %d", tt.name, i)
				b = append(b, e.line)
			}
		}
		assert.Equal(t, tt.a, a, tt.name)
		assert.Equal(t, tt.b, b, tt.name)
	}
}

func TestDiffCommand(t *testing.T) {
	t.Parallel()

	root := releaseRepo(t, record("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]", testFingerprints(t, testRepo())))
	files := testRepo()
	replace(t, files, crdPath, "maximum: 10", "maximum: 9")
	writeFile(t, root, crdPath, files[crdPath])
	before, after := testFingerprints(t, testRepo()), testFingerprints(t, files)

	var out bytes.Buffer
	_, err := execute([]string{"-root", root, "diff"}, &out, io.Discard)
	require.NoError(t, err)
	got := out.String()
	assert.Contains(t, got, "=== crds: "+before["crds"]+" at v2.1.0, "+after["crds"]+" in the working tree\n"+
		"--- crds (v2.1.0)\n+++ crds (working tree)\n")
	assert.Contains(t, got, "\n-"+strings.Repeat(" ", 18)+"\"maximum\": 10,\n+"+strings.Repeat(" ", 18)+"\"maximum\": 9,\n", "JSON is indented, a value a line")
	assert.Contains(t, got, "=== changes: unchanged since v2.1.0 ("+before["changes"]+")\n")
	assert.Contains(t, got, "=== go-api: unchanged since v2.1.0")

	// The cap.
	out.Reset()
	_, err = execute([]string{"-root", root, "diff", "-input", "crds", "-max-lines", "3"}, &out, io.Discard)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	require.Len(t, lines, 5)
	assert.Equal(t, "--- crds (v2.1.0)", lines[1])
	assert.Regexp(t, `^\.\.\. \d+ more lines; go run ./hack/gen-contract diff -input crds -max-lines 0 prints them all$`, lines[4])

	// Another ref, an input that is not there.
	out.Reset()
	_, err = execute([]string{"-root", root, "diff", "-ref", "v2.1.0", "-input", "changes"}, &out, io.Discard)
	require.NoError(t, err)
	assert.Equal(t, "=== changes: unchanged since v2.1.0 ("+before["changes"]+")\n", out.String())

	for args, wantErr := range map[string]string{
		"-input ui-schema": `-input "ui-schema" is not an input`,
		"-ref v9.9.9":      "tag v9.9.9 is not here",
		"x":                `diff: unexpected argument "x"`,
	} {
		_, err := execute(append([]string{"-root", root, "diff"}, strings.Fields(args)...), io.Discard, io.Discard)
		require.ErrorContains(t, err, wantErr, args)
	}
}

// TestDiffRaw checks that -raw prints the exact bytes each digest hashes.
func TestDiffRaw(t *testing.T) {
	t.Parallel()

	files := testRepo()
	root := gitRepo(t, files, "v2.1.0")
	replace(t, files, crdPath, "maximum: 10", "maximum: 9")
	writeFile(t, root, crdPath, files[crdPath])
	for ref, fps := range map[string]map[string]string{"": testFingerprints(t, files), "v2.1.0": testFingerprints(t, testRepo())} {
		for _, name := range inputNames() {
			var out bytes.Buffer
			_, err := execute([]string{"-root", root, "diff", "-raw", "-input", name, "-ref", ref}, &out, io.Discard)
			require.NoError(t, err)
			sum := sha256.Sum256(out.Bytes())
			assert.Equal(t, fps[name], digestPrefix+hex.EncodeToString(sum[:]), "%s at %q", name, ref)
		}
	}

	var out bytes.Buffer
	_, err := execute([]string{"-root", root, "diff", "-raw"}, &out, io.Discard)
	require.NoError(t, err)
	fps := testFingerprints(t, files)
	assert.True(t, strings.HasPrefix(out.String(), "### crds: "+fps["crds"]+", "))
	assert.True(t, strings.HasSuffix(out.String(), "### changes: "+fps["changes"]+", 20 bytes\ncontract/CHANGES 0\n\n"))

	delete(files, httpAPIPath)
	delete(files, crdsGenPath)
	root = writeRepo(t, files)
	out.Reset()
	_, err = execute([]string{"-root", root, "diff", "-raw"}, &out, io.Discard)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "### http-api: absent\n")
	_, err = execute([]string{"-root", root, "diff", "-raw", "-input", "http-api"}, io.Discard, io.Discard)
	require.ErrorContains(t, err, "http-api is absent from the working tree")
}

// TestCheckReleasePrintsTheDiff checks that a release branch that changed
// the contract shows how, when the tag of its latest release is here.
func TestCheckReleasePrintsTheDiff(t *testing.T) {
	t.Parallel()

	root := releaseRepo(t, record("2.1.0", "2.1.0", "2.0.0", "upgradable", "[2.0.0]", testFingerprints(t, testRepo())))
	files := testRepo()
	replace(t, files, httpAPIPath, "operationId: listWidgets", "operationId: getWidgets")
	writeFile(t, root, httpAPIPath, files[httpAPIPath])
	var notices bytes.Buffer
	_, err := execute([]string{"-root", root, "-check-release", "-branch", "release-2.1"}, io.Discard, &notices)
	require.ErrorContains(t, err, "http-api differs")
	assert.Contains(t, notices.String(), "--- http-api (v2.1.0)\n+++ http-api (working tree)\n")
	assert.Contains(t, notices.String(), "\n-        \"operationId\": \"listWidgets\",\n+        \"operationId\": \"getWidgets\",\n")
	assert.NotContains(t, notices.String(), "=== crds", "only the inputs that differ")
}
