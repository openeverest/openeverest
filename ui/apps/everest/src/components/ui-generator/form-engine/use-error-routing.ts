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

import { useMemo } from 'react';
import { FieldErrors } from 'react-hook-form';

/**
 * Walks the react-hook-form `errors` object and collects all leaf
 * error paths as dot-separated strings.
 *
 * A leaf node is detected by the presence of `message` or `type`,
 * which is how RHF represents individual field errors.
 */
export const flattenErrorPaths = (
  obj: Record<string, unknown>,
  prefix = ''
): string[] => {
  if (!obj || typeof obj !== 'object') return prefix ? [prefix] : [];
  if (obj.message !== undefined || obj.type !== undefined)
    return prefix ? [prefix] : [];
  return Object.keys(obj).flatMap((key) =>
    flattenErrorPaths(
      obj[key] as Record<string, unknown>,
      prefix ? `${prefix}.${key}` : key
    )
  );
};

/**
 * Maps form validation errors to the step IDs that own the corresponding
 * fields.
 */
export const useErrorRouting = (
  // `formState.errors` from react-hook-form.
  errors: FieldErrors,
  // Map from field path → step ID.
  fieldToStepMap: Record<string, string>,
  // Set of step IDs that currently exist (used to filter out stale entries after topology changes).
  validStepIds: Set<string>
  // Array of step IDs that have at least one error.
): string[] =>
  useMemo(() => {
    const errorPaths = flattenErrorPaths(errors as Record<string, unknown>);
    const stepsSet = new Set<string>();

    for (const path of errorPaths) {
      if (typeof path !== 'string' || path.length === 0) continue;

      // Try full path first, then progressively shorter prefixes.
      const parts = path.split('.');
      for (let i = parts.length; i > 0; i--) {
        const prefix = parts.slice(0, i).join('.');
        if (prefix in fieldToStepMap) {
          stepsSet.add(fieldToStepMap[prefix]);
          break;
        }
      }
    }

    return Array.from(stepsSet).filter((id) => validStepIds.has(id));
  }, [errors, fieldToStepMap, validStepIds]);
