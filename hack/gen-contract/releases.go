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
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/blang/semver/v4"
	"sigs.k8s.io/yaml"
)

// releasesDir holds one record per core release, named <version>.yaml.
const releasesDir = "contract/releases"

// Types of a release.
const (
	// typeFreshInstall is a release that can only be installed fresh.
	typeFreshInstall = "fresh-install"
	// typeUpgradable is a release that can be upgraded to in place from the
	// releases its record lists.
	typeUpgradable = "upgradable"
)

// A release is the record of a core release: what the contract was when it
// was cut.
type release struct {
	// Version is the core release.
	Version string `json:"version"`
	// ContractVersion is the contract version of the release: the last
	// release, up to this one, that changed the contract.
	ContractVersion string `json:"contractVersion"`
	// MinContractVersion is the oldest contract version the release serves.
	MinContractVersion string `json:"minContractVersion"`
	// Type is typeFreshInstall or typeUpgradable.
	Type string `json:"type"`
	// UpgradableFrom lists the releases an upgradable release can be
	// upgraded from.
	UpgradableFrom []string `json:"upgradableFrom"`
	// Fingerprint is the fingerprint of the contract the release was cut
	// with, empty for a release that predates fingerprinting.
	Fingerprint string `json:"fingerprint"`
}

var fingerprintPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

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
		var r release
		if err := yaml.UnmarshalStrict(data, &r); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if err := r.validate(); err != nil {
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
		prev, r := releases[i-1], releases[i]
		if compare(r.ContractVersion, prev.ContractVersion) < 0 || compare(r.MinContractVersion, prev.MinContractVersion) < 0 {
			return nil, fmt.Errorf("release %s: its contract window [%s, %s] goes below the window [%s, %s] of the earlier release %s",
				r.Version, r.MinContractVersion, r.ContractVersion, prev.MinContractVersion, prev.ContractVersion, prev.Version,
			)
		}
	}
	return releases, nil
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
	errs = append(errs, r.validateUpgrades())
	if r.Fingerprint != "" && !fingerprintPattern.MatchString(r.Fingerprint) {
		errs = append(errs, fmt.Errorf("fingerprint %q is neither empty nor sha256:<64 lowercase hex digits>", r.Fingerprint))
	}
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
	fingerprint    string
	upgradableFrom []string
}

// decide works out the window of a core whose contract has the fingerprint
// fp, from the release records sorted by version. While the contract is
// the one the latest release was cut with, the core has that release's
// contract version and can be upgraded from what it can. Once it differs,
// the core has the development value after the latest release, which no
// release can be upgraded to. The minimum moves only at a release.
func decide(releases []release, fp string) window {
	latest := releases[len(releases)-1]
	w := window{minVersion: latest.MinContractVersion, fingerprint: fp, upgradableFrom: []string{}}
	if latest.Fingerprint != "" && latest.Fingerprint == fp {
		w.version = latest.ContractVersion
		w.upgradableFrom = slices.Clone(latest.UpgradableFrom)
	} else {
		w.version = developmentAfter(latest.Version)
	}
	return w
}

// checkRelease returns an error unless the contract with the fingerprint fp
// is the one the latest release was cut with. Release branches check it,
// so a patch release cannot change the contract. A release that predates
// fingerprinting has nothing to compare with, so it passes; the first
// fingerprinted release closes that gap.
func checkRelease(releases []release, fp string) error {
	latest := releases[len(releases)-1]
	switch latest.Fingerprint {
	case fp, "":
		return nil
	default:
		return fmt.Errorf("the contract changed since release %s (fingerprint %s, recorded %s): "+
			"a patch release must not change what core and extensions share; "+
			"revert the change, or ship it in a minor release",
			latest.Version, fp, latest.Fingerprint,
		)
	}
}

// developmentAfter returns the development value that follows the release
// v: v.dev after a pre-release, X.Y.(Z+1)-dev after a release X.Y.Z. It
// sorts after v and before every release that can follow it, so nobody has
// to guess the name of the next release.
func developmentAfter(v string) string {
	parsed := semver.MustParse(v)
	if len(parsed.Pre) > 0 {
		return v + ".dev"
	}
	return fmt.Sprintf("%d.%d.%d-dev", parsed.Major, parsed.Minor, parsed.Patch+1)
}

// compare compares two valid SemVer 2.0.0 versions by precedence.
func compare(a, b string) int {
	return semver.MustParse(a).Compare(semver.MustParse(b))
}

// isDevelopment reports whether v has a "dev" pre-release identifier, the
// mark of a development value. The contract package has the same rule;
// this command must not import that package, which it generates.
func isDevelopment(v semver.Version) bool {
	return slices.ContainsFunc(v.Pre, func(id semver.PRVersion) bool {
		return !id.IsNum && id.VersionStr == "dev"
	})
}
