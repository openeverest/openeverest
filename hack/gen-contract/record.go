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
	"slices"
	"strings"
)

// A releaseRequest is what the person who cuts a core release decides
// about it: its version, and whether it can be upgraded to in place.
type releaseRequest struct {
	version string
	typ     string
	// minVersion and upgradableFrom are for an upgradable release only.
	minVersion     string
	upgradableFrom []string
	// fromRecords are the records of the releases in upgradableFrom, read
	// from their tags; when nil, they are read from the records here.
	fromRecords map[string]release
}

// runRelease records a core release in the release records of the
// repository at root, from the arguments of the release command.
func runRelease(root string, args []string) (string, error) {
	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	version := fs.String("version", "", "the core release to record, such as 2.1.0 or 2.1.0-rc.1, without a v")
	typ := fs.String("type", "", typeFreshInstall+" or "+typeUpgradable)
	minVersion := fs.String("min", "", "for an "+typeUpgradable+" release: the oldest contract version it serves")
	from := fs.String("upgradable-from", "", "for an "+typeUpgradable+" release: the core releases it can be upgraded from in place, comma-separated")
	tagRepo := fs.String("tag-repo", "", "a git repository holding the tags of the releases in -upgradable-from, to read their records from those tags")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() > 0 {
		return "", fmt.Errorf("release: unexpected argument %q", fs.Arg(0))
	}
	req := releaseRequest{version: *version, typ: *typ, minVersion: *minVersion}
	if *from != "" {
		for v := range strings.SplitSeq(*from, ",") {
			v = strings.TrimSpace(v)
			if v == "" {
				return "", fmt.Errorf("-upgradable-from %q has an empty entry", *from)
			}
			req.upgradableFrom = append(req.upgradableFrom, v)
		}
	}

	releases, fps, err := load(root)
	if err != nil {
		return "", err
	}
	if *tagRepo != "" {
		if req.fromRecords, err = tagRecords(*tagRepo, req.upgradableFrom, releases); err != nil {
			return "", err
		}
	}
	r, err := newRelease(releases, fps, req)
	if err != nil {
		return "", err
	}
	file := releasesDir + "/" + r.Version + ".yaml"
	if err := writeNew(filepath.Join(root, filepath.FromSlash(file)), marshalRelease(r)); err != nil {
		return "", err
	}
	msg := fmt.Sprintf("recorded release %s in %s: %s, serving contract versions [%s, %s]",
		r.Version, file, r.Type, r.MinContractVersion, r.ContractVersion,
	)
	if len(r.UpgradableFrom) > 0 {
		msg += ", upgradable from " + strings.Join(r.UpgradableFrom, ", ")
	}
	return msg + "; run make gen-contract to publish it", nil
}

// tagRecords returns the records of the releases versions, read from their
// tags in the git repository repo, which must hold them. A release cut on
// another branch since this one forked has no record here until a pull
// request brings it, but its tag carries it. The seed, which no tag
// carries, is read from records, the records here.
func tagRecords(repo string, versions []string, records []release) (map[string]release, error) {
	out := make(map[string]release, len(versions))
	var errs []error
	for _, v := range versions {
		if releaseVersion(v) != nil {
			continue
		}
		if v == seedVersion {
			if i := slices.IndexFunc(records, release.isSeed); i >= 0 {
				out[v] = records[i]
			}
			continue
		}
		tag := "v" + v
		if !hasRef(repo, tag) {
			errs = append(errs, fmt.Errorf("-upgradable-from: %w", requireTag(repo, tag)))
			continue
		}
		data, ok, err := showFile(repo, tag, recordPath(v))
		switch {
		case err != nil:
			return nil, err
		case !ok:
			errs = append(errs, fmt.Errorf("-upgradable-from: %s carries no record of release %s", tag, v))
			continue
		}
		r, err := decodeRecord(data)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("-upgradable-from: the record %s carries: %w", tag, err))
		case r.Version != v:
			errs = append(errs, fmt.Errorf("-upgradable-from: the record %s carries is of release %s", tag, r.Version))
		default:
			out[v] = r
		}
	}
	return out, errors.Join(errs...)
}

