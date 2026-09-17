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

import { describe, expect, it } from 'vitest';
import { collectSectionFields } from './cluster-overview.helpers';
import type {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import { FieldType } from 'components/ui-generator/ui-generator.types';

const numberField = (
  path: string,
  label: string,
  badge?: string
): Component => ({
  uiType: FieldType.Number,
  path,
  fieldParams: { label, ...(badge ? { badge, badgeToApi: true } : {}) },
});

const components: Record<string, Component | ComponentGroup> = {
  cpu: numberField('spec.components.engine.resources.limits.cpu', 'CPU'),
  memory: numberField(
    'spec.components.engine.resources.limits.memory',
    'Memory',
    'Gi'
  ),
  disk: numberField('spec.components.engine.storage.size', 'Disk', 'Gi'),
};

describe('collectSectionFields', () => {
  it('renders a badged field in its badge unit, converting Kubernetes-normalised quantities', () => {
    const instance = {
      spec: {
        components: {
          engine: {
            resources: { limits: { cpu: '1', memory: '644245094400m' } },
            storage: { size: '25Gi' },
          },
        },
      },
    };

    const fields = collectSectionFields(components, instance, [
      'cpu',
      'memory',
      'disk',
    ]);

    expect(fields).toEqual([
      {
        label: 'CPU',
        path: 'spec.components.engine.resources.limits.cpu',
        value: '1',
      },
      {
        label: 'Memory',
        path: 'spec.components.engine.resources.limits.memory',
        value: '0.6Gi',
      },
      {
        label: 'Disk',
        path: 'spec.components.engine.storage.size',
        value: '25Gi',
      },
    ]);
  });

  it('leaves non-standard units untouched', () => {
    const instance = {
      spec: {
        components: {
          engine: {
            resources: { limits: { cpu: '1', memory: '16kg' } },
            storage: { size: '25Gi' },
          },
        },
      },
    };

    const fields = collectSectionFields(components, instance, ['memory']);

    expect(fields[0].value).toBe('16kg');
  });
});
