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
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
)

// Release records are frozen: a record says what a release was cut with,
// and the tag of the release carries it, so the only change a record can
// take is a new scheme's fingerprints, computed from that tag. Anything
// else would let a branch claim a contract, a window or an upgrade path
// its release never had. runCheckRecords enforces it on the records a
// change touches.

// A fileChange is a change to a file, as git diff --name-status or the
// files API of a pull request lists it.
type fileChange struct {
	// status is A (added), C (copied), D (deleted), M (modified), R
	// (renamed) or T (type changed), or statusPresent for a file checked
	// as it is.
	status byte
	// path is the path after the change.
	path string
	// from is the path before a rename or copy.
	from string
}

// statusPresent is the status of a file -all checks.
const statusPresent = '='

//nolint:gochecknoglobals // immutable lookup table
var filesAPIStatus = map[string]byte{
	"added": 'A', "copied": 'C', "removed": 'D', "modified": 'M', "renamed": 'R', "changed": 'T',
}

// parseFileChanges reads changes, one per line: a status, a tab and a
// path, or for a rename or copy the status, the old path and the new one.
// A status is a letter of git diff --name-status, with or without its
// score, or a status of the files API of a pull request; unchanged files
// are skipped.
func parseFileChanges(r io.Reader) ([]fileChange, error) {
	var changes []fileChange
	s := bufio.NewScanner(r)
	for n := 1; s.Scan(); n++ {
		line := strings.TrimRight(s.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if fields[0] == "unchanged" {
			continue
		}
		status, ok := parseStatus(fields[0])
		if !ok {
			return nil, fmt.Errorf("line %d: unknown status %q", n, fields[0])
		}
		c := fileChange{status: status}
		switch {
		case (status == 'R' || status == 'C') && len(fields) == 3:
			c.from, c.path = fields[1], fields[2]
		case status != 'R' && status != 'C' && len(fields) == 2:
			c.path = fields[1]
		default:
			return nil, fmt.Errorf("line %d: %q is not a status and its paths, separated by tabs", n, line)
		}
		changes = append(changes, c)
	}
	return changes, s.Err()
}

// parseStatus returns the status a word of git diff --name-status or of the
// files API of a pull request names.
func parseStatus(word string) (byte, bool) {
	if status, ok := filesAPIStatus[word]; ok {
		return status, true
	}
	if word == "" || strings.Trim(word[1:], "0123456789") != "" {
		return 0, false
	}
	switch word[0] {
	case 'A', 'C', 'D', 'M', 'R', 'T':
		return word[0], true
	}
	return 0, false
}

// runCheckRecords checks the release records the changes listed in a file
// touch: no record is deleted, renamed or edited, and an added or
// modified record is the one the tag of its release carries, with only
// fingerprints added that this generator computes from that tag. The
// tags must be here; a missing tag fails, since a record is added only
// from its release. With -all, it checks every record as it is instead,
// which catches edits but no deletion.
func runCheckRecords(root string, args []string, notices io.Writer) (string, error) {
	flags := flag.NewFlagSet("check-records", flag.ContinueOnError)
	changesFile := flags.String("changes", "", "file listing the changed paths with their status, as git diff --name-status prints them")
	all := flags.Bool("all", false, "check every record as it is, when the changes are unknown, such as for a new branch")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	if flags.NArg() > 0 {
		return "", fmt.Errorf("check-records: unexpected argument %q", flags.Arg(0))
	}
	changes, err := readFileChanges(root, *changesFile, *all)
	if err != nil {
		return "", err
	}

	var errs []error
	checked := 0
	for _, c := range changes {
		if !underAny(c.path, []string{releasesDir}) && !underAny(c.from, []string{releasesDir}) {
			continue
		}
		checked++
		if err := checkChange(root, c, notices); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return "", fmt.Errorf("release records are frozen: only a new record, as the tag of its release carries it, "+
			"or fingerprints of a new scheme computed from that tag may be added (see contract/MAINTAINING.md):\n%w", errors.Join(errs...),
		)
	}
	if checked == 0 {
		return "no release record changed", nil
	}
	return fmt.Sprintf("%d release record %s checked against the tags of their releases", checked, plural(checked, "change", "changes")), nil
}

// readFileChanges returns the changes listed in file, or with all a change
// of status statusPresent for every file of the releases directory.
func readFileChanges(root, file string, all bool) ([]fileChange, error) {
	switch {
	case (file == "") == !all:
		return nil, errors.New("check-records takes either -changes or -all")
	case all:
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(releasesDir)))
		if err != nil {
			return nil, err
		}
		changes := make([]fileChange, 0, len(entries))
		for _, e := range entries {
			changes = append(changes, fileChange{status: statusPresent, path: releasesDir + "/" + e.Name()})
		}
		return changes, nil
	}
	f, err := os.Open(file) //nolint:gosec // the file named on the command line
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read only
	changes, err := parseFileChanges(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return changes, nil
}

