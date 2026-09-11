// everest
// Copyright (C) 2023 Percona LLC
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

// Package version provides methods to determine the correct version of components.
package version

import (
	"encoding/json"
	"strings"

	version "github.com/hashicorp/go-version"
)

var (
	// ProjectName is a component name, e.g. everest API Server, everestctl.
	ProjectName string //nolint:gochecknoglobals
	// Version is a component version e.g. v0.3.0-1-a93bef.
	Version string //nolint:gochecknoglobals
	// FullCommit is a git commit hash.
	FullCommit string //nolint:gochecknoglobals
	// EverestChannelOverride overrides the default olm channel for Everest operator.
	EverestChannelOverride string //nolint:gochecknoglobals
)

// IsRC checks if the version is a release candidate.
func IsRC(v string) bool {
	if v == "" {
		return false
	}
	ver, err := version.NewVersion(v)
	if err != nil {
		return false
	}
	return strings.Contains(ver.Prerelease(), "rc")
}

// IsDev checks if the version is a development version.
func IsDev(v string) bool {
	if v == "" {
		return false
	}
	ver, err := version.NewVersion(v)
	if err != nil {
		return false
	}
	devLatestVer := version.Must(version.NewVersion("v0.0.0"))
	return ver.Core().Equal(devLatestVer)
}

// Info represents the version information.
type Info struct {
	ProjectName   string  `json:"projectName"`
	Version       string  `json:"version"`
	FullCommit    string  `json:"fullCommit"`
	ServerVersion *string `json:"serverVersion,omitempty"`
}

// String returns the string representation of the version information.
func (v Info) String() string {
	out := []string{
		"ProjectName: " + v.ProjectName,
		"Version: " + v.Version,
		"FullCommit: " + v.FullCommit,
	}
	if v.ServerVersion != nil {
		out = append(out, "ServerVersion: "+*v.ServerVersion)
	}
	return strings.Join(out, "\n")
}

// JSONString returns the json representation of the version information.
func (v Info) JSONString() string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
