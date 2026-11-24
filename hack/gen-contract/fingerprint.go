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
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

// scheme names the way this command fingerprints the contract: the inputs
// below, what each strips and the canonical stream it hashes. A record
// keeps its digests under the name of the scheme that computed them, as
// go.sum names its hash, so a digest is only ever compared with one
// computed the same way, and a new scheme adds digests next to the old
// ones instead of rewriting them.
//
// Any change that can change the digest of an unchanged contract, such as
// an input added, removed or regrouped, a rule of what an input strips or
// the canonical form, must come with a new scheme name (c2, c3, ...), its
// golden files in testdata/golden and a rebaseline of the latest release
// record. TestGolden fails until it does.
const scheme = "c1"

const digestPrefix = "sha256:"

// An input is a logical part of the contract, which a release record
// fingerprints under its name. The name says what it is, not where it
// lives, so moving a file does not rename an input.
type input struct {
	name  string
	parts []part
}

// A part is a set of contract artifacts of an input.
type part struct {
	// pattern names the artifacts, relative to the repository root: a glob
	// of files, or a directory followed by /... for every Go package in it.
	pattern string
	// entries returns the canonical form of each artifact named by pattern,
	// or none when there is no such artifact.
	entries func(root, pattern string) ([]entry, error)
}

// inputs are every artifact core and extensions share, by input. When the
// JSON Schema of the uiSchema lands, it joins them under a new scheme. A
// change that adds an input also adds its paths to the contract-adjacent
// block of .github/CODEOWNERS.
//
//nolint:gochecknoglobals // immutable list, the one place the inputs are declared
var inputs = []input{
	// The CRDs, which the API server serves to every client.
	{name: "crds", parts: []part{{pattern: "config/crd/bases/*.yaml", entries: yamlEntries(canonicalCRD)}}},
	// The HTTP API, and the CRD-derived schemas it refers to.
	{name: "http-api", parts: []part{
		{pattern: "api/openapi/http-api.yaml", entries: yamlEntries(canonicalOpenAPI)},
		{pattern: "api/openapi/crds.gen.yaml", entries: yamlEntries(canonicalOpenAPI)},
	}},
	// The built-in BackupStorageClasses, whose storages every backup
	// executor writes to.
	{name: "storage-classes", parts: []part{{pattern: "config/backupstorageclasses/*.yaml", entries: yamlEntries(canonicalBackupStorageClass)}}},
	// The Go identifiers and rules both sides link.
	{name: "go-api", parts: []part{
		{pattern: "contract/...", entries: goAPIEntries},
		{pattern: "backupstorageclasses/...", entries: goAPIEntries},
	}},
	// The changes no artifact shows. It is never absent: a tree without
	// the log has the empty one.
	{name: "changes", parts: []part{{pattern: changesFile, entries: changesEntries}}},
}

// inputNames returns the names of the inputs, in the order they are
// declared.
func inputNames() []string {
	names := make([]string, 0, len(inputs))
	for _, in := range inputs {
		names = append(names, in.name)
	}
	return names
}

