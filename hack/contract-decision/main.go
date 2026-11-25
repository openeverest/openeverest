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

// Command contract-decision checks that a pull request which touches code
// next to the contract says whether it changes the contract.
//
// The contract version moves by itself when a contract artifact changes,
// but code can impose a new requirement on extensions without changing any
// artifact: middleware that sends every Instance write from the UI through
// each provider's webhook as a dry run changes no CRD and no API schema,
// yet breaks every provider whose webhook has side effects. So while the
// contract is unchanged since the last release, a pull request that
// touches a contract-adjacent path, as listed between
// "# BEGIN contract-adjacent" and "# END contract-adjacent" in
// .github/CODEOWNERS, needs exactly one of the labels contract-change and
// no-contract-change, and the label must agree with contract/VERSION:
// contract-change needs the development value there, which recording the
// change in contract/CHANGES produces when no artifact shows it, and
// no-contract-change needs the release value still. Markdown files never
// count as touched: documentation is not part of the contract.
//
// Only a v1 branch predates the contract. A v2 branch without
// contract/VERSION or contract/releases fails, rather than skip the
// decision, unless the pull request is the one that brings them to it.
//
// It imports only the standard library, so CI runs it without downloading
// the module's dependencies.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Labels of the decision.
const (
	labelChange   = "contract-change"
	labelNoChange = "no-contract-change"
)

// Delimiters of the contract-adjacent block of CODEOWNERS.
const (
	blockBegin = "# BEGIN contract-adjacent"
	blockEnd   = "# END contract-adjacent"
)

// maxListed is how many of the contract-adjacent paths a pull request
// touches a message lists.
const maxListed = 10

type labels []string

func (l *labels) String() string { return strings.Join(*l, ",") }

func (l *labels) Set(v string) error {
	*l = append(*l, v)
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Getenv("GITHUB_ACTIONS") == "true"))
}

// run checks the decision of the pull request args describe and returns
// the exit code: 0 when it passes, 1 when it fails, 2 on bad arguments.
func run(args []string, stdin io.Reader, stdout io.Writer, actions bool) int {
	fs := flag.NewFlagSet("contract-decision", flag.ContinueOnError)
	fs.SetOutput(stdout)
	codeowners := fs.String("codeowners", ".github/CODEOWNERS", "CODEOWNERS file holding the contract-adjacent block")
	baseRef := fs.String("base-ref", "", "the base branch, such as main or release-2.0")
	baseVersion := fs.String("base-version", "", "content of contract/VERSION on the base branch; empty when it has none")
	baseRecords := fs.Bool("base-records", false, "whether the base branch has contract/releases")
	headVersion := fs.String("head-version", "", "content of contract/VERSION in the pull request; empty when it has none")
	headRecords := fs.Bool("head-records", false, "whether the pull request has contract/releases")
	files := fs.String("files", "-", "file listing the paths the pull request changes, one per line; - for standard input")
	allAdjacent := fs.Bool("all-adjacent", false, "treat the pull request as touching contract-adjacent paths, when its paths cannot be listed")
	var prLabels labels
	fs.Var(&prLabels, "label", "a label of the pull request (repeatable)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	report := func(level, msg string) {
		// Under GitHub Actions, a workflow command shows the message on the
		// pull request.
		if actions {
			fmt.Fprintf(stdout, "::%s title=Contract decision::%s\n", level, msg)
			return
		}
		fmt.Fprintf(stdout, "%s: %s\n", level, msg)
	}
	in := input{
		baseRef:     strings.TrimPrefix(strings.TrimSpace(*baseRef), "refs/heads/"),
		baseVersion: strings.TrimSpace(*baseVersion),
		baseRecords: *baseRecords,
		headVersion: strings.TrimSpace(*headVersion),
		headRecords: *headRecords,
		labels:      prLabels,
		allAdjacent: *allAdjacent,
	}
	var err error
	if in.patterns, err = readBlock(*codeowners); err != nil {
		report("error", err.Error())
		return 1
	}
	if !in.allAdjacent {
		if in.files, err = readFiles(*files, stdin); err != nil {
			report("error", err.Error())
			return 1
		}
	}
	d := decide(in)
	for _, n := range d.notices {
		report("notice", n)
	}
	if d.failure != "" {
		report("error", d.failure)
		return 1
	}
	return 0
}

