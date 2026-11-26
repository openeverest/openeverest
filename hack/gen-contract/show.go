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
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/blang/semver/v4"
)

// The facts show prints.
const (
	showVersion       = "version"
	showMinVersion    = "min-version"
	showScheme        = "scheme"
	showDevelopment   = "development"
	showRecordSchemes = "record-schemes"
)

// runShow prints one fact of the contract, so that tools that do not build
// Go, such as release workflows, take the paths of the contract and its
// rules from one place rather than parse its files:
//
//	version         contract/VERSION, of the working tree or of -ref
//	min-version     contract/MIN_VERSION, likewise
//	scheme          the fingerprint scheme this generator computes
//	development     whether -version, or contract/VERSION, is a
//	                development value: true or false
//	record-schemes  the fingerprint schemes the record of release -version
//	                holds, comma-separated, read from the working tree or
//	                from the objects of -ref
//
// A file of the contract that is missing is an error, except on a ref of
// v1, which predates the contract: then it prints nothing and says why on
// notices.
func runShow(root string, args []string, notices io.Writer) (string, error) {
	facts := []string{showVersion, showMinVersion, showScheme, showDevelopment, showRecordSchemes}
	if len(args) == 0 || !slices.Contains(facts, args[0]) {
		return "", fmt.Errorf("show: name one of %s", strings.Join(facts, ", "))
	}
	what := args[0]
	flags := flag.NewFlagSet("show "+what, flag.ContinueOnError)
	flags.SetOutput(notices)
	version := flags.String("version", "", "development: the version to check; record-schemes: the release whose record to read")
	ref := flags.String("ref", "", "a git ref to read from instead of the working tree, such as v2.1.0 or origin/release-2.1")
	if err := flags.Parse(args[1:]); err != nil {
		return "", err
	}
	if flags.NArg() > 0 {
		return "", fmt.Errorf("show %s: unexpected argument %q", what, flags.Arg(0))
	}
	if *ref != "" && !hasRef(root, *ref) {
		return "", fmt.Errorf("show %s: %s is not here; fetch it first", what, *ref)
	}

	switch what {
	case showScheme:
		if *version != "" || *ref != "" {
			return "", errors.New("show scheme: takes neither -version nor -ref: it is the scheme of this generator")
		}
		return scheme, nil
	case showVersion, showMinVersion:
		if *version != "" {
			return "", fmt.Errorf("show %s: takes no -version", what)
		}
		file := versionFile
		if what == showMinVersion {
			file = minVersionFile
		}
		return showContractFile(root, *ref, file, notices)
	case showDevelopment:
		return showIsDevelopment(root, *ref, *version, notices)
	default:
		return showSchemesOf(root, *ref, *version, notices)
	}
}

// showContractFile returns the trimmed content of a file of the contract,
// in the working tree or in the tree of ref.
func showContractFile(root, ref, file string, notices io.Writer) (string, error) {
	data, ok, err := readContractFile(root, ref, file)
	switch {
	case err != nil:
		return "", err
	case !ok:
		return "", missingFromRef(ref, "", file, notices)
	}
	return strings.TrimSpace(string(data)), nil
}

// showIsDevelopment returns whether v, or else contract/VERSION of the
// working tree or of ref, is a development value.
func showIsDevelopment(root, ref, v string, notices io.Writer) (string, error) {
	if v != "" && ref != "" {
		return "", errors.New("show development: takes -version or -ref, not both")
	}
	if v == "" {
		var err error
		if v, err = showContractFile(root, ref, versionFile, notices); err != nil || v == "" {
			return "", err
		}
	}
	parsed, err := semver.Parse(v)
	if err != nil {
		return "", fmt.Errorf("show development: %q is not a SemVer 2.0.0 version", v)
	}
	return strconv.FormatBool(isDevelopment(parsed)), nil
}

