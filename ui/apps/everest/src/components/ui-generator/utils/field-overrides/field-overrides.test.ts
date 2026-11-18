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

import {
  Component,
  ComponentGroup,
  FieldType,
  Section,
  WidgetType,
} from 'components/ui-generator/ui-generator.types';
import { applyFieldOverrides } from './field-overrides';

const sections: Record<string, Section> = {
  location: {
    components: {
      bucket: {
        uiType: FieldType.Text,
        path: 'location.bucket',
        fieldParams: { label: 'Bucket', helperText: 'Where backups go' },
      },
      advanced: {
        uiType: 'group',
        components: {
          prefix: {
            uiType: FieldType.Text,
            path: ['location.prefix', 'location.legacyPrefix'],
            fieldParams: { label: 'Prefix' },
          },
        },
      },
      scheduling: {
        uiType: 'widget',
        widgetType: WidgetType.PodSchedulingPolicy,
        path: 'location.scheduling',
      },
    },
  },
};

const paramsOf = (item: Component | ComponentGroup | undefined) =>
  item && 'fieldParams' in item ? item.fieldParams : undefined;
const bucketParams = (result: Record<string, Section>) =>
  paramsOf(result.location.components.bucket);
const prefixParams = (result: Record<string, Section>) => {
  const group = result.location.components.advanced;
  return paramsOf('components' in group ? group.components.prefix : undefined);
};

describe('applyFieldOverrides', () => {
  it('overrides a field in a group by its path, keeping its own params', () => {
    const result = applyFieldOverrides(sections, {
      'location.bucket': { disabled: true, tooltip: 'Fixed once created' },
    });

    expect(bucketParams(result)).toEqual({
      label: 'Bucket',
      helperText: 'Where backups go',
      disabled: true,
      tooltip: 'Fixed once created',
    });
    expect(prefixParams(result)).toEqual({ label: 'Prefix' });
  });

  it('finds a field writing several paths by the first only', () => {
    const byFirst = applyFieldOverrides(sections, {
      'location.prefix': { disabled: true },
    });
    const byOther = applyFieldOverrides(sections, {
      'location.legacyPrefix': { disabled: true },
    });

    expect(prefixParams(byFirst)).toMatchObject({ disabled: true });
    expect(prefixParams(byOther)).not.toHaveProperty('disabled');
  });

  it('leaves a widget, which takes no field params, as it is', () => {
    const result = applyFieldOverrides(sections, {
      'location.scheduling': { disabled: true },
    });

    expect(result.location.components.scheduling).toBe(
      sections.location.components.scheduling
    );
  });

  it('returns the same sections when there is nothing to override', () => {
    expect(applyFieldOverrides(sections, {})).toBe(sections);
  });
});
