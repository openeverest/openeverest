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

import { useEffect, useRef } from 'react';
import { useFormContext } from 'react-hook-form';
import {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import { buildDefaultsFromComponents } from 'components/ui-generator/utils/default-values/build-defaults-from-components';

interface ApplySchemaDefaultsOptions {
  // When true, a default is written only where the field is still undefined, so
  // previously saved values survive. When false, schema defaults always win.
  onlyWhenEmpty?: boolean;
}

// Applies the schema-declared defaults for a set of UIGenerator components once
// per distinct `schemaKey` (typically the backup class name), then re-validates
// the form. The ref guard prevents re-applying on every render.
export const useApplySchemaDefaults = (
  components: Record<string, Component | ComponentGroup> | undefined,
  schemaKey: string,
  { onlyWhenEmpty = false }: ApplySchemaDefaultsOptions = {}
) => {
  const { setValue, getValues, trigger } = useFormContext();
  const appliedKeyRef = useRef<string>('');

  useEffect(() => {
    if (!schemaKey || appliedKeyRef.current === schemaKey) return;
    if (!components) return;

    const defaults = buildDefaultsFromComponents(components, '', true);
    Object.entries(defaults).forEach(([fieldName, value]) => {
      if (onlyWhenEmpty && getValues(fieldName) !== undefined) return;
      setValue(fieldName, value, {
        shouldDirty: false,
        shouldTouch: false,
        shouldValidate: false,
      });
    });

    appliedKeyRef.current = schemaKey;
    trigger();
  }, [components, schemaKey, onlyWhenEmpty, setValue, getValues, trigger]);
};
