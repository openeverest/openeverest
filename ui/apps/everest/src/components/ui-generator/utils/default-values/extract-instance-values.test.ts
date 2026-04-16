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

import { describe, it, expect } from 'vitest';
import { extractInstanceValues } from './extract-instance-values';
import {
  Section,
  Component,
  FieldType,
  FormMode,
} from 'components/ui-generator/ui-generator.types';
const makeComponent = (
  path: string,
  overrides: Partial<Component> = {}
): Component =>
  ({
    uiType: FieldType.Text,
    path,
    fieldParams: { label: path },
    _normalized: {
      sourcePath: path,
      targetPaths: [path],
    },
    ...overrides,
  }) as Component;

describe('extractInstanceValues', () => {
  it('extracts values from instance by sourcePath', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent('spec.name'),
        },
      },
    };

    const instance = { spec: { name: 'my-db' } };
    const result = extractInstanceValues(
      sections,
      instance as unknown as Record<string, unknown>
    );

    expect(result).toEqual({ spec: { name: 'my-db' } });
  });

  it('falls back to schema default when instance value is missing', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent('spec.name', {
            fieldParams: { label: 'Name', defaultValue: 'default-name' },
          }),
        },
      },
    };

    const instance = { spec: {} };
    const result = extractInstanceValues(
      sections,
      instance as unknown as Record<string, unknown>
    );

    expect(result).toEqual({ spec: { name: 'default-name' } });
  });

  it('extracts values from multiple sections', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent('spec.name'),
        },
      },
      resources: {
        components: {
          cpu: {
            uiType: FieldType.Number,
            path: 'spec.resources.cpu',
            fieldParams: { label: 'CPU' },
            _normalized: {
              sourcePath: 'spec.resources.cpu',
              targetPaths: ['spec.resources.cpu'],
            },
          } as Component,
        },
      },
    };

    const instance = {
      spec: { name: 'my-db', resources: { cpu: 4 } },
    };
    const result = extractInstanceValues(
      sections,
      instance as unknown as Record<string, unknown>
    );

    expect(result).toEqual({
      spec: { name: 'my-db', resources: { cpu: 4 } },
    });
  });

  it('strips badge suffixes from instance values when badgeToApi is enabled', () => {
    const sections: Record<string, Section> = {
      resources: {
        components: {
          disk: {
            uiType: FieldType.Number,
            path: 'spec.resources.disk',
            fieldParams: {
              label: 'Disk',
              badge: 'Gi',
              badgeToApi: true,
            },
            _normalized: {
              sourcePath: 'spec.resources.disk',
              targetPaths: ['spec.resources.disk'],
            },
          } as Component,
        },
      },
    };

    const instance = {
      spec: { resources: { disk: '25Gi' } },
    };

    const result = extractInstanceValues(
      sections,
      instance as unknown as Record<string, unknown>
    );

    expect(result).toEqual({
      spec: { resources: { disk: '25' } },
    });
  });

  it('returns empty object for empty sections', () => {
    const result = extractInstanceValues({}, {});
    expect(result).toEqual({});
  });

  it('skips schema defaults in Edit mode when instance value is missing', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent('spec.name', {
            fieldParams: { label: 'Name', defaultValue: 'default-name' },
          }),
          version: makeComponent('spec.version', {
            fieldParams: { label: 'Version', defaultValue: '8.0' },
          }),
        },
      },
    };

    const instance = { spec: { name: 'my-db' } };
    const result = extractInstanceValues(
      sections,
      instance as unknown as Record<string, unknown>,
      FormMode.Edit
    );

    // name comes from instance, version is missing — should NOT get default
    expect(result).toEqual({ spec: { name: 'my-db' } });
  });

  it('still uses schema defaults in New mode', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent('spec.name', {
            fieldParams: { label: 'Name', defaultValue: 'default-name' },
          }),
        },
      },
    };

    const instance = { spec: {} };
    const result = extractInstanceValues(
      sections,
      instance as unknown as Record<string, unknown>,
      FormMode.New
    );

    expect(result).toEqual({ spec: { name: 'default-name' } });
  });

  it('uses instance values in Edit mode when they exist', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent('spec.name'),
          cpu: makeComponent('spec.resources.cpu'),
        },
      },
    };

    const instance = { spec: { name: 'my-db', resources: { cpu: 4 } } };
    const result = extractInstanceValues(
      sections,
      instance as unknown as Record<string, unknown>,
      FormMode.Edit
    );

    expect(result).toEqual({
      spec: { name: 'my-db', resources: { cpu: 4 } },
    });
  });
});
