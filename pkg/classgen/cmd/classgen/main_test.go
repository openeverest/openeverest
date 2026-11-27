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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/pkg/classgen"
)

const (
	paramsPackage = "../../testdata/params"

	alpha1 = "v2.0.0-alpha.1"
	pseudo = "v2.0.0-dev.2.0.20261123230825-5a38fe56add1"

	unstampedChart = "apiVersion: v2\nname: monitoring-x\nversion: 0.1.0\n"

	monitoringClass = `apiVersion: monitoring.openeverest.io/v1alpha1
kind: MonitoringClass
metadata:
  name: x
  annotations:
    {{- include "openeverest.contract.annotations" . | nindent 4 }}
spec:
  controllerName: example.com/x
`
)

// sameCore is the check of a module that requires the core classgen links.
func sameCore() (module, string, error) {
	return core(alpha1), "", nil
}

func core(version string) module {
	return module{Path: coreModule, Version: version}
}

// replaced is the core release alpha1, replaced by path.
func replaced(path, replacementVersion string) module {
	m := core(alpha1)
	m.Replace = &module{Path: path, Version: replacementVersion}
	return m
}

// writeChart writes a chart with a MonitoringClass that includes the
// helper, and returns its directory.
func writeChart(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(unstampedChart), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "templates"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "monitoringclass.yaml"), []byte(monitoringClass), 0o600))
	return dir
}

func TestRunPrintsSchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	require.NoError(t, run([]string{"--package", paramsPackage, "--type", "Tag"}, &stdout, &bytes.Buffer{}, sameCore))

	out, ok := strings.CutPrefix(stdout.String(), header)
	require.True(t, ok, "output starts with the generated-code header")
	want := `
type: object
required: [name]
properties:
  name:
    type: string
    description: Name of the tag.
    pattern: ^[a-z]+$
`
	assert.YAMLEq(t, want, out)
}

func TestRunWritesOutput(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "schema.yaml")
	var stdout bytes.Buffer
	require.NoError(t, run([]string{"-package", paramsPackage, "-type", "Parameters", "-output", file}, &stdout, &bytes.Buffer{}, sameCore))
	assert.Empty(t, stdout.String())

	data, err := os.ReadFile(file) //nolint:gosec // the test's own file
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte(header)))
	schema := &apiextensionsv1.JSONSchemaProps{}
	require.NoError(t, yaml.UnmarshalStrict(data, schema))
	assert.Equal(t, []string{"url"}, schema.Required)
}

func TestRunErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "no package", args: []string{"--type", "Tag"}, wantErr: "--package and --type are required"},
		{name: "no type", args: []string{"--package", paramsPackage}, wantErr: "--package and --type are required"},
		{name: "extra argument", args: []string{"--package", paramsPackage, "--type", "Tag", "extra"}, wantErr: "unexpected arguments: extra"},
		{name: "unknown flag", args: []string{"--packages", paramsPackage}, wantErr: "flag provided but not defined"},
		{name: "unknown type", args: []string{"--package", paramsPackage, "--type", "Missing"}, wantErr: "type Missing not found"},
		{name: "stamp without a chart", args: []string{"stamp"}, wantErr: "--chart-dir is required"},
		{name: "stamp with an extra argument", args: []string{"stamp", "--chart-dir", "x", "extra"}, wantErr: "unexpected arguments: extra"},
		{name: "stamp of a missing chart", args: []string{"stamp", "--chart-dir", "missing"}, wantErr: "reading missing/Chart.yaml"},
		{name: "verify without a chart", args: []string{"verify", "--release"}, wantErr: "--chart-dir is required"},
		{name: "verify with an unknown flag", args: []string{"verify", "--chart-dir", "x", "--relase"}, wantErr: "flag provided but not defined"},
		{name: "verify of a missing chart", args: []string{"verify", "--chart-dir", "missing"}, wantErr: "loading missing/Chart.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := run(tt.args, &bytes.Buffer{}, &bytes.Buffer{}, sameCore)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestRunRefusesAnotherCore checks that no form writes anything when the
// core classgen links is not the one the module requires.
func TestRunRefusesAnotherCore(t *testing.T) {
	t.Parallel()

	refused := errors.New("this classgen links another core")
	anotherCore := func() (module, string, error) { return module{}, "", refused }

	dir := t.TempDir()
	output := filepath.Join(dir, "schema.yaml")
	err := run([]string{"--package", paramsPackage, "--type", "Tag", "--output", output}, &bytes.Buffer{}, &bytes.Buffer{}, anotherCore)
	require.ErrorIs(t, err, refused)
	assert.NoFileExists(t, output)

	var stdout bytes.Buffer
	err = run([]string{"--package", paramsPackage, "--type", "Tag"}, &stdout, &bytes.Buffer{}, anotherCore)
	require.ErrorIs(t, err, refused)
	assert.Empty(t, stdout.String())

	chartDir := writeChart(t)
	err = run([]string{"stamp", "--chart-dir", chartDir}, &bytes.Buffer{}, &bytes.Buffer{}, anotherCore)
	require.ErrorIs(t, err, refused)
	chart, err := os.ReadFile(filepath.Join(chartDir, "Chart.yaml")) //nolint:gosec // the test's own file
	require.NoError(t, err)
	assert.Equal(t, unstampedChart, string(chart))
	assert.NoFileExists(t, filepath.Join(chartDir, "templates", classgen.ChartHelperFile))

	err = run([]string{"verify", "--chart-dir", chartDir}, &bytes.Buffer{}, &bytes.Buffer{}, anotherCore)
	require.ErrorIs(t, err, refused)
}

func TestRunPrintsCoreWarning(t *testing.T) {
	t.Parallel()

	localCore := func() (module, string, error) { return core(""), "the core comes from a local directory", nil }
	var stderr bytes.Buffer
	require.NoError(t, run([]string{"stamp", "--chart-dir", writeChart(t)}, &bytes.Buffer{}, &stderr, localCore))
	assert.Contains(t, stderr.String(), "Warning: the core comes from a local directory\n")
}

func TestRunStamp(t *testing.T) {
	t.Parallel()

	dir := writeChart(t)
	chartFile := filepath.Join(dir, "Chart.yaml")
	helperFile := filepath.Join(dir, "templates", classgen.ChartHelperFile)
	stamped := unstampedChart + "annotations:\n  openeverest.io/contract-version: \"" + contract.Version + "\"\n"

	var stderr bytes.Buffer
	require.NoError(t, run([]string{"stamp", "--chart-dir", dir}, &bytes.Buffer{}, &stderr, sameCore))
	assert.Equal(t,
		"Stamped: "+dir+"/Chart.yaml (openeverest.io/contract-version="+contract.Version+")\n"+
			"Generated: "+dir+"/templates/"+classgen.ChartHelperFile+"\n"+
			floor(contract.Version)+"\n",
		stderr.String(),
	)
	chart, err := os.ReadFile(chartFile) //nolint:gosec // the test's own file
	require.NoError(t, err)
	assert.Equal(t, stamped, string(chart))
	helper, err := os.ReadFile(helperFile) //nolint:gosec // the test's own file
	require.NoError(t, err)
	assert.Equal(t, classgen.ChartHelper, string(helper))

	stderr.Reset()
	require.NoError(t, run([]string{"stamp", "--chart-dir", dir + "/"}, &bytes.Buffer{}, &stderr, sameCore))
	assert.Equal(t, floor(contract.Version)+"\n", stderr.String(), "a stamped chart is left as it is")

	require.NoError(t, os.WriteFile(chartFile, []byte(unstampedChart+"annotations:\n  openeverest.io/contract-version: 2.0.0-dev.4\n"), 0o600))
	stderr.Reset()
	require.NoError(t, run([]string{"stamp", "--chart-dir", dir}, &bytes.Buffer{}, &stderr, sameCore))
	assert.Equal(t,
		"Stamped: "+dir+"/Chart.yaml (openeverest.io/contract-version="+contract.Version+", was 2.0.0-dev.4)\n"+floor(contract.Version)+"\n",
		stderr.String(),
	)
}

func TestRunVerify(t *testing.T) {
	t.Parallel()

	dir := writeChart(t)
	err := run([]string{"verify", "--chart-dir", dir}, &bytes.Buffer{}, &bytes.Buffer{}, sameCore)
	require.ErrorContains(t, err, "3 problem(s) in "+dir+":\n  - Chart.yaml has no openeverest.io/contract-version annotation")

	require.NoError(t, run([]string{"stamp", "--chart-dir", dir}, &bytes.Buffer{}, &bytes.Buffer{}, sameCore))
	var stderr bytes.Buffer
	require.NoError(t, run([]string{"verify", "--chart-dir", dir}, &bytes.Buffer{}, &stderr, sameCore))
	assert.Equal(t,
		"Stamped with openeverest.io/contract-version="+contract.Version+":\n"+
			"  Chart.yaml\n"+
			"  MonitoringClass x (templates/monitoringclass.yaml)\n"+
			floor(contract.Version)+"\n",
		stderr.String(),
	)

	err = run([]string{"verify", "--chart-dir", dir, "--release"}, &bytes.Buffer{}, &bytes.Buffer{}, sameCore)
	if contract.IsDevelopment(contract.Version) {
		require.ErrorContains(t, err, "built against an unreleased OpenEverest ("+contract.Version+")")
	} else {
		require.NoError(t, err)
	}

	localCore := func() (module, string, error) { return replaced("../openeverest", ""), "", nil }
	err = run([]string{"verify", "--chart-dir", dir, "--release"}, &bytes.Buffer{}, &bytes.Buffer{}, localCore)
	require.ErrorContains(t, err, "go.mod replaces "+coreModule)
}

func TestCheckCore(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		linked, required module
		wantWarning      string
		wantErr          string
	}{
		"same release": {
			linked: core(alpha1), required: core(alpha1),
		},
		"same pseudo-version": {
			linked: core(pseudo), required: core(pseudo),
		},
		"another release": {
			linked: core("v2.0.0-alpha.2"), required: core(alpha1),
			wantErr: "this classgen links " + coreModule + "@v2.0.0-alpha.2, but this module requires " + coreModule + "@" + alpha1,
		},
		"same fork": {
			linked:   replaced("github.com/fork/openeverest/v2", "v2.0.0-alpha.1.1"),
			required: replaced("github.com/fork/openeverest/v2", "v2.0.0-alpha.1.1"),
		},
		"fork against upstream": {
			linked:   replaced("github.com/fork/openeverest/v2", "v2.0.0-alpha.1.1"),
			required: core(alpha1),
			wantErr:  "would write for another OpenEverest",
		},
		"both from a local directory": {
			linked:      replaced("../openeverest", "(devel)"),
			required:    replaced("../openeverest", ""),
			wantWarning: "comes from a local directory (../openeverest)",
		},
		"both from a workspace": {
			linked:      core("(devel)"),
			required:    core(""),
			wantWarning: "comes from a local directory",
		},
		"only the module replaces it with a directory": {
			linked:   core(alpha1),
			required: replaced("../openeverest", ""),
			wantErr:  "would write for another OpenEverest",
		},
		"only the binary was built from a directory": {
			linked:   replaced("/src/openeverest", "(devel)"),
			required: core(alpha1),
			wantErr:  "run it as `go tool classgen`",
		},
		"module in a workspace with core": {
			linked:      core("(devel)"),
			required:    module{Path: coreModule, Dir: "/src/openeverest"},
			wantWarning: "comes from a local directory (/src/openeverest)",
		},
		"binary built in a core checkout": {
			linked:   core(pseudo + "+dirty"),
			required: core(pseudo),
			wantErr:  "would write for another OpenEverest",
		},
		"binary built in a workspace with core": {
			linked:   core("(devel)"),
			required: core(alpha1),
			wantErr:  "would write for another OpenEverest",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			warning, err := checkCore(tt.linked, tt.required)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantWarning == "" {
				assert.Empty(t, warning)
				return
			}
			assert.Contains(t, warning, tt.wantWarning)
		})
	}
}

