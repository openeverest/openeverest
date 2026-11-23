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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const fingerprintPrefix = "sha256:"

// An input is a set of contract artifacts the fingerprint covers.
type input struct {
	// pattern names the artifacts, relative to the repository root: a glob
	// of files, or a directory followed by /... for every Go package in it.
	pattern string
	// entries returns the canonical form of each artifact named by pattern.
	entries func(root, pattern string) ([]entry, error)
}

// inputs are every artifact core and extensions share, in the order they
// are hashed. When the JSON Schema of the uiSchema lands, it joins them.
//
//nolint:gochecknoglobals // immutable list, the one place the inputs are declared
var inputs = []input{
	// The CRDs, which the API server serves to every client.
	{pattern: "config/crd/bases/*.yaml", entries: yamlEntries(canonicalCRD)},
	// The HTTP API, and the CRD-derived schemas it refers to.
	{pattern: "api/openapi/http-api.yaml", entries: yamlEntries(canonicalOpenAPI)},
	{pattern: "api/openapi/crds.gen.yaml", entries: yamlEntries(canonicalOpenAPI)},
	// The built-in BackupStorageClasses, whose storages every backup
	// executor writes to.
	{pattern: "config/backupstorageclasses/*.yaml", entries: yamlEntries(canonicalBackupStorageClass)},
	// The Go identifiers and rules both sides link.
	{pattern: "contract/...", entries: goAPIEntries},
	{pattern: "backupstorageclasses/...", entries: goAPIEntries},
	// The changes no artifact shows.
	{pattern: changesFile, entries: changesEntries},
}

// An entry is the canonical form of one artifact.
type entry struct {
	// key names the artifact, uniquely within its input.
	key  string
	data []byte
}

// fingerprint returns the SHA-256 of the canonical form of every input of
// the repository at root, as "sha256:<hex>". It changes when the contract
// does, and only then: it ignores documentation, such as descriptions and
// printer columns, the metadata of manifests, which tools stamp, the
// layout of files and every unexported Go identifier.
func fingerprint(root string, in []input) (string, error) {
	h := sha256.New()
	for _, i := range in {
		entries, err := i.entries(root, i.pattern)
		if err != nil {
			return "", fmt.Errorf("%s: %w", i.pattern, err)
		}
		if len(entries) == 0 {
			return "", fmt.Errorf("%s: no contract artifact found", i.pattern)
		}
		slices.SortFunc(entries, func(a, b entry) int { return strings.Compare(a.key, b.key) })
		for j, e := range entries {
			if j > 0 && e.key == entries[j-1].key {
				return "", fmt.Errorf("%s: two artifacts are named %s", i.pattern, e.key)
			}
			fmt.Fprintf(h, "%s %s %d\n", i.pattern, e.key, len(e.data))
			h.Write(e.data)
		}
	}
	return fingerprintPrefix + hex.EncodeToString(h.Sum(nil)), nil
}

// glob returns the files matching pattern under root, relative to root,
// with forward slashes, sorted.
func glob(root, pattern string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(matches))
	for _, m := range matches {
		rel, err := filepath.Rel(root, m)
		if err != nil {
			return nil, err
		}
		files = append(files, filepath.ToSlash(rel))
	}
	slices.Sort(files)
	return files, nil
}

// canonicalObject returns the key and the canonical content of a YAML
// object, or skip when the file holds no artifact of the input.
type canonicalObject func(file string, obj map[string]any) (key string, content any, skip bool, err error)

// yamlEntries returns the entries of the YAML files matching a pattern, each
// a single object made canonical by canonical and encoded as JSON with
// sorted keys.
func yamlEntries(canonical canonicalObject) func(root, pattern string) ([]entry, error) {
	return func(root, pattern string) ([]entry, error) {
		files, err := glob(root, pattern)
		if err != nil {
			return nil, err
		}
		var entries []entry
		for _, file := range files {
			obj, err := readYAMLObject(filepath.Join(root, filepath.FromSlash(file)))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", file, err)
			}
			key, content, skip, err := canonical(file, obj)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", file, err)
			}
			if skip {
				continue
			}
			data, err := json.Marshal(content)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", file, err)
			}
			entries = append(entries, entry{key: key, data: data})
		}
		return entries, nil
	}
}