// newRelease returns the record of the release req describes, cut from a
// core whose contract has the fingerprints fps and which holds the records
// releases, sorted by version.
func newRelease(releases []release, fps map[string]string, req releaseRequest) (release, error) {
	if err := releaseVersion(req.version); err != nil {
		return release{}, fmt.Errorf("-version: %w", err)
	}
	latest := releases[len(releases)-1]
	recorded := make(map[string]release, len(releases))
	for _, r := range releases {
		recorded[r.Version] = r
	}
	if _, ok := recorded[req.version]; ok {
		return release{}, fmt.Errorf("release %s is recorded already, in %s/%s.yaml", req.version, releasesDir, req.version)
	}
	// Each record is worked out from the latest one, so a branch records
	// its releases in order.
	if compare(req.version, latest.Version) <= 0 {
		return release{}, fmt.Errorf("release %s is not newer than %s, the latest release recorded here", req.version, latest.Version)
	}

	contractVersion, err := releaseContractVersion(latest, fps, req.version)
	if err != nil {
		return release{}, err
	}
	r := release{
		APIVersion:      recordAPIVersion,
		Kind:            recordKind,
		Version:         req.version,
		ContractVersion: contractVersion,
		Type:            req.typ,
		UpgradableFrom:  []string{},
		Fingerprints:    map[string]map[string]string{scheme: fps},
	}
	switch req.typ {
	case typeFreshInstall:
		if req.minVersion != "" || len(req.upgradableFrom) > 0 {
			return release{}, fmt.Errorf("-min and -upgradable-from are for an %s release only: "+
				"a %s release serves only the extensions built for it, and nothing can be upgraded to it",
				typeUpgradable, typeFreshInstall,
			)
		}
		// It serves only the extensions built for it, which are stamped
		// with its contract version.
		r.MinContractVersion = r.ContractVersion
	case typeUpgradable:
		if err := checkUpgradable(recorded, latest, req); err != nil {
			return release{}, err
		}
		r.MinContractVersion = req.minVersion
		r.UpgradableFrom = req.upgradableFrom
	case "":
		return release{}, fmt.Errorf("-type is required: %s or %s", typeFreshInstall, typeUpgradable)
	default:
		return release{}, fmt.Errorf("-type %q is neither %s nor %s", req.typ, typeFreshInstall, typeUpgradable)
	}
	if err := r.validate(); err != nil {
		return release{}, err
	}
	if err := checkContractOrder(latest, r); err != nil {
		return release{}, err
	}
	return r, nil
}

// releaseContractVersion returns the contract version of the release v,
// cut from a core whose contract has the fingerprints fps, after latest.
//
// A patch release of a final release keeps the contract version of its
// line, so extensions built against it run on every release of the line;
// it must keep the contract for that, which the release branch checks on
// every pull request and which is checked again here. Any other release, a
// pre-release or a new minor or major release, is the contract version of
// its own contract, even when the fingerprints are those of latest: equal
// fingerprints only show that no artifact changed and that nobody
// recorded a change in CHANGES, not that extensions built against the new
// release run on the one before. That becomes evidence once a job runs the
// extensions built against each candidate on the release before it and
// records every break it finds in CHANGES; then a release whose contract
// is unchanged could keep the contract version of the one before too.
func releaseContractVersion(latest release, fps map[string]string, v string) (string, error) {
	if !isPatchOf(latest.Version, v) {
		return v, nil
	}
	c, err := compareWith(latest, fps)
	if err != nil {
		return "", err
	}
	if !c.equal() {
		return "", fmt.Errorf("release %s is a patch release of %s, so it must keep its contract, but %s. "+
			"Revert the change, or ship it in a minor release; make contract-diff shows it",
			v, latest.Version, c.summary(),
		)
	}
	return latest.ContractVersion, nil
}

