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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime/debug"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/openeverest/openeverest/v2/contract"
)

// coreModule is the module path of OpenEverest core, which classgen is
// part of.
const coreModule = "github.com/openeverest/openeverest/v2"

// module is a module version, with the module that replaces it, if any. It
// decodes go list -json, whose field names are capitalised.
type module struct {
	Path    string  `json:"Path"`              //nolint:tagliatelle // go list -json
	Version string  `json:"Version,omitempty"` //nolint:tagliatelle // go list -json
	Replace *module `json:"Replace,omitempty"` //nolint:tagliatelle // go list -json
	// Dir is where the code of the module is.
	Dir string `json:"Dir,omitempty"` //nolint:tagliatelle // go list -json
}

func (m module) String() string {
	s := m.Path
	if m.Version != "" {
		s += "@" + m.Version
	}
	if m.Replace != nil {
		s += " => " + m.Replace.String()
	}
	return s
}

// local reports whether the code of m comes from a directory, as with a
// replace directive or a go.work workspace, so its version says nothing
// about the code.
func (m module) local() bool {
	if m.Replace != nil {
		return modfile.IsDirectoryPath(m.Replace.Path)
	}
	return m.Version == "" || m.Version == "(devel)"
}

// effective returns the module whose code is built for m.
func (m module) effective() module {
	if m.Replace != nil {
		return *m.Replace
	}
	return m
}

// linkedCore returns the core module this binary was built with.
func linkedCore() (module, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return module{}, errors.New("this classgen binary carries no build information")
	}
	// classgen is a package of core, so core is the main module of the
	// binary, even when `go tool` builds it as a dependency of another.
	for _, m := range append([]*debug.Module{&info.Main}, info.Deps...) {
		if m.Path != coreModule {
			continue
		}
		linked := module{Path: m.Path, Version: m.Version}
		if m.Replace != nil {
			linked.Replace = &module{Path: m.Replace.Path, Version: m.Replace.Version}
		}
		return linked, nil
	}
	return module{}, fmt.Errorf("this classgen binary does not link %s", coreModule)
}

// requiredCore returns the core module the Go module in dir builds with, as
// `go list -m` selects it.
func requiredCore(ctx context.Context, dir string) (module, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", coreModule)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return module{}, fmt.Errorf("finding the %s this module requires (go list -m): %w: %s; "+
			"run classgen as `go tool classgen` from a module that requires it as a tool (go get -tool %s/pkg/classgen/cmd/classgen@<version>)",
			coreModule, err, strings.TrimSpace(stderr.String()), coreModule)
	}
	var m module
	if err := json.Unmarshal(out, &m); err != nil {
		return module{}, fmt.Errorf("decoding go list -m %s: %w", coreModule, err)
	}
	return m, nil
}

// checkCore compares the core this binary links with the one the module
// classgen runs in requires. What classgen writes depends on that core: the
// stamp is its contract version, and the schemas are rendered by its code.
// It returns an error when they differ, as classgen would then write for an
// OpenEverest the module is not built against. When both come from a local
// directory their versions cannot be compared, so it returns a warning to
// print instead: under `go tool` both are the same directory.
func checkCore(linked, required module) (string, error) {
	switch {
	case linked.local() && required.local():
		dir := required.Dir
		if dir == "" {
			dir = required.effective().Path
		}
		return fmt.Sprintf("%s comes from a local directory (%s), so the stamp is the contract version of that tree, %s",
			coreModule, dir, contract.Version), nil
	case !linked.local() && !required.local():
		l, r := linked.effective(), required.effective()
		if l.Path == r.Path && l.Version == r.Version {
			return "", nil
		}
	}
	return "", fmt.Errorf("this classgen links %s, but this module requires %s, so it would write for another OpenEverest; "+
		"run it as `go tool classgen` from the module, which builds it against the core the module requires", linked, required)
}

// checkLinkedCore returns the core the module in the working directory
// requires, after checking it is the one this binary links, and a warning
// to print, if any.
func checkLinkedCore(ctx context.Context) (module, string, error) {
	linked, err := linkedCore()
	if err != nil {
		return module{}, "", err
	}
	required, err := requiredCore(ctx, ".")
	if err != nil {
		return module{}, "", err
	}
	warning, err := checkCore(linked, required)
	if err != nil {
		return module{}, "", err
	}
	return required, warning, nil
}

// releaseProblems returns why an integration built against required, and
// stamped with version, must not be released. A release tells users the
// oldest OpenEverest it works with, so that must be an OpenEverest they can
// install: the stamp must be a released contract version. The core commit
// itself may be unreleased: while the contract of OpenEverest's main branch
// is still that of a release, the stamp names that release.
func releaseProblems(required module, version string) []string {
	var problems []string
	if contract.IsDevelopment(version) {
		problems = append(problems, fmt.Sprintf("built against an unreleased OpenEverest (%s): its contract has changed since the last release; "+
			"require an OpenEverest whose contract is a released one", version))
	}
	switch {
	case required.Replace != nil:
		problems = append(problems, fmt.Sprintf("go.mod replaces %s (%s); require it directly instead", coreModule, required))
	case required.local():
		problems = append(problems, coreModule+" comes from a local directory; require it in go.mod instead")
	}
	return problems
}

// floor describes, to the integration's author, the oldest OpenEverest a
// build stamped with version works with.
func floor(version string) string {
	if contract.IsDevelopment(version) {
		return fmt.Sprintf("Warning: built against an unreleased OpenEverest (%s); it cannot be published to the Hub until that OpenEverest is released.", version)
	}
	return fmt.Sprintf("This build works with OpenEverest %s and later.", version)
}
