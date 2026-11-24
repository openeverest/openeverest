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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// The commands that read a release tag do it with git plumbing on the
// objects of the tag alone, so they work in a clone that fetched only that
// tag, at depth 1. None of them fetches: make gen never needs the network.

// git runs git in the repository at root and returns its standard output.
func git(root string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", root}, args...)...) //nolint:gosec // git, with arguments of this command
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// hasRef reports whether ref names a commit of the repository at root.
func hasRef(root, ref string) bool {
	_, err := git(root, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return err == nil
}

// requireTag returns an error saying how to fetch tag unless the
// repository at root has it.
func requireTag(root, tag string) error {
	if hasRef(root, tag) {
		return nil
	}
	return fmt.Errorf("tag %[1]s is not here; fetch it: git fetch --depth=1 origin refs/tags/%[1]s:refs/tags/%[1]s "+
		"(or git fetch upstream --tags, from the remote of openeverest/openeverest)", tag,
	)
}

// showFile returns the content of the file at path, relative to the
// repository root, in the tree of ref, and false when there is none.
func showFile(root, ref, file string) ([]byte, bool, error) {
	if _, err := git(root, "cat-file", "-e", ref+":"+file); err != nil {
		return nil, false, nil //nolint:nilerr // no such file
	}
	data, err := git(root, "cat-file", "blob", ref+":"+file)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// extractInputs writes the files that hold the artifacts of every input in
// the tree of ref to a new temporary directory, and returns it with a
// function that removes it. The fingerprints of that directory are those
// of ref.
func extractInputs(root, ref string) (string, func(), error) {
	prefixes := inputPrefixes()
	listing, err := git(root, append([]string{"ls-tree", "-r", "-z", "--full-tree", ref, "--"}, prefixes...)...)
	if err != nil {
		return "", nil, err
	}
	type blob struct{ object, path string }
	var blobs []blob
	for rec := range strings.SplitSeq(strings.TrimSuffix(string(listing), "\x00"), "\x00") {
		if rec == "" {
			continue
		}
		meta, file, ok := strings.Cut(rec, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return "", nil, fmt.Errorf("git ls-tree: unexpected line %q", rec)
		}
		if !underAny(file, prefixes) {
			continue
		}
		if mode := fields[0]; mode != "100644" && mode != "100755" {
			return "", nil, fmt.Errorf("%s at %s is not a regular file (mode %s)", file, ref, mode)
		}
		blobs = append(blobs, blob{object: fields[2], path: file})
	}

	dir, err := os.MkdirTemp("", "gen-contract-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	objects := make([]string, 0, len(blobs))
	for _, b := range blobs {
		objects = append(objects, b.object)
	}
	contents, err := catFiles(root, objects)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	for i, b := range blobs {
		file := filepath.Join(dir, filepath.FromSlash(b.path))
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			cleanup()
			return "", nil, err
		}
		if err := os.WriteFile(file, contents[i], 0o600); err != nil {
			cleanup()
			return "", nil, err
		}
	}
	return dir, cleanup, nil
}

// underAny reports whether file is one of prefixes or below one.
func underAny(file string, prefixes []string) bool {
	for _, p := range prefixes {
		if file == p || strings.HasPrefix(file, p+"/") {
			return true
		}
	}
	return false
}

// catFiles returns the content of the blobs objects, in order, read with a
// single git cat-file --batch.
func catFiles(root string, objects []string) ([][]byte, error) {
	if len(objects) == 0 {
		return nil, nil
	}
	cmd := exec.CommandContext(context.Background(), "git", "-C", root, "cat-file", "--batch") //nolint:gosec // git, in the repository named on the command line
	cmd.Stdin = strings.NewReader(strings.Join(objects, "\n") + "\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git cat-file --batch: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	r := bufio.NewReader(bytes.NewReader(out))
	contents := make([][]byte, 0, len(objects))
	for _, object := range objects {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("git cat-file --batch: %s: %w", object, err)
		}
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[1] != "blob" {
			return nil, fmt.Errorf("git cat-file --batch: %s: unexpected header %q", object, strings.TrimSpace(header))
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("git cat-file --batch: %s: %w", object, err)
		}
		data := make([]byte, size)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("git cat-file --batch: %s: %w", object, err)
		}
		if b, err := r.ReadByte(); err != nil || b != '\n' {
			return nil, errors.New("git cat-file --batch: an object does not end with a newline")
		}
		contents = append(contents, data)
	}
	return contents, nil
}

// fingerprintsAt returns the fingerprints of the tree of ref, which must
// be here.
func fingerprintsAt(root, ref string) (map[string]string, error) {
	if !hasRef(root, ref) {
		return nil, requireTag(root, ref)
	}
	dir, cleanup, err := extractInputs(root, ref)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	fps, err := fingerprints(dir)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ref, err)
	}
	return fps, nil
}
