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

import { useEffect, useMemo } from 'react';
import { useFormContext, useFormState } from 'react-hook-form';
import type { Section } from 'components/ui-generator/ui-generator.types';
import type { OutsideFieldError } from './outside-field-errors.types';
import {
  collectOutsideFieldErrors,
  indexSchemaFields,
} from './outside-field-errors.utils';

export const useOutsideFieldErrors = (
  sectionKey: string,
  sections: Record<string, Section>
): OutsideFieldError[] => {
  const { control, trigger } = useFormContext();
  const { errors } = useFormState({ control });

  // The form knows it is invalid from the start but records no error until a
  // field changes, so Save would be disabled with nothing to explain it. The
  // schema is rebuilt with the sections (e.g. once storage classes load).
  useEffect(() => {
    void trigger();
  }, [trigger, sections]);

  const schemaFields = useMemo(
    () => indexSchemaFields(sections, sectionKey),
    [sections, sectionKey]
  );

  // react-hook-form updates `errors` in place, so memoizing on it goes stale.
  return collectOutsideFieldErrors(errors, schemaFields);
};
