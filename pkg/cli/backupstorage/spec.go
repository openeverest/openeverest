// everest
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

package backupstorage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openeverest/openeverest/v2/pkg/cli"
)

// The s3 parameters the shorthand flags set.
const (
	paramRegion         = "region"
	paramEndpointURL    = "endpointURL"
	paramForcePathStyle = "forcePathStyle"
	paramImplementation = "implementation"
)

// defaultCAKey is the Secret key a CA bundle is read from when --ca-secret names none.
const defaultCAKey = "ca.crt"

// ParameterOptions are the spec.parameters inputs of create and update. The
// shorthands are nil when their flag was not passed.
type ParameterOptions struct {
	Parameters     []string
	Region         *string
	EndpointURL    *string
	ForcePathStyle *bool
	Implementation *string
}

type shorthandParameter struct {
	flag  string
	key   string
	value any
}

// build merges the --parameter entries with the shorthand flags. A key named
// twice, through either, is an error.
func (o ParameterOptions) build() (map[string]any, error) {
	params := make(map[string]any, len(o.Parameters))
	for _, entry := range o.Parameters {
		key, raw, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --%s %q: must be in the form key=value", cli.FlagBackupStorageParameter, entry)
		}
		if _, dup := params[key]; dup {
			return nil, fmt.Errorf("--%s %s is set more than once", cli.FlagBackupStorageParameter, key)
		}
		params[key] = parseParameterValue(raw)
	}
	for _, s := range o.shorthands() {
		if _, dup := params[s.key]; dup {
			return nil, fmt.Errorf("--%s conflicts with --%s %s", s.flag, cli.FlagBackupStorageParameter, s.key)
		}
		params[s.key] = s.value
	}
	return params, nil
}

func (o ParameterOptions) shorthands() []shorthandParameter {
	var set []shorthandParameter
	if o.Region != nil {
		set = append(set, shorthandParameter{cli.FlagBackupStorageRegion, paramRegion, *o.Region})
	}
	if o.EndpointURL != nil {
		set = append(set, shorthandParameter{cli.FlagBackupStorageEndpointURL, paramEndpointURL, *o.EndpointURL})
	}
	if o.ForcePathStyle != nil {
		set = append(set, shorthandParameter{cli.FlagBackupStorageForcePathStyle, paramForcePathStyle, *o.ForcePathStyle})
	}
	if o.Implementation != nil {
		set = append(set, shorthandParameter{cli.FlagBackupStorageImplementation, paramImplementation, *o.Implementation})
	}
	return set
}

// parseParameterValue keeps a JSON boolean or number as that type; anything
// else is the raw string.
func parseParameterValue(raw string) any {
	var v any
	if strings.TrimSpace(raw) != raw || json.Unmarshal([]byte(raw), &v) != nil {
		return raw
	}
	switch v.(type) {
	case bool:
		return v
	case float64:
		return json.Number(raw)
	default:
		return raw
	}
}

// parseCASecret splits a --ca-secret NAME[:KEY] value.
func parseCASecret(ref string) (string, string, error) {
	name, key, found := strings.Cut(ref, ":")
	if !found {
		key = defaultCAKey
	}
	if name == "" || key == "" {
		return "", "", fmt.Errorf("invalid --%s %q: must be NAME or NAME:KEY", cli.FlagBackupStorageCASecret, ref)
	}
	return name, key, nil
}
