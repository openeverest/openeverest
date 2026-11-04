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

import { z } from 'zod';
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  vi,
  type MockInstance,
} from 'vitest';
import {
  Component,
  ComponentGroup,
  FieldType,
  FormMode,
  GroupType,
  Section,
  ToggleableMeta,
} from 'components/ui-generator/ui-generator.types';
import { buildDefaultsFromComponents } from '../default-values/build-defaults-from-components';
import { extractInstanceValues } from '../default-values/extract-instance-values';
import { postprocessSchemaData } from '../postprocess/postprocess-schema';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { applyModeOverrides } from '../preprocess/apply-mode-overrides';
import { buildShapeFromComponents } from '../schema-builder/build-shape-from-components';
import { convertToNestedSchema } from '../schema-builder/convert-to-nested-schema';
import {
  TOGGLEABLE_SWITCHES_KEY,
  collectToggleableMetas,
  getInactiveToggleablePaths,
  getToggleableMeta,
  isToggleableOn,
  isToggleableOnInInstance,
} from './toggleable';

const endpoint = {
  uiType: FieldType.Text,
  path: 'spec.monitoring.configName',
  fieldParams: { label: 'Endpoint', defaultValue: 'default-endpoint' },
} satisfies Component;

const monitoring: ComponentGroup = {
  uiType: 'group',
  groupType: GroupType.Toggleable,
  label: 'Monitoring',
  components: {
    endpoint,
    interval: {
      uiType: FieldType.Number,
      path: 'spec.monitoring.interval',
      fieldParams: { label: 'Interval' },
    },
  },
};

const SWITCH = `${TOGGLEABLE_SWITCHES_KEY}.advanced~monitoring`;

const preprocess = (
  components: Record<string, Component | ComponentGroup>
): Record<string, Section> =>
  preprocessSchema({ replicaSet: { sections: { advanced: { components } } } })
    .replicaSet.sections;

const groupAt = (
  sections: Record<string, Section>,
  ...keys: string[]
): ComponentGroup => {
  let node: Component | ComponentGroup = {
    uiType: 'group',
    components: sections.advanced.components,
  };
  for (const key of keys) {
    if (!('components' in node)) throw new Error(`${key} is not a group`);
    node = node.components[key];
  }
  if (!('components' in node)) throw new Error('expected a group');
  return node;
};

const sections = preprocess({ monitoring });

const metaOf = (group: ComponentGroup): ToggleableMeta => {
  const meta = getToggleableMeta(group);
  if (!meta) throw new Error('expected toggleable meta');
  return meta;
};

