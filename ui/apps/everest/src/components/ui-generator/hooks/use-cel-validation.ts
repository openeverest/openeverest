import { useEffect, useRef } from 'react';
import { Control, useWatch } from 'react-hook-form';

// React hook that triggers form re-validation. This is used with CEL expressions to ensure cross-field validation when related fields are modified.
export function useCelValidation(
  // array of dependency groups from buildZodSchema
  groups: string[][],
  control: Control<Record<string, unknown>>,
  trigger: (fields?: string | string[]) => void
) {
  const watchedNames = Array.from(new Set(groups.flat()));
  const watchedValues = useWatch({ control, name: watchedNames });

  const prevValuesRef = useRef<string | null>(null);

  useEffect(() => {
    if (groups.length === 0) return;

    const currentValues = JSON.stringify(watchedValues);

    // Skip on initial render
    if (prevValuesRef.current === null) {
      prevValuesRef.current = currentValues;
      return;
    }

    if (prevValuesRef.current !== currentValues) {
      prevValuesRef.current = currentValues;

      // Trigger validation for all groups
      groups.forEach((group) => {
        trigger(group);
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...watchedValues]);
}
