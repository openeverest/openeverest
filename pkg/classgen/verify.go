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
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/releaseutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
)

// coreInterprets reports whether OpenEverest interprets the objects of the
// kind gk an extension applies. Each must carry the stamp of the core the
// extension was built against, as OpenEverest decides from it whether it
// can serve the object.
func coreInterprets(gk schema.GroupKind) bool {
	switch gk {
	case schema.GroupKind{Group: monitoringv1alpha1.GroupVersion.Group, Kind: "MonitoringClass"},
		schema.GroupKind{Group: backupv1alpha1.GroupVersion.Group, Kind: "BackupStorageClass"},
		schema.GroupKind{Group: backupv1alpha1.GroupVersion.Group, Kind: "BackupClass"},
		schema.GroupKind{Group: corev1alpha1.GroupVersion.Group, Kind: "Provider"},
		schema.GroupKind{Group: corev1alpha1.GroupVersion.Group, Kind: "ClusterInstancePreset"},
		schema.GroupKind{Group: extensionsv1alpha1.GroupVersion.Group, Kind: "Plugin"}:
		return true
	default:
		return false
	}
}

const (
	// acknowledgeProbe is the release VerifyChart acknowledges to check
	// that the acknowledgement reaches every stamped object. Any release
	// does: only whether it reaches them matters.
	acknowledgeProbe = "1.0.0"

	// restamp is the fix of a stamp or a helper that is missing or stale.
	restamp = "run `go tool classgen stamp`"

	// includeHelper is the fix of an object that does not carry the stamp,
	// or the acknowledgement.
	includeHelper = `include "` + AnnotationsHelper + `", from templates/` + ChartHelperFile + `, in its metadata.annotations`
)

// ChartReport is the outcome of VerifyChart.
type ChartReport struct {
	// Stamped lists what carries the expected stamp: Chart.yaml and the
	// objects OpenEverest interprets.
	Stamped []string
	// Problems lists what is wrong. The chart passes when it is empty.
	Problems []string
}

// VerifyChart checks the stamp of the chart in chartDir against version,
// the contract version of the core the integration is built against:
//
//   - Chart.yaml carries it;
//   - templates/ChartHelperFile is ChartHelper;
//   - every object OpenEverest interprets that the chart's own templates
//     render with its default values carries it, and the chart renders at
//     least one;
//   - with compatibility.acknowledge.coreVersion set, every stamped object
//     carries the acknowledgement too, since the startup gate and
//     OpenEverest read it on the object.
//
// The objects of subcharts are not checked: they belong to other charts,
// with stamps of their own. Nothing is read from a cluster, so lookup
// returns nothing. A missing helper is reported, and the chart rendered with
// ChartHelper. VerifyChart returns an error only when it cannot check, such
// as when the chart does not render.
func VerifyChart(chartDir, version string) (ChartReport, error) {
	var res ChartReport

	chartFile := filepath.Join(chartDir, "Chart.yaml")
	meta, err := chartutil.LoadChartfile(chartFile)
	if err != nil {
		return res, fmt.Errorf("loading %s: %w", chartFile, err)
	}
	chartStamp, stamped := meta.Annotations[contract.VersionAnnotation]
	switch {
	case !stamped:
		res.Problems = append(res.Problems, fmt.Sprintf("Chart.yaml has no %s annotation; %s", contract.VersionAnnotation, restamp))
	case chartStamp != version:
		res.Problems = append(res.Problems, fmt.Sprintf("Chart.yaml is stamped %s, but the integration is built against %s; %s", chartStamp, version, restamp))
	default:
		res.Stamped = append(res.Stamped, "Chart.yaml")
	}

	problem, err := helperProblem(chartDir)
	if err != nil {
		return res, err
	}
	if problem != "" {
		res.Problems = append(res.Problems, problem)
	}

	objects, err := renderChart(chartDir, nil)
	if err != nil {
		return res, err
	}
	for _, o := range objects {
		problem := stampProblem(o, version, chartStamp)
		if problem == "" {
			res.Stamped = append(res.Stamped, o.String())
			continue
		}
		res.Problems = append(res.Problems, problem)
	}
	if len(objects) == 0 {
		res.Problems = append(res.Problems, "the chart renders no object OpenEverest interprets with its default values, so none carries the stamp")
	}

	acknowledged, err := renderChart(chartDir, map[string]any{
		"compatibility": map[string]any{"acknowledge": map[string]any{"coreVersion": acknowledgeProbe}},
	})
	if err != nil {
		return res, fmt.Errorf("rendering the chart with %s set: %w", contract.AcknowledgeValuePath, err)
	}
	for _, o := range acknowledged {
		// An object without the stamp is reported above.
		if _, ok := o.annotations[contract.VersionAnnotation]; !ok || o.annotations[contract.AcknowledgedAnnotation] == acknowledgeProbe {
			continue
		}
		msg := fmt.Sprintf("%s has no %s annotation when %s is set, so a user cannot run it on an OpenEverest that does not support it; %s",
			o, contract.AcknowledgedAnnotation, contract.AcknowledgeValuePath, includeHelper,
		)
		res.Problems = append(res.Problems, msg)
	}
	return res, nil
}

