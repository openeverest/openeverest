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

package jobspec

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ReadFromFilepath reads the Spec from a JSON file at the specified filepath.
// Implements the dash convention: if filepath is "-", reads from stdin.
func (in *Spec) ReadFromFilepath(filepath string) error {
	var reader io.Reader
	if filepath == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(filepath) //nolint:gosec
		if err != nil {
			return fmt.Errorf("error opening config file: %w", err)
		}
		defer file.Close() //nolint:errcheck
		reader = file
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}
	return json.Unmarshal(data, in)
}
