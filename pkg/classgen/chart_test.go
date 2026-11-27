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

package classgen_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/pkg/classgen"
)

const (
	stamp = "2.0.0-alpha.2"

	unstampedChart = "apiVersion: v2\nname: monitoring-x\nversion: 0.1.0\n"

	stampedChart = unstampedChart + "annotations:\n  openeverest.io/contract-version: \"" + stamp + "\"\n"

	defaultValues = "class:\n  create: true\n  name: x\n"

	// monitoringClass is the MonitoringClass of an integration chart, which
	// carries the stamp through the helper.
	monitoringClass = `{{- if .Values.class.create }}
apiVersion: monitoring.openeverest.io/v1alpha1
kind: MonitoringClass
metadata:
  name: {{ .Values.class.name }}
  annotations:
    {{- include "openeverest.contract.annotations" . | nindent 4 }}
spec:
  controllerName: example.com/x
{{- end }}
`

	deployment = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: x
`
)

// writeChart writes files, by path relative to the chart, into a new
// chart directory and returns it.
func writeChart(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		file := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o750))
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))
	}
	return dir
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name))) //nolint:gosec // the test's own file
	require.NoError(t, err)
	return string(data)
}

func TestStampChart(t *testing.T) {
	t.Parallel()

	dir := writeChart(t, map[string]string{"Chart.yaml": unstampedChart})

	res, err := classgen.StampChart(dir, "2.0.0-alpha.1")
	require.NoError(t, err)
	assert.Equal(t, classgen.ChartStamp{Written: []string{"Chart.yaml", "templates/" + classgen.ChartHelperFile}}, res)
	assert.Equal(t, unstampedChart+"annotations:\n  openeverest.io/contract-version: \"2.0.0-alpha.1\"\n", readFile(t, dir, "Chart.yaml"))
	assert.Equal(t, classgen.ChartHelper, readFile(t, dir, "templates/"+classgen.ChartHelperFile))

	res, err = classgen.StampChart(dir, "2.0.0-alpha.1")
	require.NoError(t, err)
	assert.Equal(t, classgen.ChartStamp{Previous: "2.0.0-alpha.1"}, res, "a stamped chart is left as it is")

	res, err = classgen.StampChart(dir, stamp)
	require.NoError(t, err)
	assert.Equal(t, classgen.ChartStamp{Previous: "2.0.0-alpha.1", Written: []string{"Chart.yaml"}}, res)
	assert.Equal(t, stampedChart, readFile(t, dir, "Chart.yaml"))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", classgen.ChartHelperFile), []byte("edited"), 0o600))
	res, err = classgen.StampChart(dir, stamp)
	require.NoError(t, err)
	assert.Equal(t, classgen.ChartStamp{Previous: stamp, Written: []string{"templates/" + classgen.ChartHelperFile}}, res)
	assert.Equal(t, classgen.ChartHelper, readFile(t, dir, "templates/"+classgen.ChartHelperFile))
}

func TestStampChartErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files   map[string]string
		version string
		wantErr string
	}{
		"invalid version": {
			files:   map[string]string{"Chart.yaml": unstampedChart},
			version: "v2.0.0",
			wantErr: `contract version "v2.0.0" is not a SemVer 2.0.0 version`,
		},
		"no Chart.yaml": {
			version: stamp,
			wantErr: "reading",
		},
		"Chart.yaml not a mapping": {
			files:   map[string]string{"Chart.yaml": "- a\n"},
			version: stamp,
			wantErr: "not a YAML mapping",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := writeChart(t, tt.files)

			_, err := classgen.StampChart(dir, tt.version)
			require.ErrorContains(t, err, tt.wantErr)
			assert.NoFileExists(t, filepath.Join(dir, "templates", classgen.ChartHelperFile))
		})
	}
}

func TestVerifyChart(t *testing.T) {
	t.Parallel()

	const (
		class    = "MonitoringClass x (templates/monitoringclass.yaml)"
		helper   = "templates/" + classgen.ChartHelperFile
		stampKey = "openeverest.io/contract-version"
		rerun    = "run `go tool classgen stamp`"
		include  = `include "openeverest.contract.annotations", from templates/_openeverest_contract.tpl, in its metadata.annotations`
	)
	// chart returns the files of a stamped chart with the MonitoringClass,
	// with the files of overrides over them. An empty override removes the
	// file.
	chart := func(overrides map[string]string) map[string]string {
		files := map[string]string{
			"Chart.yaml":                     stampedChart,
			"values.yaml":                    defaultValues,
			helper:                           classgen.ChartHelper,
			"templates/monitoringclass.yaml": monitoringClass,
			"templates/deployment.yaml":      deployment,
			"templates/NOTES.txt":            "Installed.\n",
		}
		for name, content := range overrides {
			if content == "" {
				delete(files, name)
				continue
			}
			files[name] = content
		}
		return files
	}

	tests := map[string]struct {
		files        map[string]string
		wantStamped  []string
		wantProblems []string
	}{
		"stamped": {
			files:       chart(nil),
			wantStamped: []string{"Chart.yaml", class},
		},
		"several objects in one template": {
			files: chart(map[string]string{
				"templates/monitoringclass.yaml": monitoringClass + "---\n" + `apiVersion: backup.openeverest.io/v1alpha1
kind: BackupStorageClass
metadata:
  name: s3
  annotations:
    {{- include "openeverest.contract.annotations" . | nindent 4 }}
`,
			}),
			wantStamped: []string{"Chart.yaml", class, "BackupStorageClass s3 (templates/monitoringclass.yaml)"},
		},
		"objects of subcharts are not checked": {
			files: chart(map[string]string{
				"charts/sub/Chart.yaml":                     "apiVersion: v2\nname: sub\nversion: 0.1.0\n",
				"charts/sub/templates/monitoringclass.yaml": "apiVersion: monitoring.openeverest.io/v1alpha1\nkind: MonitoringClass\nmetadata:\n  name: sub\n",
			}),
			wantStamped: []string{"Chart.yaml", class},
		},
		"Chart.yaml not stamped": {
			files:       chart(map[string]string{"Chart.yaml": unstampedChart}),
			wantStamped: nil,
			wantProblems: []string{
				"Chart.yaml has no " + stampKey + " annotation; " + rerun,
				class + " has no " + stampKey + " annotation; " + include,
			},
		},
		"stamp of another core": {
			files: chart(map[string]string{"Chart.yaml": unstampedChart + "annotations:\n  openeverest.io/contract-version: 2.0.0-alpha.1\n"}),
			wantProblems: []string{
				"Chart.yaml is stamped 2.0.0-alpha.1, but the integration is built against " + stamp + "; " + rerun,
				class + " is stamped 2.0.0-alpha.1, but the integration is built against " + stamp + "; " + rerun,
			},
		},
		"helper missing": {
			files:        chart(map[string]string{helper: ""}),
			wantStamped:  []string{"Chart.yaml", class},
			wantProblems: []string{helper + " is missing; " + rerun},
		},
		"not stamped yet": {
			files:       chart(map[string]string{"Chart.yaml": unstampedChart, helper: ""}),
			wantStamped: nil,
			wantProblems: []string{
				"Chart.yaml has no " + stampKey + " annotation; " + rerun,
				helper + " is missing; " + rerun,
				class + " has no " + stampKey + " annotation; " + include,
			},
		},
		"helper edited": {
			files: chart(map[string]string{helper: classgen.ChartHelper + "\n{{/* edited */}}\n"}),
			wantStamped: []string{
				"Chart.yaml", class,
			},
			wantProblems: []string{helper + " is not the one this classgen writes; " + rerun},
		},
		"object without the helper": {
			files:       chart(map[string]string{"templates/monitoringclass.yaml": "apiVersion: monitoring.openeverest.io/v1alpha1\nkind: MonitoringClass\nmetadata:\n  name: x\n"}),
			wantStamped: []string{"Chart.yaml"},
			wantProblems: []string{
				class + " has no " + stampKey + " annotation; " + include,
			},
		},
		"stamp written by hand": {
			files: chart(map[string]string{
				"templates/monitoringclass.yaml": "apiVersion: monitoring.openeverest.io/v1alpha1\nkind: MonitoringClass\nmetadata:\n  name: x\n  annotations:\n    openeverest.io/contract-version: \"2.0.0\"\n",
			}),
			wantStamped: []string{"Chart.yaml"},
			wantProblems: []string{
				class + " is stamped 2.0.0, but the integration is built against " + stamp + "; " + include + " instead of setting the annotation",
				class + " has no openeverest.io/compatibility-acknowledged annotation when compatibility.acknowledge.coreVersion is set, " +
					"so a user cannot run it on an OpenEverest that does not support it; " + include,
			},
		},
		"acknowledgement not rendered": {
			files: chart(map[string]string{
				"templates/monitoringclass.yaml": "apiVersion: monitoring.openeverest.io/v1alpha1\nkind: MonitoringClass\nmetadata:\n  name: x\n  annotations:\n    openeverest.io/contract-version: {{ index .Chart.Annotations \"openeverest.io/contract-version\" | quote }}\n",
			}),
			wantStamped: []string{"Chart.yaml", class},
			wantProblems: []string{
				class + " has no openeverest.io/compatibility-acknowledged annotation when compatibility.acknowledge.coreVersion is set, " +
					"so a user cannot run it on an OpenEverest that does not support it; " + include,
			},
		},
		"no object OpenEverest interprets": {
			files:        chart(map[string]string{"values.yaml": "class:\n  create: false\n"}),
			wantStamped:  []string{"Chart.yaml"},
			wantProblems: []string{"the chart renders no object OpenEverest interprets with its default values, so none carries the stamp"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			res, err := classgen.VerifyChart(writeChart(t, tt.files), stamp)

			require.NoError(t, err)
			assert.Equal(t, tt.wantStamped, res.Stamped)
			assert.Equal(t, tt.wantProblems, res.Problems)
		})
	}
}

func TestVerifyChartErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files   map[string]string
		wantErr string
	}{
		"no Chart.yaml": {
			files:   map[string]string{"values.yaml": defaultValues},
			wantErr: "loading",
		},
		"template does not render": {
			files: map[string]string{
				"Chart.yaml":            stampedChart,
				"templates/broken.yaml": "{{ fail \"broken\" }}\n",
			},
			wantErr: "broken",
		},
		"invalid object": {
			files: map[string]string{
				"Chart.yaml":           stampedChart,
				"templates/class.yaml": "apiVersion: a/b/c\nkind: MonitoringClass\nmetadata:\n  name: x\n",
			},
			wantErr: "templates/class.yaml: MonitoringClass x",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := classgen.VerifyChart(writeChart(t, tt.files), stamp)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestChartHelper renders the annotations the helper gives an object.
func TestChartHelper(t *testing.T) {
	t.Parallel()

	const ack = "compatibility:\n  acknowledge:\n    coreVersion: 2.0.0\n"
	tests := map[string]struct {
		chart  string
		values string
		want   map[string]string
	}{
		"stamp": {
			chart:  stampedChart,
			values: defaultValues,
			want:   map[string]string{"openeverest.io/contract-version": stamp},
		},
		"stamp and acknowledgement": {
			chart:  stampedChart,
			values: defaultValues + ack,
			want: map[string]string{
				"openeverest.io/contract-version":           stamp,
				"openeverest.io/compatibility-acknowledged": "2.0.0",
			},
		},
		"empty acknowledgement": {
			chart:  stampedChart,
			values: defaultValues + "compatibility:\n  acknowledge:\n    coreVersion: \"\"\n",
			want:   map[string]string{"openeverest.io/contract-version": stamp},
		},
		"compatibility not a map": {
			chart:  stampedChart,
			values: defaultValues + "compatibility: none\n",
			want:   map[string]string{"openeverest.io/contract-version": stamp},
		},
		"no stamp": {
			chart:  unstampedChart,
			values: defaultValues + ack,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := writeChart(t, map[string]string{
				"Chart.yaml":                            tt.chart,
				"values.yaml":                           tt.values,
				"templates/" + classgen.ChartHelperFile: classgen.ChartHelper,
				"templates/monitoringclass.yaml":        monitoringClass,
			})
			chrt, err := loader.Load(dir)
			require.NoError(t, err)
			vals, err := chartutil.ToRenderValues(chrt, make(map[string]any), chartutil.ReleaseOptions{Name: "x", Namespace: "default", IsInstall: true}, chartutil.DefaultCapabilities)
			require.NoError(t, err)

			files, err := engine.Render(chrt, vals)
			require.NoError(t, err)

			var class struct {
				Metadata struct {
					Annotations map[string]string `json:"annotations"`
				} `json:"metadata"`
			}
			require.NoError(t, yaml.Unmarshal([]byte(files["monitoring-x/templates/monitoringclass.yaml"]), &class))
			assert.Equal(t, tt.want, class.Metadata.Annotations)
		})
	}
}