// readBlock returns the patterns of the contract-adjacent block of a
// CODEOWNERS file.
func readBlock(file string) ([]string, error) {
	f, err := os.Open(file) //nolint:gosec // the CODEOWNERS file named on the command line
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read only
	return parseBlock(f)
}

func parseBlock(r io.Reader) ([]string, error) {
	var patterns []string
	inside, closed := false, false
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		switch {
		case line == blockBegin:
			if inside || closed {
				return nil, fmt.Errorf("CODEOWNERS: more than one %q", blockBegin)
			}
			inside = true
		case line == blockEnd:
			if !inside {
				return nil, fmt.Errorf("CODEOWNERS: %q before %q", blockEnd, blockBegin)
			}
			inside, closed = false, true
		case inside && line != "" && !strings.HasPrefix(line, "#"):
			patterns = append(patterns, strings.Fields(line)[0])
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	switch {
	case inside:
		return nil, fmt.Errorf("CODEOWNERS: %q without %q", blockBegin, blockEnd)
	case !closed:
		return nil, fmt.Errorf("CODEOWNERS: no %q block", blockBegin)
	case len(patterns) == 0:
		return nil, errors.New("CODEOWNERS: the contract-adjacent block lists no path")
	}
	return patterns, nil
}

// readFiles returns the paths listed in a file, one per line, or in stdin
// for "-".
func readFiles(name string, stdin io.Reader) ([]string, error) {
	r := stdin
	if name != "-" {
		f, err := os.Open(name) //nolint:gosec // the file named on the command line
		if err != nil {
			return nil, err
		}
		defer f.Close() //nolint:errcheck // read only
		r = f
	}
	var files []string
	s := bufio.NewScanner(r)
	for s.Scan() {
		if line := strings.TrimSpace(s.Text()); line != "" {
			files = append(files, line)
		}
	}
	return files, s.Err()
}

// matcher returns a regular expression matching the paths a CODEOWNERS
// pattern matches, by the gitignore rules CODEOWNERS follows: a pattern
// with a slash before its end is anchored at the root, and one without
// matches at any depth; a trailing slash matches a directory, so every
// path below it, while a pattern without one also matches a file; * and ?
// match within a path segment, ** across segments.
func matcher(pattern string) (*regexp.Regexp, error) {
	dir := strings.HasSuffix(pattern, "/")
	p := strings.TrimSuffix(pattern, "/")
	anchored := strings.Contains(p, "/")
	p = strings.TrimPrefix(p, "/")
	if p == "" || p == "*" || p == "**" {
		return nil, fmt.Errorf("CODEOWNERS: contract-adjacent pattern %q matches every path", pattern)
	}
	var re strings.Builder
	if anchored {
		re.WriteString("^")
	} else {
		re.WriteString("^(.*/)?")
	}
	for i := 0; i < len(p); i++ {
		switch {
		case strings.HasPrefix(p[i:], "**/"):
			re.WriteString("(.*/)?")
			i += 2
		case strings.HasPrefix(p[i:], "**"):
			re.WriteString(".*")
			i++
		case p[i] == '*':
			re.WriteString("[^/]*")
		case p[i] == '?':
			re.WriteString("[^/]")
		default:
			re.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
	}
	if dir {
		re.WriteString("/.*$")
	} else {
		re.WriteString("(/.*)?$")
	}
	return regexp.Compile(re.String())
}

type input struct {
	// baseRef is the base branch.
	baseRef string
	// baseVersion is contract/VERSION on the base branch, empty when the
	// base branch has none.
	baseVersion string
	// baseRecords is whether the base branch has contract/releases.
	baseRecords bool
	// headVersion is contract/VERSION in the pull request, empty when it
	// has none.
	headVersion string
	// headRecords is whether the pull request has contract/releases.
	headRecords bool
	// patterns are the contract-adjacent patterns.
	patterns []string
	// files are the paths the pull request changes.
	files []string
	// allAdjacent treats every path as contract-adjacent.
	allAdjacent bool
	labels      []string
}

type decision struct {
	notices []string
	// failure says what the pull request must do; empty when it passes.
	failure string
}

// whereTheContractIs decides a pull request whose base branch is of v1,
// which predates the contract, or where either side lacks the contract,
// and returns false for any other.
func whereTheContractIs(in input) (decision, bool) {
	if isV1Branch(in.baseRef) {
		return decision{notices: []string{fmt.Sprintf("%s is a v1 branch, which predates the contract: there is nothing to decide.", in.baseRef)}}, true
	}
	baseMissing := missingContract(in.baseVersion, in.baseRecords)
	headMissing := missingContract(in.headVersion, in.headRecords)
	switch {
	case len(baseMissing) > 0 && len(headMissing) == 0:
		return decision{notices: []string{fmt.Sprintf(
			"The base branch %s has no %s, and this pull request adds %s: it brings the contract to the branch, so there is nothing to decide.",
			branchName(in.baseRef), strings.Join(baseMissing, " or "), plural(len(baseMissing), "it", "them"),
		)}}, true
	case len(baseMissing) > 0:
		return decision{failure: fmt.Sprintf(
			"The base branch %s is a v2 branch, which carries the contract, but it has no %s. %s. See contract/MAINTAINING.md.",
			branchName(in.baseRef), strings.Join(baseMissing, " or "), fixMissingContract(in.baseRef, len(baseMissing)),
		)}, true
	case len(headMissing) > 0:
		return decision{failure: fmt.Sprintf(
			"This pull request removes %s, which every v2 branch carries, from %s: restore %s.",
			strings.Join(headMissing, " and "), branchName(in.baseRef), plural(len(headMissing), "it", "them"),
		)}, true
	}
	return decision{}, false
}

// decide checks the decision of a pull request.
func decide(in input) decision {
	if d, ok := whereTheContractIs(in); ok {
		return d
	}
	if isDevelopment(in.baseVersion) {
		return decision{notices: []string{fmt.Sprintf(
			"The contract has changed since the last release already (contract/VERSION is %s on the base branch), so no decision is needed.",
			in.baseVersion,
		)}}
	}

	adjacent, docs, err := adjacentFiles(in)
	if err != nil {
		return decision{failure: err.Error()}
	}
	switch {
	case len(adjacent) == 0 && len(docs) > 0:
		return decision{notices: []string{fmt.Sprintf(
			"This pull request touches no contract-adjacent path but Markdown files (%s), which are never part of the contract.",
			describeFiles(docs),
		)}}
	case len(adjacent) == 0:
		return decision{notices: []string{"This pull request touches no contract-adjacent path."}}
	}

	var chosen []string
	for _, l := range []string{labelChange, labelNoChange} {
		if slices.Contains(in.labels, l) {
			chosen = append(chosen, l)
		}
	}
	touched := describeFiles(adjacent)
	switch {
	case len(chosen) != 1:
		return decision{failure: fmt.Sprintf(
			"This pull request touches contract-adjacent paths (%s), and the contract is unchanged since the last release "+
				"(contract/VERSION is %s on the base branch). Add exactly one label: %s if an extension built against the "+
				"base branch could break or need to change, even though no CRD or API schema changes; %s otherwise. "+
				"See contract/MAINTAINING.md.",
			touched, in.baseVersion, labelChange, labelNoChange,
		)}
	case chosen[0] == labelChange && !isDevelopment(in.headVersion):
		return decision{failure: fmt.Sprintf(
			"This pull request is labelled %s, but contract/VERSION is still %s. Record the change in contract/CHANGES, "+
				"then run make gen-contract, which moves contract/VERSION to the development value.",
			labelChange, in.headVersion,
		)}
	case chosen[0] == labelNoChange && isDevelopment(in.headVersion):
		return decision{failure: fmt.Sprintf(
			"This pull request is labelled %s, but it changes the contract: a contract artifact or contract/CHANGES "+
				"changed, so contract/VERSION moves to %s. Label it %s instead.",
			labelNoChange, in.headVersion, labelChange,
		)}
	}
	return decision{notices: []string{fmt.Sprintf("Labelled %s; touches %s.", chosen[0], touched)}}
}

// adjacentFiles returns the paths of in that match a contract-adjacent
// pattern, and apart from them the Markdown files that do. Documentation
// is never part of the contract, so a Markdown file never counts as
// touched: gen-contract's tests fail when a fingerprint input, or what a
// contract-adjacent package embeds, could match one.
func adjacentFiles(in input) ([]string, []string, error) {
	if in.allAdjacent {
		return []string{"every path, since they could not be listed"}, nil, nil
	}
	matchers := make([]*regexp.Regexp, 0, len(in.patterns))
	for _, p := range in.patterns {
		m, err := matcher(p)
		if err != nil {
			return nil, nil, err
		}
		matchers = append(matchers, m)
	}
	var adjacent, docs []string
	for _, f := range in.files {
		switch {
		case !slices.ContainsFunc(matchers, func(m *regexp.Regexp) bool { return m.MatchString(f) }):
		case isMarkdown(f):
			docs = append(docs, f)
		default:
			adjacent = append(adjacent, f)
		}
	}
	return adjacent, docs, nil
}

// isMarkdown reports whether path names a Markdown file.
func isMarkdown(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".md")
}

func describeFiles(files []string) string {
	if len(files) <= maxListed {
		return strings.Join(files, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(files[:maxListed], ", "), len(files)-maxListed)
}

// v1Branch matches the branches of v1, which predate the contract: v1.x
// and its release branches release-1.Y, and those of earlier majors.
var v1Branch = regexp.MustCompile(`^(?:v([0-9]+)\.x|release-([0-9]+)\..*)$`)

// isV1Branch reports whether branch is a v1 branch. Every other branch,
// main, a v2 release branch or one this cannot tell, is v2, and must carry
// the contract.
func isV1Branch(branch string) bool {
	m := v1Branch.FindStringSubmatch(branch)
	if m == nil {
		return false
	}
	major, err := strconv.Atoi(m[1] + m[2])
	return err == nil && major < 2
}

// missingContract returns the paths of the contract a tree lacks, given
// its contract/VERSION and whether it has contract/releases.
func missingContract(version string, records bool) []string {
	var missing []string
	if version == "" {
		missing = append(missing, "contract/VERSION")
	}
	if !records {
		missing = append(missing, "contract/releases")
	}
	return missing
}

// fixMissingContract says how the v2 branch gets the n paths of the
// contract it lacks.
func fixMissingContract(branch string, n int) string {
	if branch == "" || branch == "main" {
		return fmt.Sprintf("Restore %s on %s", plural(n, "it", "them"), branchName(branch))
	}
	return fmt.Sprintf("Merge main into %s when preparing its next pre-release, "+
		"which brings the contract to a release branch cut before it", branch,
	)
}

// branchName returns branch, or a phrase for an unknown one.
func branchName(branch string) string {
	if branch == "" {
		return "(unnamed)"
	}
	return branch
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// isDevelopment reports whether the valid version v is a development value,
// as contract.IsDevelopment does: its last pre-release identifier is next,
// or its pre-release is exactly alpha.0.
func isDevelopment(v string) bool {
	v, _, _ = strings.Cut(v, "+")
	_, pre, ok := strings.Cut(v, "-")
	return ok && (pre == "next" || strings.HasSuffix(pre, ".next") || pre == "alpha.0")
}