// TestLinkedCoreIsRequiredCore runs the check in core itself, whose test
// binary is built from the tree, as `go run` builds classgen there.
func TestLinkedCoreIsRequiredCore(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the go command")
	}
	t.Parallel()

	linked, err := linkedCore()
	require.NoError(t, err)
	assert.Equal(t, core("(devel)"), linked)

	required, warning, err := checkLinkedCore(t.Context())
	require.NoError(t, err)
	assert.Equal(t, coreModule, required.Path)
	assert.Empty(t, required.Version)
	assert.Contains(t, warning, coreModule+" comes from a local directory ("+required.Dir+")")
}

func TestRequiredCoreOutsideAModule(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the go command")
	}
	t.Parallel()

	_, err := requiredCore(t.Context(), t.TempDir())
	require.ErrorContains(t, err, "run classgen as `go tool classgen` from a module that requires it as a tool")
}

func TestReleaseProblems(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		required module
		version  string
		want     []string
	}{
		"released core": {
			required: core(alpha1), version: "2.0.0-alpha.1",
		},
		"development contract version": {
			required: core(alpha1), version: "2.0.0-alpha.1.next",
			want: []string{"built against an unreleased OpenEverest (2.0.0-alpha.1.next)"},
		},
		"development contract version after a GA release": {
			required: core("v2.0.0"), version: "2.1.0-alpha.0",
			want: []string{"built against an unreleased OpenEverest (2.1.0-alpha.0)"},
		},
		"pseudo-version, contract of a release": {
			required: core(pseudo), version: "2.0.0-alpha.1",
		},
		"core dev build": {
			required: core("v2.0.0-dev.4"), version: "2.0.0-dev.4",
		},
		"replaced by a fork release": {
			required: replaced("github.com/fork/openeverest/v2", "v2.0.0-alpha.1"), version: "2.0.0-alpha.1",
			want: []string{"go.mod replaces " + coreModule},
		},
		"replaced by a directory, unreleased": {
			required: replaced("../openeverest", ""), version: "2.0.0-alpha.1.next",
			want: []string{"built against an unreleased OpenEverest", "go.mod replaces " + coreModule},
		},
		"workspace module": {
			required: core(""), version: "2.0.0-alpha.1",
			want: []string{coreModule + " comes from a local directory"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := releaseProblems(tt.required, tt.version)

			require.Len(t, got, len(tt.want))
			for i, want := range tt.want {
				assert.Contains(t, got[i], want)
			}
		})
	}
}

func TestFloor(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"2.0.0-alpha.1":      "This build works with OpenEverest 2.0.0-alpha.1 and later.",
		"2.0.0":              "This build works with OpenEverest 2.0.0 and later.",
		"2.0.0-dev.4":        "This build works with OpenEverest 2.0.0-dev.4 and later.",
		"2.0.0-alpha.1.next": "Warning: built against an unreleased OpenEverest (2.0.0-alpha.1.next); it cannot be published to the Hub until that OpenEverest is released.",
		"2.1.0-alpha.0":      "Warning: built against an unreleased OpenEverest (2.1.0-alpha.0); it cannot be published to the Hub until that OpenEverest is released.",
	}
	for version, want := range tests {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, want, floor(version))
		})
	}
}