// showSchemesOf returns the fingerprint schemes the record of the release
// v holds, comma-separated, from the working tree or from the objects of
// ref.
func showSchemesOf(root, ref, v string, notices io.Writer) (string, error) {
	if v == "" {
		return "", errors.New("show record-schemes: -version is required: the release whose record to read")
	}
	if err := releaseVersion(v); err != nil {
		return "", fmt.Errorf("show record-schemes: -version: %w", err)
	}
	data, ok, err := readContractFile(root, ref, recordPath(v))
	if err != nil {
		return "", err
	}
	if !ok {
		hasDir, err := hasContractDir(root, ref, releasesDir)
		switch {
		case err != nil:
			return "", err
		case hasDir:
			return "", fmt.Errorf("show record-schemes: %s holds no record of release %s, %s", where(ref), v, recordPath(v))
		}
		return "", missingFromRef(ref, v, releasesDir, notices)
	}
	r, err := decodeRecord(data)
	if err != nil {
		return "", fmt.Errorf("%s in %s: %w", recordPath(v), where(ref), err)
	}
	schemes := slices.Collect(maps.Keys(r.Fingerprints))
	slices.SortFunc(schemes, func(a, b string) int {
		na, _ := strconv.Atoi(strings.TrimPrefix(a, "c"))
		nb, _ := strconv.Atoi(strings.TrimPrefix(b, "c"))
		return na - nb
	})
	return strings.Join(schemes, ","), nil
}

// readContractFile returns the content of file, relative to the
// repository root, in the working tree or, when ref is set, in the tree
// of ref, and false when there is none.
func readContractFile(root, ref, file string) ([]byte, bool, error) {
	if ref != "" {
		return showFile(root, ref, file)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file))) //nolint:gosec // a file of this repository
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return data, err == nil, err
}

// hasContractDir reports whether dir, relative to the repository root, is
// in the working tree or, when ref is set, in the tree of ref.
func hasContractDir(root, ref, dir string) (bool, error) {
	if ref != "" {
		_, err := git(root, "cat-file", "-e", ref+":"+dir)
		return err == nil, nil
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir)))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil && info.IsDir(), err
}

// missingFromRef returns nil, after a notice, when ref or the release v
// is of v1, which predates the contract and so lacks file; otherwise an
// error that says how a v2 branch gets the contract.
func missingFromRef(ref, v, file string, notices io.Writer) error {
	if isV1Ref(ref) || (v != "" && isV1Ref(v)) {
		name := ref
		if name == "" {
			name = v
		}
		notifyf(notices, "%s is of v1, which predates the contract, so it has no %s", name, file)
		return nil
	}
	fix := "Every v2 branch carries it: merge main into the branch, or restore it."
	if branch, ok := releaseBranchOf(ref); ok {
		fix = fmt.Sprintf("Merge main into %s when preparing its next pre-release, "+
			"which brings the contract to a release branch cut before it.", branch,
		)
	}
	return fmt.Errorf("%s has no %s, though only v1 predates the contract. %s See contract/MAINTAINING.md", where(ref), file, fix)
}

// where names the tree a file is read from.
func where(ref string) string {
	if ref == "" {
		return "the working tree"
	}
	return ref
}

var (
	refPrefix = regexp.MustCompile(`^(?:refs/(?:heads|tags)/|refs/remotes/[^/]+/|origin/|upstream/)`)
	v1xBranch = regexp.MustCompile(`^v([0-9]+)\.x$`)
	anyLine   = regexp.MustCompile(`^release-([0-9]+)\.`)
)

// refName returns the name of a branch or tag ref, without the prefixes
// of a full or remote-tracking ref.
func refName(ref string) string {
	return refPrefix.ReplaceAllString(ref, "")
}

// isV1Ref reports whether ref names v1, which predates the contract: the
// branch v1.x, a release branch release-1.Y, or a tag or version of major
// 1, and those of earlier majors. Every other ref, main, a commit or one
// this cannot tell, is v2.
func isV1Ref(ref string) bool {
	name := refName(ref)
	var major string
	switch {
	case v1xBranch.MatchString(name):
		major = v1xBranch.FindStringSubmatch(name)[1]
	case anyLine.MatchString(name):
		major = anyLine.FindStringSubmatch(name)[1]
	default:
		v, err := semver.Parse(strings.TrimPrefix(name, "v"))
		return err == nil && v.Major < 2
	}
	n, err := strconv.Atoi(major)
	return err == nil && n < 2
}

// releaseBranchOf returns the release branch release-X.Y ref names, if it
// names one.
func releaseBranchOf(ref string) (string, bool) {
	name := refName(ref)
	_, ok := lineOfBranch(name)
	return name, ok
}