// readYAMLObject reads the single YAML document of a file. Numbers keep
// their exact text, so no value depends on floating-point rounding.
func readYAMLObject(file string) (map[string]any, error) {
	f, err := os.Open(file) //nolint:gosec // a contract artifact of this repository
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read only

	reader := utilyaml.NewYAMLReader(bufio.NewReader(f))
	var obj map[string]any
	for {
		doc, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		data, err := yaml.YAMLToJSON(doc)
		if err != nil {
			return nil, err
		}
		if string(bytes.TrimSpace(data)) == "null" {
			continue
		}
		if obj != nil {
			return nil, errors.New("holds more than one YAML document")
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		if err := dec.Decode(&obj); err != nil {
			return nil, err
		}
	}
	if obj == nil {
		return nil, errors.New("holds no YAML document")
	}
	return obj, nil
}

// canonicalCRD returns the spec of a CustomResourceDefinition without what
// does not change how it is served: metadata, which holds controller-gen's
// version and the window tools stamp, printer columns, and documentation.
func canonicalCRD(_ string, obj map[string]any) (string, any, bool, error) {
	if obj["kind"] != "CustomResourceDefinition" {
		return "", nil, false, fmt.Errorf("kind %v is not CustomResourceDefinition", obj["kind"])
	}
	spec, ok := obj["spec"].(map[string]any)
	if !ok {
		return "", nil, false, errors.New("no spec")
	}
	names, _ := spec["names"].(map[string]any)
	plural, _ := names["plural"].(string)
	group, _ := spec["group"].(string)
	if plural == "" || group == "" {
		return "", nil, false, errors.New("no spec.group or spec.names.plural")
	}
	versions, _ := spec["versions"].([]any)
	for _, v := range versions {
		version, ok := v.(map[string]any)
		if !ok {
			continue
		}
		delete(version, "additionalPrinterColumns")
		delete(version, "deprecationWarning")
		if schema, ok := version["schema"].(map[string]any); ok {
			stripSchema(schema["openAPIV3Schema"])
		}
	}
	return plural + "." + group, spec, false, nil
}

// canonicalBackupStorageClass returns the spec of a built-in
// BackupStorageClass without its presentation: display name, description
// and the uiSchema core's UI renders it with.
func canonicalBackupStorageClass(file string, obj map[string]any) (string, any, bool, error) {
	if filepath.Base(file) == "kustomization.yaml" {
		return "", nil, true, nil
	}
	if obj["kind"] != "BackupStorageClass" {
		return "", nil, false, fmt.Errorf("kind %v is not BackupStorageClass", obj["kind"])
	}
	metadata, _ := obj["metadata"].(map[string]any)
	name, _ := metadata["name"].(string)
	spec, ok := obj["spec"].(map[string]any)
	if name == "" || !ok {
		return "", nil, false, errors.New("no metadata.name or spec")
	}
	delete(spec, "displayName")
	delete(spec, "description")
	delete(spec, "uiSchema")
	if params, ok := spec["parametersSchema"].(map[string]any); ok {
		stripSchema(params["openAPIV3Schema"])
	}
	stripValidationMessages(spec["validations"])
	return name, spec, false, nil
}

// Keywords of a JSON schema whose value holds schemas: a map of them, a
// schema or a list of them.
//
//nolint:gochecknoglobals // immutable lookup tables
var (
	schemaMapKeywords    = []string{"properties", "patternProperties", "definitions", "dependencies"}
	schemaOrListKeywords = []string{"items", "additionalProperties", "additionalItems", "not", "allOf", "anyOf", "oneOf"}
)

// stripSchema removes documentation from a CRD's OpenAPI v3 schema in place:
// description and example of every schema, and message and
// messageExpression of its CEL rules, whose rule stays. It recurses only
// into keywords that hold schemas, so a property named description is kept.
func stripSchema(node any) {
	schema, ok := node.(map[string]any)
	if !ok {
		return
	}
	delete(schema, "description")
	delete(schema, "example")
	stripValidationMessages(schema["x-kubernetes-validations"])
	for _, k := range schemaMapKeywords {
		if m, ok := schema[k].(map[string]any); ok {
			for _, s := range m {
				stripSchema(s)
			}
		}
	}
	for _, k := range schemaOrListKeywords {
		switch v := schema[k].(type) {
		case map[string]any:
			stripSchema(v)
		case []any:
			for _, s := range v {
				stripSchema(s)
			}
		}
	}
}

// stripValidationMessages removes the messages of a list of CEL rules.
func stripValidationMessages(node any) {
	rules, _ := node.([]any)
	for _, r := range rules {
		if rule, ok := r.(map[string]any); ok {
			delete(rule, "message")
			delete(rule, "messageExpression")
		}
	}
}

//nolint:gochecknoglobals // immutable lookup tables
var (
	// openAPIDocKeywords are the OpenAPI keywords that hold documentation.
	openAPIDocKeywords = []string{"description", "summary", "example", "examples"}

	// openAPINameMaps are the OpenAPI keywords whose value maps names chosen
	// by the author, such as property names or paths, to objects. A name
	// there is never a keyword, even when it reads like one.
	openAPINameMaps = map[string]bool{
		"paths": true, "properties": true, "patternProperties": true, "schemas": true,
		"responses": true, "parameters": true, "headers": true, "content": true,
		"securitySchemes": true, "requestBodies": true, "callbacks": true, "links": true,
		"encoding": true, "variables": true, "definitions": true, "mapping": true, "scopes": true,
	}

	// openAPIValues are the OpenAPI keywords whose value is data, not a part
	// of the document, so a key there is never a keyword.
	openAPIValues = map[string]bool{"default": true, "enum": true, "const": true}
)

// canonicalOpenAPI returns an OpenAPI document without documentation:
// descriptions, summaries and examples, wherever they are keywords.
func canonicalOpenAPI(file string, obj map[string]any) (string, any, bool, error) {
	if _, ok := obj["openapi"]; !ok {
		return "", nil, false, errors.New("not an OpenAPI document")
	}
	stripOpenAPI(obj, false)
	return file, obj, false, nil
}

func stripOpenAPI(node any, names bool) {
	switch n := node.(type) {
	case map[string]any:
		if names {
			for _, v := range n {
				stripOpenAPI(v, false)
			}
			return
		}
		for _, k := range openAPIDocKeywords {
			delete(n, k)
		}
		for k, v := range n {
			switch {
			case k == "x-kubernetes-validations":
				stripValidationMessages(v)
			case openAPIValues[k], strings.HasPrefix(k, "x-"):
			default:
				stripOpenAPI(v, openAPINameMaps[k])
			}
		}
	case []any:
		for _, v := range n {
			stripOpenAPI(v, false)
		}
	}
}

// changesFile is the log of contract changes no artifact shows.
const changesFile = "contract/CHANGES"

// changesEntries returns the entry of the CHANGES log: every line of it,
// without blank lines, trailing spaces and carriage returns. It has no
// comment syntax, so no change can be recorded in a way that does not
// count.
func changesEntries(root, pattern string) ([]entry, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(pattern))) //nolint:gosec // a file of this repository
	if err != nil {
		return nil, err
	}
	var lines []string
	for line := range strings.Lines(string(data)) {
		if line = strings.TrimRight(line, " \t\r\n"); line != "" {
			lines = append(lines, line)
		}
	}
	return []entry{{key: pattern, data: []byte(strings.Join(lines, "\n"))}}, nil
}
