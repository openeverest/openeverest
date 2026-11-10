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

package backupstorageclasses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// DecodeParameters decodes the raw spec.parameters of a BackupStorage into v
// strictly: a key v has no field for is ErrUnsupportedParameter, any other
// decoding error ErrInvalidParameters. Empty input leaves v unchanged.
func DecodeParameters(raw []byte, v any) error {
	if len(raw) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if msg, ok := strings.CutPrefix(err.Error(), "json: "); ok && strings.HasPrefix(msg, "unknown field ") {
			return fmt.Errorf("%w: %s", ErrUnsupportedParameter, msg)
		}
		return fmt.Errorf("%w: %w", ErrInvalidParameters, err)
	}
	return nil
}
