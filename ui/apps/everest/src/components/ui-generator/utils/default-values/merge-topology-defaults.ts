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
