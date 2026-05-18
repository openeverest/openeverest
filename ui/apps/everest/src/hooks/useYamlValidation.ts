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
import { parse, YAMLParseError } from 'yaml';

export interface YamlValidationResult {
  /** The parsed YAML as a JS value, or null on parse failure */
  parsed: unknown;
  /** Human-readable error message, or null if valid */
  error: string | null;
}

/**
 * Parses a YAML string and returns the parsed result or a validation error.
 *
 * Uses the `yaml` package (already a devDependency in the repo) for
 * standards-compliant YAML 1.2 parsing. Intentionally lightweight —
 * no JSON Schema validation is included in this PoC.
 */
export const useYamlValidation = (content: string): YamlValidationResult => {
  return useMemo(() => {
    if (!content.trim()) {
      return { parsed: null, error: null };
    }

    try {
      const parsed: unknown = parse(content);
      return { parsed, error: null };
    } catch (err) {
      if (err instanceof YAMLParseError) {
        return { parsed: null, error: err.message };
      }
      return {
        parsed: null,
        error: err instanceof Error ? err.message : 'Unknown YAML error',
      };
    }
  }, [content]);
};
