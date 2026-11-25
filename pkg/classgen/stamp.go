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

package classgen

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/openeverest/openeverest/v2/contract"
)

const (
	// ChartHelperFile is the chart template, under templates/, that
	// StampChart writes: it defines AnnotationsHelper.
	ChartHelperFile = "_openeverest_contract.tpl"

	// AnnotationsHelper is the helper that renders the stamp, and the
	// acknowledgement set in the chart value
	// compatibility.acknowledge.coreVersion, to include in
	// metadata.annotations of the objects OpenEverest interprets.
	AnnotationsHelper = "openeverest.contract.annotations"
)

// ChartHelper is the content of ChartHelperFile. The stamp itself lives in
// Chart.yaml, so the helper only reads it: every object carries the value of
// the chart, and an installer reads it before rendering anything.
//
//go:embed chartfiles/_openeverest_contract.tpl
var ChartHelper string

// ChartStamp is what StampChart did.
type ChartStamp struct {
	// Previous is the stamp Chart.yaml carried before, or empty.
	Previous string
	// Written lists the files StampChart changed, relative to the chart.
	Written []string
}

// StampChart stamps the chart in chartDir with the contract version
// version: the contract.VersionAnnotation in Chart.yaml, and ChartHelper in
// templates/ChartHelperFile, which renders it on the objects that include
// AnnotationsHelper. It writes only the files whose content changes.
//
// The stamp tells installers and OpenEverest the oldest OpenEverest the
// chart's build works with, so it must be the contract version of the core
// the integration is built against, never a value written by hand.
func StampChart(chartDir, version string) (ChartStamp, error) {
	var res ChartStamp
	if !contract.Valid(version) {
		return res, fmt.Errorf("contract version %q is not a SemVer 2.0.0 version", version)
	}

	chartFile := filepath.Join(chartDir, "Chart.yaml")
	data, err := os.ReadFile(chartFile) //nolint:gosec // the chart the caller names
	if err != nil {
		return res, fmt.Errorf("reading %s: %w", chartFile, err)
	}
	stamped, err := setChartAnnotation(data, contract.VersionAnnotation, version)
	if err != nil {
		return res, fmt.Errorf("stamping %s: %w", chartFile, err)
	}
	res.Previous = chartStamp(data)
	if !bytes.Equal(stamped, data) {
		if err := os.WriteFile(chartFile, stamped, 0o600); err != nil { //nolint:gosec // the chart the caller names
			return res, fmt.Errorf("writing %s: %w", chartFile, err)
		}
		res.Written = append(res.Written, "Chart.yaml")
	}

	helper := "templates/" + ChartHelperFile
	helperFile := filepath.Join(chartDir, "templates", ChartHelperFile)
	switch current, err := os.ReadFile(helperFile); { //nolint:gosec // the chart the caller names
	case err == nil && string(current) == ChartHelper:
		return res, nil
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return res, fmt.Errorf("reading %s: %w", helperFile, err)
	}
	if err := os.MkdirAll(filepath.Dir(helperFile), 0o750); err != nil {
		return res, fmt.Errorf("creating %s: %w", filepath.Dir(helperFile), err)
	}
	if err := os.WriteFile(helperFile, []byte(ChartHelper), 0o600); err != nil {
		return res, fmt.Errorf("writing %s: %w", helperFile, err)
	}
	res.Written = append(res.Written, helper)
	return res, nil
}

// chartStamp returns the stamp in the Chart.yaml data, or empty.
func chartStamp(data []byte) string {
	var chart map[string]any
	if err := yaml.Unmarshal(data, &chart); err != nil {
		return ""
	}
	annotations, _ := chart["annotations"].(map[string]any)
	if v, ok := annotations[contract.VersionAnnotation]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// setChartAnnotation returns the Chart.yaml data with annotations[key] set
// to value. Chart.yaml is written by hand, so the edit is made in place in
// the text, at the positions the parsed YAML gives: comments, blank lines,
// quoting and the order of keys all stay as they were. Layouts the in-place
// edit does not handle (flow-style or empty annotations) fall back to
// re-encoding the parsed document, which keeps comments but not blank lines.
func setChartAnnotation(data []byte, key, value string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("not a YAML mapping")
	}
	want, err := expectedChart(data, key, value)
	if err != nil {
		return nil, err
	}

	if out, ok := spliceAnnotation(data, doc.Content[0], key, value); ok && chartEquals(out, want) {
		return out, nil
	}

	out, err := reencodeWithAnnotation(&doc, key, value)
	if err != nil {
		return nil, err
	}
	if !chartEquals(out, want) {
		return nil, errors.New("setting the annotation changed other content")
	}
	return out, nil
}

