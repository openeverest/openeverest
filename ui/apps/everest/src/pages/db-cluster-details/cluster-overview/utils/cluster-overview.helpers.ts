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
  WidgetComponent,
} from 'components/ui-generator/ui-generator.types';
import { isWidgetComponent } from 'components/ui-generator/ui-generator.types';
import type {
  WidgetSummary,
  WidgetSummaryRegistry,
} from 'components/ui-generator/widget-summary-registry';
import {
  getByPath,
  formatDisplayValue,
} from 'components/ui-generator/utils/object-path';
import {
  getComponentSourcePath,
  getComponentTargetPaths,
} from 'components/ui-generator/utils/preprocess/normalized-component';
import { stripBadgeFromValue } from 'components/ui-generator/utils/badge-to-api/badge-to-api';
import {
  getToggleableMeta,
  isToggleableOnInInstance,
} from 'components/ui-generator/utils/toggleable/toggleable';
import { Messages } from '../cluster-overview.messages';

export type SectionField = {
  label: string;
  path: string;
  value: string;
  // When set, the field renders its own read-only widget summary full-width
  // instead of the scalar `value` row.
  summary?: {
    Component: WidgetSummary;
    value: unknown;
    item: WidgetComponent;
  };
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
  componentsOrder?: string[],
  summaryRegistry?: WidgetSummaryRegistry
): SectionField[] => {
  const fields: SectionField[] = [];
  const keys = componentsOrder ?? Object.keys(components);

  for (const key of keys) {
    const comp = components[key];
    if (!comp) continue;

    if (comp.uiType === 'group' || comp.uiType === 'hidden') {
      const group = comp as ComponentGroup;
      const toggleable = getToggleableMeta(group);
      if (toggleable && !isToggleableOnInInstance(toggleable, instance)) {
        fields.push({
          label: group.label || key,
          path: toggleable.switchName,
          value: Messages.fields.disabled,
        });
        continue;
      }
      if (group.components) {
        fields.push(
          ...collectSectionFields(
            group.components,
            instance,
            group.componentsOrder,
            summaryRegistry
          )
        );
      }
      continue;
    }

    const component = comp as Component;

    // Widget components (e.g. affinity) hold structured values that don't
    // flatten to a scalar row; delegate to their registered read-only summary.
    if (isWidgetComponent(component)) {
      const path = getComponentSourcePath(component);
      const Summary = summaryRegistry?.[component.uiType];
      if (!path || !Summary) continue;

      fields.push({
        label: component.fieldParams?.label ?? key,
        path,
        value: '',
        summary: {
          Component: Summary,
          value: getByPath(instance, path),
          item: component,
        },
      });
      continue;
    }

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
