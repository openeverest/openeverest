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

  return z.object(shape);
};