describe('toggleable group switch', () => {
  it('names the switch after the group position and lists its paths', () => {
    expect(getToggleableMeta(groupAt(sections, 'monitoring'))).toEqual({
      switchName: SWITCH,
      childPaths: ['spec.monitoring.configName', 'spec.monitoring.interval'],
    });
  });

  it('keeps the same switch when a form mode hides the first field', () => {
    const hiddenFirst = preprocess({
      monitoring: {
        ...monitoring,
        components: {
          ...monitoring.components,
          endpoint: {
            ...endpoint,
            modes: { [FormMode.Edit]: { uiType: 'hidden' } },
          },
        },
      },
    });
    const edit = applyModeOverrides(hiddenFirst, FormMode.Edit);

    expect(metaOf(groupAt(edit, 'monitoring')).switchName).toBe(SWITCH);
  });

  it('starts off for a new instance even when children have schema defaults', () => {
    const defaults = buildDefaultsFromComponents(
      sections.advanced.components,
      'advanced'
    );
    expect(defaults[SWITCH]).toBe(false);
    expect(defaults['spec.monitoring.configName']).toBe('default-endpoint');
  });

  it('adds no switch when applying only schema-declared defaults', () => {
    const defaults = buildDefaultsFromComponents(
      sections.advanced.components,
      '',
      true
    );
    expect(defaults).not.toHaveProperty(SWITCH);
  });

  it('is on when the saved instance has any section field set', () => {
    const meta = metaOf(groupAt(sections, 'monitoring'));
    const instance = { spec: { monitoring: { interval: 30 } } };
    expect(isToggleableOnInInstance(meta, instance)).toBe(true);

    const values = extractInstanceValues(sections, instance, FormMode.Edit);
    expect(isToggleableOn(values, meta.switchName)).toBe(true);
  });

  it('is off when the saved instance has none of the section fields', () => {
    const values = extractInstanceValues(
      sections,
      { spec: { other: 1 } },
      FormMode.Edit
    );
    expect(
      isToggleableOn(values, metaOf(groupAt(sections, 'monitoring')).switchName)
    ).toBe(false);
  });

  it('keeps the form valid with or without the switches object', () => {
    const { schemaShape } = buildShapeFromComponents(
      sections.advanced.components
    );
    const schema = z.object(convertToNestedSchema(schemaShape)).passthrough();
    const spec = { monitoring: { configName: 'pmm' } };

    expect(schema.safeParse({ spec }).success).toBe(true);
    expect(
      schema.safeParse({
        spec,
        [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~monitoring': true },
      }).success
    ).toBe(true);
  });

  it('never sends the form-only switch to the API', () => {
    const formValues = {
      [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~monitoring': true },
      spec: { monitoring: { configName: 'pmm' } },
    };
    const result = postprocessSchemaData(formValues);
    expect(result).toEqual({ spec: { monitoring: { configName: 'pmm' } } });
    expect(formValues[TOGGLEABLE_SWITCHES_KEY]).toBeDefined();
  });
});

describe('toggleable groups that cannot own their paths', () => {
  const backup: ComponentGroup = {
    ...monitoring,
    components: {
      schedule: {
        uiType: FieldType.Text,
        path: 'spec.backup.schedule',
        fieldParams: { label: 'Schedule' },
      },
    },
  };

  let warn: MockInstance<typeof console.warn>;
  beforeEach(() => {
    warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
  });
  afterEach(() => warn.mockRestore());

  it('degrades a toggleable nested inside another toggleable to bordered', () => {
    const result = preprocess({
      outer: {
        uiType: 'group',
        groupType: GroupType.Toggleable,
        components: { inner: monitoring },
      },
    });

    expect(groupAt(result, 'outer').groupType).toBe(GroupType.Toggleable);
    expect(groupAt(result, 'outer', 'inner').groupType).toBe(
      GroupType.Bordered
    );
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('nested'));
  });

  it('degrades a toggleable sharing a path with a field outside it', () => {
    const result = preprocess({
      monitoring,
      endpointCopy: {
        uiType: FieldType.Text,
        path: 'spec.monitoring.interval',
        fieldParams: { label: 'Interval' },
      },
    });

    expect(groupAt(result, 'monitoring').groupType).toBe(GroupType.Bordered);
    expect(getToggleableMeta(groupAt(result, 'monitoring'))).toBeUndefined();
    expect(warn).toHaveBeenCalledWith(
      expect.stringContaining('spec.monitoring.interval')
    );
  });

  it('degrades both toggleables that share a path, whatever their order', () => {
    const result = preprocess({ first: monitoring, second: monitoring });

    expect(groupAt(result, 'first').groupType).toBe(GroupType.Bordered);
    expect(groupAt(result, 'second').groupType).toBe(GroupType.Bordered);
  });

  it('degrades a toggleable without path-bound fields', () => {
    const result = preprocess({
      scheduling: {
        uiType: 'group',
        groupType: GroupType.Toggleable,
        components: {
          note: {
            id: 'note',
            uiType: FieldType.Text,
            fieldParams: { label: 'Note' },
          },
        },
      },
    });

    expect(groupAt(result, 'scheduling').groupType).toBe(GroupType.Bordered);
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('path'));
  });

  it('degrades a toggleable whose key cannot name a switch', () => {
    const result = preprocess({ 'monitoring.v2': monitoring });

    expect(groupAt(result, 'monitoring.v2').groupType).toBe(GroupType.Bordered);
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('key'));
  });

  it('keeps independent toggleable groups side by side', () => {
    const result = preprocess({ monitoring, backup });

    expect(groupAt(result, 'monitoring').groupType).toBe(GroupType.Toggleable);
    expect(groupAt(result, 'backup').groupType).toBe(GroupType.Toggleable);
    expect(warn).not.toHaveBeenCalled();
  });
});

describe('switched-off toggleable paths', () => {
  const withMultiPath: ComponentGroup = {
    uiType: 'group',
    groupType: GroupType.Toggleable,
    components: {
      version: {
        uiType: FieldType.Text,
        path: ['spec.engine.version', 'spec.proxy.version'],
        fieldParams: { label: 'Version' },
      },
    },
  };

  it('lists every API path of switched-off groups, multi-path targets included', () => {
    const values = {
      [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~withMultiPath': false },
    };
    expect(
      getInactiveToggleablePaths(preprocess({ withMultiPath }), values)
    ).toEqual(['spec.engine.version', 'spec.proxy.version']);
  });

  it('ignores degraded groups so their visible fields are never dropped', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const nested = preprocess({
      outer: {
        uiType: 'group',
        groupType: GroupType.Toggleable,
        components: { inner: monitoring, withMultiPath },
      },
    });
    warn.mockRestore();

    expect(
      collectToggleableMetas(nested).map((meta) => meta.switchName)
    ).toEqual([`${TOGGLEABLE_SWITCHES_KEY}.advanced~outer`]);
  });
});

