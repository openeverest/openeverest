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

// This is needed for correct reading of Zod and CEL validation errors.
// Example: { "spec.replica.nodes": 3 } => { spec: { replica: { nodes: 3 } } }
// Form ids are always dot-separated (see toFormId); parsePath still works and
// also accepts any accidental bracket-quoted flat keys.

import { parsePath } from 'components/ui-generator/utils/object-path';

export const convertToNestedObject = (
  flatObj: Record<string, unknown>
): Record<string, unknown> => {
  const result: Record<string, unknown> = {};

  Object.entries(flatObj).forEach(([path, value]) => {
    const keys = parsePath(path);
    let current: Record<string, unknown> = result;

    keys.forEach((key, index) => {
      if (index === keys.length - 1) {
        // Last key - set the value
        current[key] = value;
      } else {
        // Intermediate key - ensure object exists
        if (!current[key] || typeof current[key] !== 'object') {
          current[key] = {};
        }
        current = current[key] as Record<string, unknown>;
      }
    });
  });

  return result;
};