// spliceAnnotation edits data in place. ok is false when the layout is not
// one it handles.
func spliceAnnotation(data []byte, root *yaml.Node, key, value string) ([]byte, bool) {
	if root.Style&yaml.FlowStyle != 0 {
		return nil, false
	}
	lines := strings.SplitAfter(string(data), "\n")
	quoted := strconv.Quote(value)

	annotationsKey, annotations := mappingEntry(root, "annotations")
	if annotations == nil {
		out := string(data)
		if out != "" && !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		return []byte(out + "annotations:\n  " + key + ": " + quoted + "\n"), true
	}
	if annotations.Kind != yaml.MappingNode || annotations.Style&yaml.FlowStyle != 0 || len(annotations.Content) == 0 {
		return nil, false
	}

	if _, v := mappingEntry(annotations, key); v != nil {
		if v.Kind == yaml.ScalarNode && v.Value == value {
			return data, true
		}
		line, ok := replaceScalar(lines, v, quoted)
		if !ok {
			return nil, false
		}
		lines[v.Line-1] = line
		return []byte(strings.Join(lines, "")), true
	}

	// Insert the annotation as the first entry, right below the
	// annotations key, so it does not come between an entry and the
	// comments above it.
	first := annotations.Content[0]
	entry := strings.Repeat(" ", first.Column-1) + key + ": " + quoted + "\n"
	at := annotationsKey.Line
	if at > len(lines) || !strings.HasSuffix(lines[at-1], "\n") {
		return nil, false
	}
	lines = append(lines[:at], append([]string{entry}, lines[at:]...)...)
	return []byte(strings.Join(lines, "")), true
}

// replaceScalar returns the line of the single-line scalar v with its text
// replaced by replacement, keeping what follows it, such as a comment.
func replaceScalar(lines []string, v *yaml.Node, replacement string) (string, bool) {
	if v.Kind != yaml.ScalarNode || v.Line < 1 || v.Line > len(lines) {
		return "", false
	}
	line := lines[v.Line-1]
	start := runeOffset(line, v.Column-1)
	if start < 0 {
		return "", false
	}
	rest := line[start:]
	var end int
	switch v.Style {
	case yaml.DoubleQuotedStyle:
		end = closingQuote(rest, '"')
	case yaml.SingleQuotedStyle:
		end = closingQuote(rest, '\'')
	case 0:
		if !strings.HasPrefix(rest, v.Value) {
			return "", false
		}
		end = len(v.Value)
	case yaml.TaggedStyle, yaml.LiteralStyle, yaml.FoldedStyle, yaml.FlowStyle:
		return "", false
	default:
		return "", false
	}
	if end < 0 {
		return "", false
	}
	return line[:start] + replacement + rest[end:], true
}

// closingQuote returns the offset just past the quoted scalar that s starts
// with, or -1 when it does not end on this line.
func closingQuote(s string, quote byte) int {
	for i := 1; i < len(s); i++ {
		switch {
		case quote == '"' && s[i] == '\\':
			i++
		case s[i] == quote && quote == '\'' && i+1 < len(s) && s[i+1] == '\'':
			i++
		case s[i] == quote:
			return i + 1
		}
	}
	return -1
}

// runeOffset returns the byte offset of the n-th rune of s, as yaml.v3
// counts columns in characters.
func runeOffset(s string, n int) int {
	for i := range s {
		if n == 0 {
			return i
		}
		n--
	}
	if n == 0 {
		return len(s)
	}
	return -1
}

func reencodeWithAnnotation(doc *yaml.Node, key, value string) ([]byte, error) {
	root := doc.Content[0]
	_, annotations := mappingEntry(root, "annotations")
	if annotations == nil {
		annotations = &yaml.Node{Kind: yaml.MappingNode}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "annotations"}, annotations)
	}
	if annotations.Kind != yaml.MappingNode {
		*annotations = yaml.Node{Kind: yaml.MappingNode, HeadComment: annotations.HeadComment, LineComment: annotations.LineComment}
	}
	// Block style for a mapping that may have been written as {}.
	annotations.Style &^= yaml.FlowStyle
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: value, Style: yaml.DoubleQuotedStyle}
	if _, v := mappingEntry(annotations, key); v != nil {
		*v = *valueNode
	} else {
		annotations.Content = append(annotations.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, valueNode)
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("encoding: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encoding: %w", err)
	}
	return buf.Bytes(), nil
}

// mappingEntry returns the key and value nodes of key in the mapping m.
func mappingEntry(m *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i], m.Content[i+1]
		}
	}
	return nil, nil
}

// expectedChart returns the content of the Chart.yaml data once
// annotations[key] is value.
func expectedChart(data []byte, key, value string) (map[string]any, error) {
	var chart map[string]any
	if err := yaml.Unmarshal(data, &chart); err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}
	if chart == nil {
		chart = make(map[string]any)
	}
	annotations, ok := chart["annotations"].(map[string]any)
	if !ok {
		if chart["annotations"] != nil {
			return nil, errors.New("annotations is not a mapping")
		}
		annotations = make(map[string]any)
	}
	annotations[key] = value
	chart["annotations"] = annotations
	return chart, nil
}

func chartEquals(data []byte, want map[string]any) bool {
	var got map[string]any
	if err := yaml.Unmarshal(data, &got); err != nil {
		return false
	}
	return reflect.DeepEqual(got, want)
}
