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
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"sigs.k8s.io/yaml"
)

// The annotations core publishes its window in, as the contract package
// names them; this command cannot import it.
const (
	versionAnnotation        = "openeverest.io/contract-version"
	minVersionAnnotation     = "openeverest.io/min-contract-version"
	upgradableFromAnnotation = "openeverest.io/upgradable-from"
)

// A stamp is a set of manifests core ships that carry its contract version.
type stamp struct {
	// pattern is a glob of the manifests, relative to the repository root.
	pattern string
	// window is whether the manifests carry the whole window, not only the
	// contract version.
	window bool
}

// stamps are the manifests this command stamps, after the generators that
// write them, with the contract version of this core.
//
// The CRDs carry the whole window. Helm installs CRDs only once and never
// upgrades them, so only the CRDs the API server serves tell what a cluster
// actually runs; any client that can read a CRD finds the window there
// without knowing where core is installed; and the window changes in the
// same apply as the schemas it describes. The built-in
// BackupStorageClasses carry the contract version, as the objects every
// extension ships do.
//
//nolint:gochecknoglobals // immutable list, the one place the stamped manifests are declared
var stamps = []stamp{
	{pattern: "config/crd/bases/*.yaml", window: true},
	{pattern: "config/backupstorageclasses/*.yaml"},
	{pattern: "backupstorageclasses/*/class.yaml"},
}

// annotations returns the annotations a stamp sets for the window w.
func (s stamp) annotations(w window) map[string]string {
	a := map[string]string{versionAnnotation: w.version}
	if s.window {
		a[minVersionAnnotation] = w.minVersion
		a[upgradableFromAnnotation] = strings.Join(w.upgradableFrom, ",")
	}
	return a
}

// stampFiles stamps every manifest of stamps in the repository at root.
func stampFiles(root string, w window) error {
	for _, s := range stamps {
		files, err := glob(root, s.pattern)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return fmt.Errorf("%s: no manifest to stamp", s.pattern)
		}
		for _, file := range files {
			if filepath.Base(file) == "kustomization.yaml" {
				continue
			}
			path := filepath.Join(root, filepath.FromSlash(file))
			data, err := os.ReadFile(path) //nolint:gosec // a manifest of this repository
			if err != nil {
				return err
			}
			stamped, err := stampManifest(data, s.annotations(w))
			if err != nil {
				return fmt.Errorf("%s: %w", file, err)
			}
			if bytes.Equal(stamped, data) {
				continue
			}
			if err := os.WriteFile(path, stamped, 0o600); err != nil { //nolint:gosec // a manifest of this repository
				return err
			}
		}
	}
	return nil
}

// stampManifest returns a YAML manifest with annotations set on it. Only its
// top-level metadata block changes: it is rendered again the way
// controller-gen and the class generator render whole manifests, with
// sorted keys, so the rest of the file keeps the layout its generator gave
// it and a manifest already stamped stays the same.
func stampManifest(data []byte, annotations map[string]string) ([]byte, error) {
	var obj map[string]any
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	metadata, ok := obj["metadata"].(map[string]any)
	if !ok {
		return nil, errors.New("no metadata")
	}
	existing, _ := metadata["annotations"].(map[string]any)
	merged := make(map[string]any, len(existing)+len(annotations))
	maps.Copy(merged, existing)
	for k, v := range annotations {
		merged[k] = v
	}
	metadata["annotations"] = merged
	block, err := yaml.Marshal(map[string]any{"metadata": metadata})
	if err != nil {
		return nil, err
	}

	lines := strings.SplitAfter(string(data), "\n")
	isMetadata := func(line string) bool { return strings.TrimRight(line, "\r\n") == "metadata:" }
	start := slices.IndexFunc(lines, isMetadata)
	switch {
	case start < 0:
		return nil, errors.New("no top-level metadata block in block style")
	case slices.ContainsFunc(lines[start+1:], isMetadata):
		return nil, errors.New("more than one top-level metadata block")
	}
	// The block ends at the next line that is not indented: a top-level key,
	// a comment or the end of the document.
	end := start + 1
	for end < len(lines) && (strings.HasPrefix(lines[end], " ") || strings.TrimSpace(lines[end]) == "") {
		end++
	}
	for end > start+1 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	stamped := []byte(strings.Join(lines[:start], "") + string(block) + strings.Join(lines[end:], ""))

	// Check that nothing but the annotations changed.
	var got map[string]any
	if err := yaml.Unmarshal(stamped, &got); err != nil {
		return nil, fmt.Errorf("stamping breaks the manifest: %w", err)
	}
	if !reflect.DeepEqual(got, obj) {
		return nil, errors.New("stamping would change more than the annotations of the manifest")
	}
	return stamped, nil
}
