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
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

// runRebaseline adds the fingerprints of the current scheme, computed from
// the tag of a release, to the record of the release. It never changes
// anything else: a record holds what the release was cut with, so a new
// scheme only adds the digests it computes from the same tree, as go.sum
// adds a hash. It refuses to change digests the record holds already.
func runRebaseline(root string, args []string) (string, error) {
	flags := flag.NewFlagSet("rebaseline", flag.ContinueOnError)
	version := flags.String("version", "", "the release whose record to rebaseline, such as 2.1.0")
	tag := flags.String("tag", "", "the tag of the release (default v<version>)")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	switch {
	case flags.NArg() > 0:
		return "", fmt.Errorf("rebaseline: unexpected argument %q", flags.Arg(0))
	case *version == "":
		return "", errors.New("rebaseline: -version is required: the release whose record to rebaseline")
	case *tag == "":
		*tag = "v" + *version
	}
	file := filepath.Join(root, filepath.FromSlash(recordPath(*version)))
	data, r, err := readRecordOf(file, *version)
	if err != nil {
		return "", err
	}
	if r.isSeed() {
		return "", fmt.Errorf("release %s is the seed, which predates fingerprinting and is exempt by name: "+
			"its tag carries no contract package, so no digest computed from it says what changed since", seedVersion,
		)
	}
	fps, err := fingerprintsAt(root, *tag)
	if err != nil {
		return "", err
	}
	if recorded, ok := r.Fingerprints[scheme]; ok {
		if maps.Equal(recorded, fps) {
			return fmt.Sprintf("%s holds the fingerprints of %s under scheme %s already", recordPath(*version), *tag, scheme), nil
		}
		return "", fmt.Errorf("%s holds fingerprints under scheme %s that differ from those computed from %s, for %s: "+
			"a record is never rewritten. If this generator changed how it hashes, bump the scheme",
			recordPath(*version), scheme, *tag, strings.Join(differentInputs(recorded, fps), ", "),
		)
	}
	updated, err := addScheme(data, r, scheme, fps)
	if err != nil {
		return "", fmt.Errorf("%s: %w", recordPath(*version), err)
	}
	info, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(file, updated, info.Mode().Perm()); err != nil {
		return "", err
	}
	return fmt.Sprintf("added the fingerprints of %s under scheme %s to %s: %s",
		*tag, scheme, recordPath(*version), strings.Join(sortedInputs(fps), ", "),
	), nil
}

// readRecordOf returns the content of the record file of the release v,
// and the record.
func readRecordOf(file, v string) ([]byte, release, error) {
	if err := releaseVersion(v); err != nil {
		return nil, release{}, fmt.Errorf("-version: %w", err)
	}
	data, err := os.ReadFile(file) //nolint:gosec // a release record of this repository
	if err != nil {
		return nil, release{}, err
	}
	r, err := decodeRecord(data)
	if err != nil {
		return nil, release{}, fmt.Errorf("%s: %w", recordPath(v), err)
	}
	if r.Version != v {
		return nil, release{}, fmt.Errorf("%s records release %s, not %s", recordPath(v), r.Version, v)
	}
	return data, r, nil
}

// fingerprintsLine is the line that opens the fingerprints of a record.
var fingerprintsLine = regexp.MustCompile(`^fingerprints:[ \t]*(#.*)?$`)

// addScheme returns the content data of the record r with the digests of
// the scheme s added at the end of its fingerprints. It changes no other
// line, so comments and the layout stay, and checks that the result reads
// as r with only that scheme added.
func addScheme(data []byte, r release, s string, digests map[string]string) ([]byte, error) {
	var block strings.Builder
	fmt.Fprintf(&block, "  %s:\n", s)
	for _, name := range sortedInputs(digests) {
		fmt.Fprintf(&block, "    %s: %s\n", name, digests[name])
	}

	out, err := insertScheme(string(data), block.String())
	if err != nil {
		return nil, err
	}

	got, err := decodeRecord([]byte(out))
	if err != nil {
		return nil, fmt.Errorf("adding fingerprints.%s breaks the record: %w", s, err)
	}
	if !maps.Equal(got.Fingerprints[s], digests) {
		return nil, fmt.Errorf("fingerprints.%s does not read back as written; is fingerprints in block style, indented by two spaces?", s)
	}
	delete(got.Fingerprints, s)
	if !reflect.DeepEqual(got, r) {
		return nil, fmt.Errorf("adding fingerprints.%s would change more than that", s)
	}
	return []byte(out), nil
}

// insertScheme returns the text of a record with the lines block, a scheme
// and its digests, added at the end of its fingerprints, which it adds
// when the record has none.
func insertScheme(text, block string) (string, error) {
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	lines := strings.SplitAfter(text, "\n")
	start := slices.IndexFunc(lines, func(l string) bool { return fingerprintsLine.MatchString(strings.TrimRight(l, "\r\n")) })
	switch {
	case start < 0 && slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, "fingerprints:") }):
		return "", errors.New("fingerprints is not in block style, which rebaseline adds to")
	case start < 0:
		return text + "fingerprints:\n" + block, nil
	}
	// The block ends at the next line that is not indented: a top-level
	// key, a comment or the end of the file.
	end := start + 1
	for end < len(lines) && (strings.HasPrefix(lines[end], " ") || strings.TrimSpace(lines[end]) == "") {
		end++
	}
	for end > start+1 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return strings.Join(lines[:end], "") + block + strings.Join(lines[end:], ""), nil
}
