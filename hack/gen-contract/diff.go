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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
)

// defaultMaxLines is how many lines the diff command prints per input.
const defaultMaxLines = 200

// runDiff prints, for each input, a unified diff of the canonical form of
// the tree of a ref and that of the working tree: what a digest saw
// change, which the digest alone cannot say. With -raw it prints the exact
// canonical streams the digests hash instead.
func runDiff(root string, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("diff", flag.ContinueOnError)
	ref := flags.String("ref", "", "the ref to compare the working tree with (default: the tag of the latest release); "+
		"with -raw, the ref whose streams to print (default: the working tree)")
	only := flags.String("input", "", "only this input: "+strings.Join(inputNames(), ", "))
	raw := flags.Bool("raw", false, "print the canonical streams the digests hash, byte for byte, instead of a diff")
	maxLines := flags.Int("max-lines", defaultMaxLines, "lines of diff to print per input; 0 prints them all")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("diff: unexpected argument %q", flags.Arg(0))
	}
	names := inputNames()
	if *only != "" {
		if !slices.Contains(names, *only) {
			return fmt.Errorf("-input %q is not an input: %s", *only, strings.Join(names, ", "))
		}
		names = []string{*only}
	}
	if *raw {
		return printStreams(root, *ref, names, *only != "", stdout)
	}
	if *ref == "" {
		releases, err := readReleases(filepath.Join(root, filepath.FromSlash(releasesDir)))
		if err != nil {
			return err
		}
		*ref = "v" + releases[len(releases)-1].Version
	}
	d, err := diffInputs(root, *ref, names, *maxLines)
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdout, d)
	return err
}

