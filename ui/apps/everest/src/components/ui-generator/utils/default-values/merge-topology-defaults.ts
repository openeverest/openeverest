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

/** Deep-merge topology defaults into current values so every nested object
 *  is initialised before re-validation (prevents Zod parent-level errors). */
export const mergeTopologyDefaults = (
  current: Record<string, unknown>,
  defaults: Record<string, unknown>
): Record<string, unknown> => {
  const result: Record<string, unknown> = { ...current };
  for (const key of Object.keys(defaults)) {
    if (result[key] === undefined || result[key] === null) {
      result[key] = defaults[key];
    } else if (
      typeof defaults[key] === 'object' &&
      defaults[key] !== null &&
      !Array.isArray(defaults[key]) &&
      typeof result[key] === 'object' &&
      result[key] !== null &&
      !Array.isArray(result[key])
    ) {
      result[key] = mergeTopologyDefaults(
        result[key] as Record<string, unknown>,
        defaults[key] as Record<string, unknown>
      );
    }
  }
  return result;
};
