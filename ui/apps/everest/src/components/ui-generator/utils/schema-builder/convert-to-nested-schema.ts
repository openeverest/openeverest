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

import { z } from 'zod';

export const convertToNestedSchema = (
  flatSchema: Record<string, z.ZodTypeAny>
): Record<string, z.ZodTypeAny> => {
  const nested: Record<string, unknown> = {};

  // Build nested structure from flat paths
  Object.entries(flatSchema).forEach(([path, zodType]) => {
    const keys = path.split('.');
    let current: Record<string, unknown> = nested;

    keys.forEach((key, index) => {
      if (index === keys.length - 1) {
        // Last key - set the Zod type
        current[key] = zodType;
      } else {
        // Intermediate key - ensure object exists
        if (!current[key]) {
          current[key] = {};
        }
        current = current[key] as Record<string, unknown>;
      }
    });
  });

  // Convert nested objects to Zod schemas
  return Object.fromEntries(
    Object.entries(nested).map(([key, value]) => [
      key,
      typeof value === 'object' &&
      value !== null &&
      !(value instanceof z.ZodType)
        ? convertToZodRecursively(value)
        : value,
    ])
  ) as Record<string, z.ZodTypeAny>;
};

const convertToZodRecursively = (obj: unknown): z.ZodTypeAny => {
  if (!obj || typeof obj !== 'object' || Array.isArray(obj)) {
    throw new Error('Expected object for Zod schema conversion');
  }

  const shape: Record<string, z.ZodTypeAny> = {};

  Object.entries(obj as Record<string, unknown>).forEach(([key, value]) => {
    if (value && typeof value === 'object' && !(value instanceof z.ZodType)) {
      // Nested object - convert recursively
      shape[key] = convertToZodRecursively(value);
    } else {
      // Zod type - use as-is
      shape[key] = value as z.ZodTypeAny;
    }
  });

  // passthrough() preserves keys not declared in this shape.
  // This is critical for section-edit mode where the Zod shape only covers
  // one section's fields but CEL superRefine needs access to fields from
  // other sections (e.g. spec.topology.config.shards).
  return z.object(shape).passthrough();
};