// printStreams prints the canonical streams of the inputs names in the
// tree of ref, or in the working tree when ref is empty. A single input is
// printed alone, so its SHA-256 is its digest; several are each preceded
// by a line naming them, which is not part of the stream.
func printStreams(root, ref string, names []string, single bool, w io.Writer) error {
	dir, where := root, "the working tree"
	if ref != "" {
		if !hasRef(root, ref) {
			return requireTag(root, ref)
		}
		extracted, cleanup, err := extractInputs(root, ref)
		if err != nil {
			return err
		}
		defer cleanup()
		dir, where = extracted, ref
	}
	s, err := streams(dir)
	if err != nil {
		return err
	}
	for _, name := range names {
		data, ok := s[name]
		var err error
		switch {
		case single && !ok:
			return fmt.Errorf("%s is absent from %s", name, where)
		case single:
			_, err = w.Write(data)
		case !ok:
			_, err = fmt.Fprintf(w, "### %s: absent\n", name)
		default:
			_, err = fmt.Fprintf(w, "### %s: %s, %d bytes\n%s", name, digest(data), len(data), data)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// diffInputs returns, for each input of names, whether its digest in the
// tree of ref and in the working tree differ, and if so a unified diff of
// their canonical forms, indented for reading, capped at maxLines lines
// unless it is 0.
func diffInputs(root, ref string, names []string, maxLines int) (string, error) {
	if !hasRef(root, ref) {
		return "", requireTag(root, ref)
	}
	dir, cleanup, err := extractInputs(root, ref)
	if err != nil {
		return "", err
	}
	defer cleanup()

	var b strings.Builder
	for _, in := range inputs {
		if !slices.Contains(names, in.name) {
			continue
		}
		before, err := in.entries(dir)
		if err != nil {
			return "", fmt.Errorf("%s: %w", ref, err)
		}
		after, err := in.entries(root)
		if err != nil {
			return "", err
		}
		was, is := describeDigest(before), describeDigest(after)
		if was == is {
			fmt.Fprintf(&b, "=== %s: unchanged since %s (%s)\n", in.name, ref, is)
			continue
		}
		fmt.Fprintf(&b, "=== %s: %s at %s, %s in the working tree\n", in.name, was, ref, is)
		lines := unifiedDiff(diffLines(prettyLines(before), prettyLines(after)), in.name+" ("+ref+")", in.name+" (working tree)")
		if maxLines > 0 && len(lines) > maxLines {
			more := len(lines) - maxLines
			lines = append(lines[:maxLines], fmt.Sprintf("... %d more %s; go run ./hack/gen-contract diff -input %s -max-lines 0 prints them all",
				more, plural(more, "line", "lines"), in.name,
			))
		}
		for _, l := range lines {
			b.WriteString(l)
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

// describeDigest returns the digest of entries, or "absent".
func describeDigest(entries []entry) string {
	if len(entries) == 0 {
		return "absent"
	}
	return digest(stream(entries))
}

// prettyLines returns the canonical form of entries for reading: each entry
// as a line naming it, then its data, JSON indented, line by line. A Go
// declaration or a change is a line already.
func prettyLines(entries []entry) []string {
	var lines []string
	for _, e := range entries {
		lines = append(lines, "# "+e.key)
		data := e.data
		if e.json {
			var buf bytes.Buffer
			if err := json.Indent(&buf, data, "", "  "); err == nil {
				data = buf.Bytes()
			}
		}
		if len(data) > 0 {
			lines = append(lines, strings.Split(string(data), "\n")...)
		}
	}
	return lines
}

// An edit is a line of a diff: kept (' '), removed ('-') or added ('+').
type edit struct {
	kind byte
	line string
	// a and b are how many lines of each side come before it.
	a, b int
}

// maxEditDistance bounds the work of diffLines: past this many removed and
// added lines, it removes and adds the whole changed range instead of
// finding the shortest script, which the cap of the output would cut
// anyway.
const maxEditDistance = 1000

// diffLines returns a shortest edit script turning a into b.
func diffLines(a, b []string) []edit {
	p := 0
	for p < len(a) && p < len(b) && a[p] == b[p] {
		p++
	}
	s := 0
	for s < len(a)-p && s < len(b)-p && a[len(a)-1-s] == b[len(b)-1-s] {
		s++
	}
	edits := make([]edit, 0, p+s)
	for i := range p {
		edits = append(edits, edit{kind: ' ', line: a[i], a: i, b: i})
	}
	edits = append(edits, myers(a[p:len(a)-s], b[p:len(b)-s], p, p)...)
	for i := range s {
		ai, bi := len(a)-s+i, len(b)-s+i
		edits = append(edits, edit{kind: ' ', line: a[ai], a: ai, b: bi})
	}
	return edits
}

// myers returns a shortest edit script turning a into b by Myers'
// algorithm, with the positions of its lines offset by aOff and bOff.
func myers(a, b []string, aOff, bOff int) []edit {
	n, m := len(a), len(b)
	if n+m == 0 {
		return nil
	}
	trace, ok := myersTrace(a, b)
	if !ok {
		edits := make([]edit, 0, n+m)
		for i, l := range a {
			edits = append(edits, edit{kind: '-', line: l, a: aOff + i, b: bOff})
		}
		for i, l := range b {
			edits = append(edits, edit{kind: '+', line: l, a: aOff + n, b: bOff + i})
		}
		return edits
	}
	return myersBacktrack(a, b, aOff, bOff, trace)
}

// myersTrace runs the forward pass of Myers' algorithm and returns the
// furthest reaching x of every diagonal k, offset by len(v)/2, before each
// step d, or false past maxEditDistance steps.
func myersTrace(a, b []string) ([][]int, bool) {
	n, m := len(a), len(b)
	limit := min(n+m, maxEditDistance)
	offset := limit + 1
	v := make([]int, 2*limit+3)
	var trace [][]int
	for d := 0; d <= limit; d++ {
		trace = append(trace, slices.Clone(v))
		for k := -d; k <= d; k += 2 {
			x := v[offset+k-1] + 1
			if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
				x = v[offset+k+1]
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[offset+k] = x
			if x >= n && y >= m {
				return trace, true
			}
		}
	}
	return nil, false
}

// myersBacktrack returns the edit script a trace of myersTrace found.
func myersBacktrack(a, b []string, aOff, bOff int, trace [][]int) []edit {
	offset := len(trace[0]) / 2
	var rev []edit
	x, y := len(a), len(b)
	for d, vd := range slices.Backward(trace) {
		k := x - y
		prevK := k - 1
		if k == -d || (k != d && vd[offset+k-1] < vd[offset+k+1]) {
			prevK = k + 1
		}
		prevX := vd[offset+prevK]
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			x--
			y--
			rev = append(rev, edit{kind: ' ', line: a[x], a: aOff + x, b: bOff + y})
		}
		switch {
		case d == 0:
		case x == prevX:
			rev = append(rev, edit{kind: '+', line: b[prevY], a: aOff + prevX, b: bOff + prevY})
		default:
			rev = append(rev, edit{kind: '-', line: a[prevX], a: aOff + prevX, b: bOff + prevY})
		}
		x, y = prevX, prevY
	}
	slices.Reverse(rev)
	return rev
}

// diffContext is how many kept lines a hunk shows around a change.
const diffContext = 3

// unifiedDiff returns the lines of a unified diff of an edit script, or
// none when nothing changed.
func unifiedDiff(edits []edit, labelA, labelB string) []string {
	var changes []int
	for i, e := range edits {
		if e.kind != ' ' {
			changes = append(changes, i)
		}
	}
	if len(changes) == 0 {
		return nil
	}
	lines := []string{"--- " + labelA, "+++ " + labelB}
	// A hunk spans start to end, and names the entry of its first change.
	first := changes[0]
	start := max(first-diffContext, 0)
	end := min(first+1+diffContext, len(edits))
	for _, c := range changes[1:] {
		if c-diffContext > end {
			lines = append(lines, hunk(edits[start:end], entryOf(edits[:first+1]))...)
			first, start = c, c-diffContext
		}
		end = min(c+1+diffContext, len(edits))
	}
	return append(lines, hunk(edits[start:end], entryOf(edits[:first+1]))...)
}

// entryOf returns the line naming the entry the last of edits is in, which
// the header of a hunk shows as git shows the function a hunk is in. No
// canonical form has a line that starts with "# " but those that name an
// entry: JSON is indented, Go has no #, and a change starts with "#<n>:".
func entryOf(edits []edit) string {
	for _, e := range slices.Backward(edits) {
		if strings.HasPrefix(e.line, "# ") {
			return e.line
		}
	}
	return ""
}

// hunk returns the lines of a hunk of a unified diff in the entry entry.
func hunk(edits []edit, entry string) []string {
	var aLen, bLen int
	for _, e := range edits {
		if e.kind != '+' {
			aLen++
		}
		if e.kind != '-' {
			bLen++
		}
	}
	aStart, bStart := edits[0].a, edits[0].b
	if aLen > 0 {
		aStart++
	}
	if bLen > 0 {
		bStart++
	}
	lines := make([]string, 0, 1+len(edits))
	lines = append(lines, strings.TrimSpace(fmt.Sprintf("@@ -%d,%d +%d,%d @@ %s", aStart, aLen, bStart, bLen, entry)))
	for _, e := range edits {
		lines = append(lines, string(e.kind)+e.line)
	}
	return lines
}