// helperProblem returns what is wrong with templates/ChartHelperFile of the
// chart in chartDir, or nothing.
func helperProblem(chartDir string) (string, error) {
	helper := "templates/" + ChartHelperFile
	got, err := os.ReadFile(filepath.Join(chartDir, "templates", ChartHelperFile)) //nolint:gosec // the chart the caller names
	switch {
	case errors.Is(err, os.ErrNotExist):
		return fmt.Sprintf("%s is missing; %s", helper, restamp), nil
	case err != nil:
		return "", fmt.Errorf("reading %s: %w", helper, err)
	case string(got) != ChartHelper:
		return fmt.Sprintf("%s is not the one this classgen writes; %s", helper, restamp), nil
	default:
		return "", nil
	}
}

// stampProblem returns what is wrong with the stamp of o, given version, the
// stamp it must carry, and chartStamp, the one in Chart.yaml, or nothing.
func stampProblem(o renderedObject, version, chartStamp string) string {
	stamp, ok := o.annotations[contract.VersionAnnotation]
	switch {
	case !ok:
		return fmt.Sprintf("%s has no %s annotation; %s", o, contract.VersionAnnotation, includeHelper)
	case stamp == version:
		return ""
	case stamp == chartStamp:
		return fmt.Sprintf("%s is stamped %s, but the integration is built against %s; %s", o, stamp, version, restamp)
	default:
		return fmt.Sprintf("%s is stamped %s, but the integration is built against %s; %s instead of setting the annotation", o, stamp, version, includeHelper)
	}
}

// renderedObject is an object of a kind OpenEverest interprets that a
// chart's own templates render.
type renderedObject struct {
	source      string
	groupKind   schema.GroupKind
	name        string
	annotations map[string]string
}

func (o renderedObject) String() string {
	return fmt.Sprintf("%s %s (%s)", o.groupKind.Kind, o.name, o.source)
}

// renderChart renders the chart in dir as a fresh install with values over
// its default values would, and returns the objects of the kinds
// OpenEverest interprets that its own templates render, in template order.
func renderChart(dir string, values map[string]any) ([]renderedObject, error) {
	if values == nil {
		values = make(map[string]any)
	}
	chrt, err := loader.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("loading chart %s: %w", dir, err)
	}
	// A chart whose helper is missing is reported as such; rendering it
	// with the helper still checks what its templates include.
	helper := "templates/" + ChartHelperFile
	if !slices.ContainsFunc(chrt.Templates, func(f *chart.File) bool { return f.Name == helper }) {
		chrt.Templates = append(chrt.Templates, &chart.File{Name: helper, Data: []byte(ChartHelper)})
	}
	if err := chartutil.ProcessDependenciesWithMerge(chrt, values); err != nil {
		return nil, fmt.Errorf("processing the dependencies of chart %s: %w", dir, err)
	}
	vals, err := chartutil.ToRenderValues(chrt, values, chartutil.ReleaseOptions{
		Name:      chrt.Name(),
		Namespace: "default",
		Revision:  1,
		IsInstall: true,
	}, chartutil.DefaultCapabilities,
	)
	if err != nil {
		return nil, fmt.Errorf("computing the values of chart %s: %w", dir, err)
	}
	files, err := engine.Render(chrt, vals)
	if err != nil {
		return nil, fmt.Errorf("rendering chart %s: %w", dir, err)
	}
	return ownObjects(chrt.Name(), files)
}

// ownObjects returns the objects of the kinds OpenEverest interprets in
// files, the templates of the chart named chartName as the engine renders
// them, that the chart's own templates render, in template order.
func ownObjects(chartName string, files map[string]string) ([]renderedObject, error) {
	// The engine names files after the chart, as in
	// monitoring-pmm/templates/monitoringclass.yaml, and those of subcharts
	// as in monitoring-pmm/charts/sub/templates/x.yaml.
	own := chartName + "/templates/"
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasPrefix(name, own) && path.Base(name) != "NOTES.txt" && !strings.HasPrefix(path.Base(name), "_") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var objects []renderedObject
	for _, name := range names {
		docs := releaseutil.SplitManifests(files[name])
		keys := make([]string, 0, len(docs))
		for k := range docs {
			keys = append(keys, k)
		}
		sort.Sort(releaseutil.BySplitManifestsOrder(keys))
		source := strings.TrimPrefix(name, chartName+"/")
		for _, k := range keys {
			o, err := decodeObject(docs[k])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", source, err)
			}
			if !coreInterprets(o.groupKind) {
				continue
			}
			o.source = source
			objects = append(objects, o)
		}
	}
	return objects, nil
}

func decodeObject(doc string) (renderedObject, error) {
	var head struct {
		APIVersion string `json:"apiVersion"`
		Kind       string `json:"kind"`
		Metadata   struct {
			Name        string            `json:"name"`
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(doc), &head); err != nil {
		return renderedObject{}, fmt.Errorf("decoding a rendered object: %w", err)
	}
	if head.Kind == "" {
		return renderedObject{}, nil
	}
	gv, err := schema.ParseGroupVersion(head.APIVersion)
	if err != nil {
		return renderedObject{}, fmt.Errorf("%s %s: %w", head.Kind, head.Metadata.Name, err)
	}
	return renderedObject{
		groupKind:   schema.GroupKind{Group: gv.Group, Kind: head.Kind},
		name:        head.Metadata.Name,
		annotations: head.Metadata.Annotations,
	}, nil
}
