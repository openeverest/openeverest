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
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/blang/semver/v4"
	"sigs.k8s.io/yaml"
)

// releasesDir holds one record per core release, named <version>.yaml.
const releasesDir = "contract/releases"

// The type of a release record. It is data for this command, not a
// Kubernetes resource; the header lets a later format be told apart.
const (
	recordAPIVersion = "contract.openeverest.io/v1"
	recordKind       = "ReleaseRecord"
)

// seedVersion is the first release, whose record predates fingerprinting.
// Its tag carries no contract package, and the package was added before
// any check decided whether a change touched the contract, so no digest
// computed from that tag would say what main has changed since. It is
// exempt by name: its record has no fingerprints, and while it is the
// latest release, the contract version is the development value after it.
const seedVersion = "2.0.0-alpha.1"

// Types of a release.
const (
	// typeFreshInstall is a release that can only be installed fresh.
	typeFreshInstall = "fresh-install"
	// typeUpgradable is a release that can be upgraded to in place from the
	// releases its record lists.
	typeUpgradable = "upgradable"
)

// A release is the record of a core release: what the person who cut it
// decided, and what the contract was when it was cut.
type release struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	// Computed when the release is cut.

	// Version is the core release.
	Version string `json:"version"`
	// ContractVersion is the contract version extensions built against
	// the release are stamped with: the release itself, or, for a patch
	// release of a final release, the contract version of its line, since
	// a patch release cannot change the contract.
	ContractVersion string `json:"contractVersion"`

	// Decided by the person who cuts the release.

	// MinContractVersion is the oldest contract version the release serves.
	MinContractVersion string `json:"minContractVersion"`
	// Type is typeFreshInstall or typeUpgradable.
	Type string `json:"type"`
	// UpgradableFrom lists the releases an upgradable release can be
	// upgraded from.
	UpgradableFrom []string `json:"upgradableFrom"`

	// Computed, and only ever added to.

	// Fingerprints holds the digests of the inputs of the contract the
	// release was cut with, by fingerprint scheme, then by input. A scheme
	// is added next to the earlier ones and never rewritten, so every
	// digest stays comparable with what computed it. Only the seed has
	// none.
	Fingerprints map[string]map[string]string `json:"fingerprints,omitempty"`
}