// checkUpgradable checks the window of an upgradable release against the
// releases recorded before it, the latest of which is latest.
func checkUpgradable(recorded map[string]release, latest release, req releaseRequest) error {
	var errs []error
	if req.minVersion == "" {
		errs = append(errs, fmt.Errorf("-min is required for an %s release: the oldest contract version it serves", typeUpgradable))
	} else if err := releaseVersion(req.minVersion); err != nil {
		errs = append(errs, fmt.Errorf("-min: %w", err))
	}
	if len(req.upgradableFrom) == 0 {
		errs = append(errs, fmt.Errorf("-upgradable-from is required for an %s release: the releases it can be upgraded from in place", typeUpgradable))
	}
	contractOf := func(v string) (string, bool) {
		r, ok := recorded[v]
		if req.fromRecords != nil && v != latest.Version {
			r, ok = req.fromRecords[v]
		}
		return r.ContractVersion, ok
	}
	for _, from := range req.upgradableFrom {
		if err := releaseVersion(from); err != nil {
			errs = append(errs, fmt.Errorf("-upgradable-from: %w", err))
		} else if _, ok := contractOf(from); !ok {
			errs = append(errs, fmt.Errorf("-upgradable-from: release %s has no record in %s; "+
				"read the records of releases cut on another branch from their tags with -tag-repo",
				from, releasesDir,
			))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// Users upgrade core first and their extensions after, so an
	// upgradable release must serve the extensions built for each release
	// it is upgraded from, and those built for the latest release, which
	// are the ones published when it ships.
	for _, from := range slices.Concat([]string{latest.Version}, req.upgradableFrom) {
		if c, _ := contractOf(from); compare(req.minVersion, c) > 0 {
			return fmt.Errorf("-min %s is above %s, the contract version of release %s: "+
				"extensions built for %s would stop working as soon as core is upgraded, before users can update them. "+
				"An %s release serves the extensions of the releases before it; lower -min, or cut a %s release",
				req.minVersion, c, from, from, typeUpgradable, typeFreshInstall,
			)
		}
	}
	return nil
}

// marshalRelease returns the content of the record of r, a valid release,
// in the layout of the first record: the header, then each field with a
// comment saying whether the generator computed it or the person who cut
// the release decided it. Every value is a SemVer version or a digest,
// which YAML reads as the string it is.
func marshalRelease(r release) []byte {
	const computed, decided = "computed", "decided"
	from := "[" + strings.Join(r.UpgradableFrom, ", ") + "]"
	fields := []struct{ key, value, origin string }{
		{"version", r.Version, computed},
		{"contractVersion", r.ContractVersion, computed},
		{"minContractVersion", r.MinContractVersion, decided},
		{"type", r.Type, decided},
		{"upgradableFrom", from, decided},
	}
	// The comments line up as in the first record, or further right.
	width := 29
	for _, f := range fields {
		width = max(width, len(f.key)+2+len(f.value))
	}
	var b strings.Builder
	b.WriteString("# OpenEverest release record: data for hack/gen-contract, not a Kubernetes resource.\n")
	fmt.Fprintf(&b, "apiVersion: %s\nkind: %s\n", recordAPIVersion, recordKind)
	for _, f := range fields {
		line := f.key + ": " + f.value
		fmt.Fprintf(&b, "%-*s  # %s\n", width, line, f.origin)
	}
	fmt.Fprintf(&b, "%-*s  # computed; add-only; key = fingerprint scheme\n", width, "fingerprints:")
	for _, s := range slices.Sorted(maps.Keys(r.Fingerprints)) {
		fmt.Fprintf(&b, "  %s:\n", s)
		for _, name := range sortedInputs(r.Fingerprints[s]) {
			fmt.Fprintf(&b, "    %s: %s\n", name, r.Fingerprints[s][name])
		}
	}
	return []byte(b.String())
}

// writeNew writes data to a file that must not exist yet.
func writeNew(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // a release record of this repository
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return errors.Join(err, f.Close())
}
