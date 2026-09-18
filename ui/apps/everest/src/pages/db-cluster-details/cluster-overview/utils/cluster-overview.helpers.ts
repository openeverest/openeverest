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

import type {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import {
  getByPath,
  formatDisplayValue,
} from 'components/ui-generator/utils/object-path';
import { getComponentTargetPaths } from 'components/ui-generator/utils/preprocess/normalized-component';
import { stripBadgeFromValue } from 'components/ui-generator/utils/badge-to-api/badge-to-api';

export type SectionField = {
  label: string;
  path: string;
  value: string;
};

// Kubernetes may store a quantity in a different unit than the field's badge —
// e.g. it normalises "0.6Gi" to the milli-byte value "644245094400m" (#2423).
// Render badged fields in their badge unit (0.6Gi, 25Gi) instead of the raw
// stored quantity; leave unconvertible/non-standard values as-is.
const formatBadgedValue = (rawValue: unknown, badge?: string): string => {
  if (badge && typeof rawValue === 'string' && rawValue.trim() !== '') {
    const stripped = stripBadgeFromValue(rawValue, badge);
    if (typeof stripped === 'string' && Number.isFinite(Number(stripped))) {
      return `${stripped}${badge}`;
    }
  }
  return formatDisplayValue(rawValue);
};

export const collectSectionFields = (
  components: Record<string, Component | ComponentGroup>,
  instance: Record<string, unknown>,
  componentsOrder?: string[]
): SectionField[] => {
  const fields: SectionField[] = [];
  const keys = componentsOrder ?? Object.keys(components);

  for (const key of keys) {
    const comp = components[key];
    if (!comp) continue;

    if (comp.uiType === 'group' || comp.uiType === 'hidden') {
      const group = comp as ComponentGroup;
      if (group.components) {
        fields.push(
          ...collectSectionFields(
            group.components,
            instance,
            group.componentsOrder
          )
        );
      }
      continue;
    }

    const component = comp as Component;
    const paths = getComponentTargetPaths(component);
    const path = paths[0];
    if (!path) continue;

    fields.push({
      label: component.fieldParams?.label ?? key,
      path,
      value: formatBadgedValue(
        getByPath(instance, path),
        component.fieldParams?.badge
      ),
    });
  }

  return fields;
};
