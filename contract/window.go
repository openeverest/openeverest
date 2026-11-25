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

package contract

// Window is the range of contract versions a core serves, from Min to
// Current inclusive.
type Window struct {
	Min     string
	Current string
}

// WindowFromAnnotations returns the window published in the annotations of a
// core CRD or Chart.yaml. ok is false when neither MinVersionAnnotation nor
// VersionAnnotation is present: the object predates published windows. A
// window with only one of them, or with an empty value, is returned as it is,
// and Check reports it as InvalidVersion.
func WindowFromAnnotations(annotations map[string]string) (Window, bool) {
	minimum, hasMin := annotations[MinVersionAnnotation]
	current, hasCurrent := annotations[VersionAnnotation]
	if !hasMin && !hasCurrent {
		return Window{}, false
	}
	return Window{Min: minimum, Current: current}, true
}

// Intersect returns the window every one of ws serves: the highest Min and
// the lowest Current. A reader that sees several core CRDs, some of them
// possibly synced ahead of the others, thus never claims more than every one
// of them serves. An invalid value in any window carries over to the result,
// so Check reports it instead of a valid value hiding it, and windows that do
// not overlap give an empty window, which Check reports as CoreVersionSkew.
// The intersection of no windows is the zero Window, which is invalid.
func Intersect(ws ...Window) Window {
	if len(ws) == 0 {
		return Window{}
	}
	out := ws[0]
	for _, w := range ws[1:] {
		out.Min = higher(out.Min, w.Min)
		out.Current = lower(out.Current, w.Current)
	}
	return out
}

// Check reports whether an extension stamped with ext can run on a core that
// serves w.
func (w Window) Check(ext string) Result {
	return Check(ext, w.Min, w.Current)
}

func higher(a, b string) string {
	switch {
	case !Valid(a):
		return a
	case !Valid(b), Compare(b, a) > 0:
		return b
	default:
		return a
	}
}

func lower(a, b string) string {
	switch {
	case !Valid(a):
		return a
	case !Valid(b), Compare(b, a) < 0:
		return b
	default:
		return a
	}
}