describe('toggleable group gate switch (#3290)', () => {
  const monitoringWithGate: ComponentGroup = {
    uiType: 'group',
    groupType: GroupType.Toggleable,
    label: 'Monitoring',
    components: {
      enabled: {
        uiType: FieldType.Toggle,
        path: 'spec.monitoring.enabled',
        gate: true,
        fieldParams: { label: 'Enable monitoring' },
      },
      interval: {
        uiType: FieldType.Number,
        path: 'spec.monitoring.interval',
        fieldParams: { label: 'Interval' },
      },
    },
    componentsOrder: ['enabled', 'interval'],
  };

  it('Gate Hoist: names switch after gate path and excludes gate path from childPaths', () => {
    const res = preprocess({ monitoring: monitoringWithGate });
    const meta = metaOf(groupAt(res, 'monitoring'));
    expect(meta.switchName).toBe('spec.monitoring.enabled');
    expect(meta.gatePath).toBe('spec.monitoring.enabled');
    expect(meta.gateKey).toBe('enabled');
    expect(meta.childPaths).toEqual(['spec.monitoring.interval']);
    // Verify gate key is removed from componentsOrder
    expect(groupAt(res, 'monitoring').componentsOrder).toEqual(['interval']);
  });

  describe('Gate Validation', () => {
    it('rejects gate: true on a non-toggle component', () => {
      expect(() =>
        preprocess({
          monitoring: {
            uiType: 'group',
            groupType: GroupType.Toggleable,
            components: {
              interval: {
                uiType: FieldType.Number,
                path: 'spec.monitoring.interval',
                gate: true,
                fieldParams: { label: 'Interval' },
              },
            },
          },
        })
      ).toThrowError(/Only toggle components can be gate switches/);
    });

    it('rejects gate: true outside a toggleable group (section root)', () => {
      expect(() =>
        preprocess({
          enabled: {
            uiType: FieldType.Toggle,
            path: 'spec.monitoring.enabled',
            gate: true,
            fieldParams: { label: 'Enable' },
          },
        })
      ).toThrowError(/placed outside a toggleable group/);
    });

    it('rejects gate: true in a bordered group', () => {
      expect(() =>
        preprocess({
          monitoring: {
            uiType: 'group',
            groupType: GroupType.Bordered,
            components: {
              enabled: {
                uiType: FieldType.Toggle,
                path: 'spec.monitoring.enabled',
                gate: true,
                fieldParams: { label: 'Enable' },
              },
            },
          },
        })
      ).toThrowError(/placed outside a toggleable group/);
    });

    it('rejects more than one gate per group', () => {
      expect(() =>
        preprocess({
          monitoring: {
            uiType: 'group',
            groupType: GroupType.Toggleable,
            components: {
              gate1: {
                uiType: FieldType.Toggle,
                path: 'spec.monitoring.gate1',
                gate: true,
                fieldParams: { label: 'Gate 1' },
              },
              gate2: {
                uiType: FieldType.Toggle,
                path: 'spec.monitoring.gate2',
                gate: true,
                fieldParams: { label: 'Gate 2' },
              },
            },
          },
        })
      ).toThrowError(/more than one gate component/);
    });
  });

  describe('Load from Gate', () => {
    const res = preprocess({ monitoring: monitoringWithGate });
    const meta = metaOf(groupAt(res, 'monitoring'));

    it('initializes switch to ON when instance has enabled: true', () => {
      const instance = {
        spec: { monitoring: { enabled: true, interval: 30 } },
      };
      expect(isToggleableOnInInstance(meta, instance)).toBe(true);

      const values = extractInstanceValues(res, instance, FormMode.Edit);
      expect(isToggleableOn(values, meta.switchName)).toBe(true);
    });

    it('initializes switch to OFF when instance has enabled: false', () => {
      const instance = {
        spec: { monitoring: { enabled: false, interval: 30 } },
      };
      expect(isToggleableOnInInstance(meta, instance)).toBe(false);

      const values = extractInstanceValues(res, instance, FormMode.Edit);
      expect(isToggleableOn(values, meta.switchName)).toBe(false);
    });
  });

  describe('Gate Kept When Off', () => {
    const res = preprocess({ monitoring: monitoringWithGate });

    it('deletes child values from payload when OFF but preserves gate switch false value', () => {
      const formValues = {
        spec: {
          monitoring: {
            enabled: false,
            interval: 30,
          },
          name: 'test-db',
        },
      };

      const result = postprocessSchemaData(formValues, {
        schema: { replicaSet: { sections: res } },
        selectedTopology: 'replicaSet',
      });

      expect(result).toEqual({
        spec: {
          monitoring: { enabled: false },
          name: 'test-db',
        },
      });
    });

    it('preserves both gate and child values when ON', () => {
      const formValues = {
        spec: {
          monitoring: {
            enabled: true,
            interval: 30,
          },
          name: 'test-db',
        },
      };

      const result = postprocessSchemaData(formValues, {
        schema: { replicaSet: { sections: res } },
        selectedTopology: 'replicaSet',
      });

      expect(result).toEqual({
        spec: {
          monitoring: { enabled: true, interval: 30 },
          name: 'test-db',
        },
      });
    });
  });
});