var (
	digestPattern     = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	schemePattern     = regexp.MustCompile(`^c[1-9][0-9]*$`)
	inputNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	releaseBranchName = regexp.MustCompile(`^release-(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	releaseLine       = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
)

// isSeed reports whether r is the record of the seed release, which is
// exempt from fingerprints by name.
func (r release) isSeed() bool {
	return r.Version == seedVersion
}

// recordPath returns the path of the record of the release v, relative to
// the repository root.
func recordPath(v string) string {
	return releasesDir + "/" + v + ".yaml"
}

// readReleases returns the release records in dir, sorted by version.
func readReleases(dir string) ([]release, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	releases := make([]release, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file) //nolint:gosec // a release record of this repository
		if err != nil {
			return nil, err
		}
		r, err := decodeRecord(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if want := r.Version + ".yaml"; filepath.Base(file) != want {
			return nil, fmt.Errorf("%s: the record of release %s must be named %s", file, r.Version, want)
		}
		releases = append(releases, r)
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("%s holds no release record", dir)
	}
	slices.SortFunc(releases, func(a, b release) int { return compare(a.Version, b.Version) })
	for i := 1; i < len(releases); i++ {
		if err := checkContractOrder(releases[i-1], releases[i]); err != nil {
			return nil, err
		}
	}
	return releases, nil
}

// checkContractOrder returns an error when the contract version of the
// release r is below that of prev, the release before it. The minimum has
// no such rule: a fresh-install release can serve only its own extensions,
// and a later upgradable release can serve older ones again, as long as
// each of its upgrade paths keeps the extensions of the release it starts
// from working.
func checkContractOrder(prev, r release) error {
	if compare(r.ContractVersion, prev.ContractVersion) < 0 {
		return fmt.Errorf("release %s: its contract version %s is below %s, the contract version of the earlier release %s",
			r.Version, r.ContractVersion, prev.ContractVersion, prev.Version,
		)
	}
	return nil
}

// decodeRecord returns the release record in data, valid.
//
// Every format a release has shipped stays readable for good: tags carry
// their records in the format of their time, and the commands that
// rebaseline a record or check that a record equals the one of its tag
// read them. A new format adds a case here and keeps the earlier ones.
func decodeRecord(data []byte) (release, error) {
	var header struct {
		APIVersion string `json:"apiVersion"`
		Kind       string `json:"kind"`
	}
	if err := yaml.Unmarshal(data, &header); err != nil {
		return release{}, err
	}
	if header.Kind != recordKind {
		return release{}, fmt.Errorf("kind %q is not %s", header.Kind, recordKind)
	}
	var r release
	switch header.APIVersion {
	case recordAPIVersion:
		if err := yaml.UnmarshalStrict(data, &r); err != nil {
			return release{}, err
		}
	default:
		return release{}, fmt.Errorf("apiVersion %q is not %s, the only format this command reads", header.APIVersion, recordAPIVersion)
	}
	if r.UpgradableFrom == nil {
		r.UpgradableFrom = []string{}
	}
	if err := r.validate(); err != nil {
		return release{}, err
	}
	return r, nil
}

func (r release) validate() error {
	var errs []error
	for _, v := range []struct{ field, value string }{
		{"version", r.Version},
		{"contractVersion", r.ContractVersion},
		{"minContractVersion", r.MinContractVersion},
	} {
		if err := releaseVersion(v.value); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", v.field, err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	if compare(r.ContractVersion, r.Version) > 0 {
		errs = append(errs, fmt.Errorf("contractVersion %s is above version %s", r.ContractVersion, r.Version))
	}
	if compare(r.MinContractVersion, r.ContractVersion) > 0 {
		errs = append(errs, fmt.Errorf("minContractVersion %s is above contractVersion %s", r.MinContractVersion, r.ContractVersion))
	}
	errs = append(errs, r.validateUpgrades(), r.validateFingerprints())
	return errors.Join(errs...)
}

// validateUpgrades checks the type of a release against the releases it
// can be upgraded from.
func (r release) validateUpgrades() error {
	var errs []error
	switch r.Type {
	case typeFreshInstall:
		if len(r.UpgradableFrom) > 0 {
			errs = append(errs, fmt.Errorf("upgradableFrom must be empty for a %s release", typeFreshInstall))
		}
	case typeUpgradable:
		if len(r.UpgradableFrom) == 0 {
			errs = append(errs, fmt.Errorf("upgradableFrom must list the releases an %s release can be upgraded from", typeUpgradable))
		}
	default:
		errs = append(errs, fmt.Errorf("type %q is neither %s nor %s", r.Type, typeFreshInstall, typeUpgradable))
	}
	for i, from := range r.UpgradableFrom {
		switch err := releaseVersion(from); {
		case err != nil:
			errs = append(errs, fmt.Errorf("upgradableFrom: %w", err))
		case compare(from, r.Version) >= 0:
			errs = append(errs, fmt.Errorf("upgradableFrom: %s is not older than %s", from, r.Version))
		case slices.Contains(r.UpgradableFrom[:i], from):
			errs = append(errs, fmt.Errorf("upgradableFrom: %s is listed twice", from))
		}
	}
	return errors.Join(errs...)
}

// validateFingerprints checks the fingerprints of a release. Only the seed
// has none, so removing them cannot turn a check off. A record may hold
// schemes this command does not compute, which a later generator added;
// under the one it computes, it holds only inputs it knows, and always
// changes, which no tree lacks.
func (r release) validateFingerprints() error {
	if r.isSeed() {
		if r.Fingerprints != nil {
			return fmt.Errorf("fingerprints: release %s predates fingerprinting and is exempt by name, so its record has none", seedVersion)
		}
		return nil
	}
	if len(r.Fingerprints) == 0 {
		return fmt.Errorf("fingerprints: missing; only the record of release %s may have none. "+
			"Restore the record from its tag: git show v%s:%s", seedVersion, r.Version, recordPath(r.Version),
		)
	}
	known := inputNames()
	var errs []error
	for _, s := range slices.Sorted(maps.Keys(r.Fingerprints)) {
		digests := r.Fingerprints[s]
		if !schemePattern.MatchString(s) {
			errs = append(errs, fmt.Errorf("fingerprints: %q is not a scheme such as c1", s))
			continue
		}
		if len(digests) == 0 {
			errs = append(errs, fmt.Errorf("fingerprints.%s: holds no digest", s))
			continue
		}
		for _, name := range slices.Sorted(maps.Keys(digests)) {
			switch {
			case !inputNamePattern.MatchString(name):
				errs = append(errs, fmt.Errorf("fingerprints.%s: %q is not an input name", s, name))
			case s == scheme && !slices.Contains(known, name):
				errs = append(errs, fmt.Errorf("fingerprints.%s: %s is not an input of the scheme, which has %s", s, name, strings.Join(known, ", ")))
			case !digestPattern.MatchString(digests[name]):
				errs = append(errs, fmt.Errorf("fingerprints.%s.%s: %q is not sha256:<64 lowercase hex digits>", s, name, digests[name]))
			}
		}
		if _, ok := digests["changes"]; s == scheme && !ok {
			errs = append(errs, fmt.Errorf("fingerprints.%s: changes is missing, though no tree lacks it", s))
		}
	}
	return errors.Join(errs...)
}

// releaseVersion checks that v can name a release: a SemVer 2.0.0 version
// without build metadata, which SemVer precedence ignores, and without a
// development identifier.
func releaseVersion(v string) error {
	parsed, err := semver.Parse(v)
	switch {
	case err != nil:
		return fmt.Errorf("%q is not a SemVer 2.0.0 version such as 2.1.0 or 2.1.0-alpha.1", v)
	case len(parsed.Build) > 0:
		return fmt.Errorf("%q carries build metadata, which no release does", v)
	case isDevelopment(parsed):
		return fmt.Errorf("%q is a development value, which no release has", v)
	}
	return nil
}

// A window is what this command generates: the contract version of this
// core and the window of contract versions it serves.
type window struct {
	version        string
	minVersion     string
	fingerprints   map[string]string
	upgradableFrom []string
}

// A comparison is the outcome of comparing the fingerprints of a tree with
// those a release record holds under scheme.
type comparison struct {
	release release
	// differ are the inputs whose digests differ from the record's,
	// including those the record holds and the tree lacks.
	differ []string
	// absent are the inputs of differ the tree lacks.
	absent []string
	// skipped are the inputs the tree has and the record does not, which
	// were absent from the tag of the release: they are not compared.
	skipped []string
	// compared are the inputs whose digests are equal.
	compared []string
}

// equal reports whether every input the record holds is the same.
func (c comparison) equal() bool {
	return len(c.differ) == 0
}

// compareWith compares the fingerprints fps of a tree with those the
// record of the release r holds under scheme. An input the record holds
// is compared, and differs when the tree lacks it. An input the record
// does not hold was absent from the tag of the release, so there is
// nothing to compare it with: it is skipped, and messages name it.
func compareWith(r release, fps map[string]string) (comparison, error) {
	recorded, ok := r.Fingerprints[scheme]
	if !ok {
		return comparison{}, noSchemeError(r)
	}
	c := comparison{release: r}
	for _, name := range inputNames() {
		want, inRecord := recorded[name]
		got, here := fps[name]
		switch {
		case inRecord && !here:
			c.differ = append(c.differ, name)
			c.absent = append(c.absent, name)
		case inRecord && want != got:
			c.differ = append(c.differ, name)
		case inRecord:
			c.compared = append(c.compared, name)
		case here:
			c.skipped = append(c.skipped, name)
		}
	}
	return c, nil
}

// noSchemeError says how to add the fingerprints of the current scheme to
// the record of the release r, which has none: without them, nothing can
// tell whether the contract changed since r, and guessing either way would
// move the contract version for no reason or keep it while it changed.
func noSchemeError(r release) error {
	return fmt.Errorf("the record of release %[1]s, %[2]s, has no fingerprints under scheme %[3]s, "+
		"the one this generator computes, so nothing can tell whether the contract changed since %[1]s. "+
		"Add them from the tag of the release:\n\n"+
		"\tgit fetch upstream --tags   # or origin: the remote of openeverest/openeverest\n"+
		"\tgo run ./hack/gen-contract rebaseline -version %[1]s\n\n"+
		"then run make gen-contract again and commit the record. See contract/MAINTAINING.md",
		r.Version, recordPath(r.Version), scheme,
	)
}

// summary describes the outcome of a comparison in a sentence, naming the
// inputs that differ.
func (c comparison) summary() string {
	if c.equal() {
		return fmt.Sprintf("the contract is the one release %s was cut with (%s equal under fingerprint scheme %s)",
			c.release.Version, strings.Join(c.compared, ", "), scheme,
		)
	}
	msg := fmt.Sprintf("the contract changed since release %s: %s %s from its record (fingerprint scheme %s)",
		c.release.Version, strings.Join(c.differ, ", "), plural(len(c.differ), "differs", "differ"), scheme,
	)
	if len(c.absent) > 0 {
		msg += fmt.Sprintf("; %s %s absent here", strings.Join(c.absent, ", "), plural(len(c.absent), "is", "are"))
	}
	return msg
}

// skippedNotice names the inputs a comparison skipped, or is empty.
func (c comparison) skippedNotice() string {
	if len(c.skipped) == 0 {
		return ""
	}
	return fmt.Sprintf("not compared: %s, which the record of release %s does not hold under fingerprint scheme %s, "+
		"since %s absent from its tag",
		strings.Join(c.skipped, ", "), c.release.Version, scheme, plural(len(c.skipped), "it was", "they were"),
	)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// decide works out the window of a core whose contract has the
// fingerprints fps, from the release records sorted by version. While the
// contract is the one the latest release was cut with, the core has that
// release's contract version and can be upgraded from what it can. Once it
// differs, the core has the development value after the latest release,
// which no release can be upgraded to. The minimum moves only at a
// release. While the latest release is the seed, which has no
// fingerprints, the core has the development value after it.
//
// It fails when the latest record has no fingerprints under scheme, rather
// than guess: see noSchemeError.
func decide(releases []release, fps map[string]string) (window, comparison, error) {
	latest := releases[len(releases)-1]
	w := window{minVersion: latest.MinContractVersion, fingerprints: fps, upgradableFrom: []string{}}
	if latest.isSeed() {
		w.version = developmentAfter(latest.Version)
		return w, comparison{release: latest}, nil
	}
	c, err := compareWith(latest, fps)
	if err != nil {
		return window{}, comparison{}, err
	}
	if c.equal() {
		w.version = latest.ContractVersion
		w.upgradableFrom = slices.Clone(latest.UpgradableFrom)
	} else {
		w.version = developmentAfter(latest.Version)
	}
	return w, c, nil
}

// isPatchOf reports whether the next release of the line v, a version or
// X.Y, is a patch release of latest: latest is a final release, without
// pre-release, of the same X.Y. Only then must the next release keep the
// contract; a pre-release or a new minor release may change it.
func isPatchOf(latest, v string) bool {
	l, err := semver.Parse(latest)
	if err != nil || len(l.Pre) > 0 {
		return false
	}
	major, minor, ok := lineOf(v)
	return ok && l.Major == major && l.Minor == minor
}

// lineOf returns the X.Y of v, a version or X.Y.
func lineOf(v string) (uint64, uint64, bool) {
	if m := releaseLine.FindStringSubmatch(v); m != nil {
		major, errMajor := strconv.ParseUint(m[1], 10, 64)
		minor, errMinor := strconv.ParseUint(m[2], 10, 64)
		return major, minor, errMajor == nil && errMinor == nil
	}
	parsed, err := semver.Parse(v)
	if err != nil {
		return 0, 0, false
	}
	return parsed.Major, parsed.Minor, true
}

// lineOfBranch returns the X.Y of a release branch release-X.Y, or false
// for any other branch.
func lineOfBranch(branch string) (string, bool) {
	m := releaseBranchName.FindStringSubmatch(branch)
	if m == nil {
		return "", false
	}
	return m[1] + "." + m[2], true
}

// checkRelease checks that the next release of the line v, a version or
// X.Y, keeps the contract when it is a patch release, and returns what to
// tell the user and the comparison it made. A patch release of a final
// release must not change what core and extensions share; anything else
// may, so it passes with a notice. The latest record must have
// fingerprints under scheme, as for decide, so blanking them cannot turn
// the check off; only the seed has none, and it is a pre-release.
//
// When v is the latest release recorded here, the release workflow is
// about to tag the commit that recorded it: then the contract must be the
// one its record holds, whatever release it is.
func checkRelease(releases []release, fps map[string]string, v string) (string, comparison, error) {
	latest := releases[len(releases)-1]
	var c comparison
	if !latest.isSeed() {
		var err error
		if c, err = compareWith(latest, fps); err != nil {
			return "", comparison{}, err
		}
	}
	if v == latest.Version && !latest.isSeed() {
		if !c.equal() {
			return "", c, fmt.Errorf("%s, so this tree is not the one release %s was recorded from", c.summary(), v)
		}
		return c.summary(), c, nil
	}
	if !isPatchOf(latest.Version, v) {
		next := "release " + v
		if releaseLine.MatchString(v) {
			next = "the next " + v + " release"
		}
		return fmt.Sprintf("%s is not a patch release of %s, the latest release recorded here, "+
			"so it may change the contract: only a patch release of a final release must keep it", next, latest.Version,
		), c, nil
	}
	if !c.equal() {
		return "", c, fmt.Errorf("%s. A patch release must not change what core and extensions share: "+
			"revert the change, or ship it in a minor release. make contract-diff shows the change",
			c.summary(),
		)
	}
	return c.summary(), c, nil
}

// developmentAfter returns the development value that follows the release
// v, which sorts after v and before every release main can build next, so
// nobody has to guess the name of the next release:
//
//   - v.next after a pre-release, since SemVer sorts nothing between two
//     pre-releases but v with more identifiers;
//   - X.(Y+1).0-alpha.0 after a release X.Y.Z, since patch releases come
//     from release branches and main builds the next minor release. It
//     sorts above every X.Y patch release and below X.(Y+1).0-alpha.1.
func developmentAfter(v string) string {
	parsed := semver.MustParse(v)
	if len(parsed.Pre) > 0 {
		return v + ".next"
	}
	return fmt.Sprintf("%d.%d.0-alpha.0", parsed.Major, parsed.Minor+1)
}

// compare compares two valid SemVer 2.0.0 versions by precedence.
func compare(a, b string) int {
	return semver.MustParse(a).Compare(semver.MustParse(b))
}

// isDevelopment reports whether v is a development value: its last
// pre-release identifier is next, or its pre-release is exactly alpha.0.
// The contract package has the same rule; this command must not import
// that package, which it generates.
func isDevelopment(v semver.Version) bool {
	if len(v.Pre) == 0 {
		return false
	}
	if last := v.Pre[len(v.Pre)-1]; !last.IsNum && last.VersionStr == "next" {
		return true
	}
	return len(v.Pre) == 2 &&
		!v.Pre[0].IsNum && v.Pre[0].VersionStr == "alpha" &&
		v.Pre[1].IsNum && v.Pre[1].VersionNum == 0
}