// inputPrefixes returns the paths that hold the artifacts of every input:
// the directory of each pattern, or the file it names.
func inputPrefixes() []string {
	var prefixes []string
	for _, in := range inputs {
		for _, p := range in.parts {
			prefix := strings.TrimSuffix(p.pattern, "/...")
			for strings.ContainsAny(prefix, "*?[") {
				prefix = path.Dir(prefix)
			}
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes
}

// sortedInputs returns the names of a map of digests, in the order of the
// inputs, then those of no input, sorted.
func sortedInputs(digests map[string]string) []string {
	names := make([]string, 0, len(digests))
	for _, name := range inputNames() {
		if _, ok := digests[name]; ok {
			names = append(names, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(digests)) {
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}

// An entry is the canonical form of one artifact.
type entry struct {
	// key names the artifact, uniquely within its input.
	key  string
	data []byte
	// json is whether data is JSON, which a diff indents.
	json bool
}

// entries returns the canonical form of every artifact of an input in the
// repository at root, sorted by key; none when the input is absent.
func (in input) entries(root string) ([]entry, error) {
	var entries []entry
	for _, p := range in.parts {
		e, err := p.entries(root, p.pattern)
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", in.name, p.pattern, err)
		}
		entries = append(entries, e...)
	}
	slices.SortFunc(entries, func(a, b entry) int { return strings.Compare(a.key, b.key) })
	for j := 1; j < len(entries); j++ {
		if entries[j].key == entries[j-1].key {
			return nil, fmt.Errorf("%s: two artifacts are named %s", in.name, entries[j].key)
		}
	}
	return entries, nil
}

// stream returns the canonical stream of the entries of an input: each
// entry as its key, a space, the length of its data, a newline, the data
// and a newline. The length keeps an entry from reading as the start of
// the next. The digest of an input is the SHA-256 of its stream, which
// diff -raw prints.
func stream(entries []entry) []byte {
	var b bytes.Buffer
	for _, e := range entries {
		fmt.Fprintf(&b, "%s %d\n", e.key, len(e.data))
		b.Write(e.data)
		b.WriteByte('\n')
	}
	return b.Bytes()
}

// digest returns the digest of a canonical stream, as "sha256:<hex>".
func digest(stream []byte) string {
	sum := sha256.Sum256(stream)
	return digestPrefix + hex.EncodeToString(sum[:])
}

// streams returns the canonical stream of every input present in the
// repository at root, by input name. An input none of whose artifacts
// exist, such as the Go API at a tag that predates the contract package,
// is absent.
func streams(root string) (map[string][]byte, error) {
	out := make(map[string][]byte, len(inputs))
	for _, in := range inputs {
		entries, err := in.entries(root)
		if err != nil {
			return nil, err
		}
		if len(entries) > 0 {
			out[in.name] = stream(entries)
		}
	}
	return out, nil
}

// fingerprints returns the digest of every input present in the repository
// at root, by input name, under scheme. A digest changes when its part of
// the contract does, and only then: it ignores documentation, such as
// descriptions and printer columns, the metadata of manifests, which tools
// stamp, the layout of files and every unexported Go identifier.
func fingerprints(root string) (map[string]string, error) {
	s, err := streams(root)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(s))
	for name, data := range s {
		out[name] = digest(data)
	}
	return out, nil
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
			entries = append(entries, entry{key: key, data: data, json: true})
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

// changeEntry is an entry of the CHANGES log: "- #<pull request>: <what
// changed>", on a line of its own.
var changeEntry = regexp.MustCompile(`^- (#[1-9][0-9]*: \S.*)$`)

// changesEntries returns the entry of the CHANGES log: its entries, in the
// order of the file. A missing log has no entries, the same as a log with
// only comments, so a tag that predates the log hashes as the empty one.
func changesEntries(root, pattern string) ([]entry, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(pattern))) //nolint:gosec // a file of this repository
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	changes, err := parseChanges(data)
	if err != nil {
		return nil, err
	}
	return []entry{{key: pattern, data: []byte(strings.Join(changes, "\n"))}}, nil
}

// parseChanges returns the entries of a CHANGES log, without the "- " and
// trailing spaces. Every line is blank, a comment, which is "#" alone or
// followed by a space, or an entry; any other line is an error, so a line
// meant as an entry cannot silently fail to count. Only entries are
// hashed, so the comments can explain the file without moving the
// contract version.
func parseChanges(data []byte) ([]string, error) {
	var changes []string
	n := 0
	for line := range strings.Lines(string(data)) {
		n++
		line = strings.TrimRight(line, " \t\r\n")
		switch {
		case line == "", line == "#", strings.HasPrefix(line, "# "):
		case changeEntry.MatchString(line):
			changes = append(changes, changeEntry.FindStringSubmatch(line)[1])
		default:
			return nil, fmt.Errorf("line %d is neither a comment (\"# ...\") nor an entry (\"- #<pull request>: <what changed>\"): %q", n, line)
		}
	}
	return changes, nil
}