// checkChange checks one change to the releases directory.
func checkChange(root string, c fileChange, notices io.Writer) error {
	switch c.status {
	case 'D':
		return fmt.Errorf("%s: deleted; a release record is never deleted", c.path)
	case 'R':
		return fmt.Errorf("%s: renamed to %s; a release record is never renamed", c.from, c.path)
	case 'T':
		return fmt.Errorf("%s: its type changed; a release record is a regular file", c.path)
	}
	v, err := recordVersion(c.path)
	if err != nil {
		return err
	}
	if v != seedVersion {
		return checkRecord(root, c.path, v)
	}
	switch c.status {
	case 'M':
		return fmt.Errorf("%s: modified; the record of the seed release is never modified once merged", c.path)
	case statusPresent:
	default:
		notifyf(notices, "%s: no tag carries the record of the seed release, so only later changes to it are checked", c.path)
	}
	return nil
}

// checkRecord checks that the record at file, of the release v, is the one
// the tag of v carries, with only fingerprints added that this generator
// computes from that tag.
func checkRecord(root, file, v string) error {
	r, err := readRecordFile(root, file)
	if err != nil {
		return err
	}
	if r.Version != v {
		return fmt.Errorf("%s: records release %s; it must be named %s.yaml", file, r.Version, r.Version)
	}
	tag := "v" + v
	tagged, err := taggedRecord(root, file, tag)
	if err != nil {
		return err
	}

	var errs []error
	var added []string
	for _, s := range slices.Sorted(maps.Keys(r.Fingerprints)) {
		if _, ok := tagged.Fingerprints[s]; !ok {
			added = append(added, s)
		}
	}
	for _, s := range slices.Sorted(maps.Keys(tagged.Fingerprints)) {
		if !maps.Equal(r.Fingerprints[s], tagged.Fingerprints[s]) {
			errs = append(errs, fmt.Errorf("%s: fingerprints.%s differs from the one %s carries; digests are never changed or removed", file, s, tag))
		}
	}
	// The fingerprints are compared above; every other field must be equal.
	stripped := r
	stripped.Fingerprints, tagged.Fingerprints = nil, nil
	if fields := differentFields(stripped, tagged); len(fields) > 0 {
		errs = append(errs, fmt.Errorf("%s: %s %s from the record %s carries", file, strings.Join(fields, ", "), plural(len(fields), "differs", "differ"), tag))
	}
	for _, s := range added {
		if s != scheme {
			errs = append(errs, fmt.Errorf("%s: fingerprints.%s was added, but this generator computes only scheme %s, so it cannot check them", file, s, scheme))
			continue
		}
		fps, err := fingerprintsAt(root, tag)
		if err != nil {
			return err
		}
		if !maps.Equal(r.Fingerprints[s], fps) {
			errs = append(errs, fmt.Errorf("%s: fingerprints.%s is not what rebaseline computes from %s, for %s",
				file, s, tag, strings.Join(differentInputs(r.Fingerprints[s], fps), ", "),
			))
		}
		if dropped := droppedInputs(r, fps); len(dropped) > 0 {
			errs = append(errs, fmt.Errorf("%s: fingerprints.%s leaves out %s, which rebaseline refuses: comparisons under %s would skip %s",
				file, s, strings.Join(dropped, ", "), s, plural(len(dropped), "it", "them"),
			))
		}
	}
	return errors.Join(errs...)
}

// taggedRecord returns the record at file in the tree of tag, which must
// be here and carry it.
func taggedRecord(root, file, tag string) (release, error) {
	if !hasRef(root, tag) {
		return release{}, fmt.Errorf("%s: %w; a record is added only from the tag of its release, so a record without one fails", file, requireTag(root, tag))
	}
	data, ok, err := showFile(root, tag, file)
	switch {
	case err != nil:
		return release{}, err
	case !ok:
		return release{}, fmt.Errorf("%s: %s carries no record of its release, so this one did not come from it", file, tag)
	}
	tagged, err := decodeRecord(data)
	if err != nil {
		return release{}, fmt.Errorf("%s: the record %s carries: %w", file, tag, err)
	}
	return tagged, nil
}

// differentFields returns the fields, by their YAML name, that differ
// between two records.
func differentFields(a, b release) []string {
	var fields []string
	ta := reflect.TypeFor[release]()
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	for i := range ta.NumField() {
		if !reflect.DeepEqual(va.Field(i).Interface(), vb.Field(i).Interface()) {
			name, _, _ := strings.Cut(ta.Field(i).Tag.Get("json"), ",")
			fields = append(fields, name)
		}
	}
	return fields
}

// errNotRecord is returned for a file of the releases directory that is
// not a release record.
var errNotRecord = errors.New("contract/releases holds only release records, named <version>.yaml")

// recordVersion returns the release a record path names.
func recordVersion(file string) (string, error) {
	dir, name := filepath.Split(filepath.FromSlash(file))
	if filepath.ToSlash(filepath.Clean(dir)) != releasesDir || !strings.HasSuffix(name, ".yaml") {
		return "", fmt.Errorf("%s: %w", file, errNotRecord)
	}
	v := strings.TrimSuffix(name, ".yaml")
	if err := releaseVersion(v); err != nil {
		return "", fmt.Errorf("%s: %w: %w", file, errNotRecord, err)
	}
	return v, nil
}

// readRecordFile returns the record at file, relative to root.
func readRecordFile(root, file string) (release, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file))) //nolint:gosec // a release record of this repository
	if err != nil {
		return release{}, err
	}
	r, err := decodeRecord(data)
	if err != nil {
		return release{}, fmt.Errorf("%s: %w", file, err)
	}
	return r, nil
}
