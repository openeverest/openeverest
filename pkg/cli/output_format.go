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

package cli

import "fmt"

// OutputFormat is the value of -o/--output: json or yaml.
type OutputFormat string

const (
	// OutputJSON is machine-readable JSON.
	OutputJSON OutputFormat = "json"
	// OutputYAML is machine-readable YAML.
	OutputYAML OutputFormat = "yaml"
)

// String returns the format, or "" when unset (pretty mode).
func (o *OutputFormat) String() string {
	if o == nil {
		return ""
	}
	return string(*o)
}

// Set parses s as an OutputFormat for pflag. Only "json" and "yaml" are valid.
func (o *OutputFormat) Set(s string) error {
	switch OutputFormat(s) {
	case OutputJSON, OutputYAML:
		*o = OutputFormat(s)
		return nil
	default:
		return fmt.Errorf(`invalid output format %q: must be "json" or "yaml"`, s)
	}
}

// Type is the pflag type name shown in help.
func (*OutputFormat) Type() string {
	return "string"
}
