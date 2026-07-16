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
import { applyModeOverrides } from './apply-mode-overrides';
import {
  FieldType,
  Section,
  Component,
  ComponentGroup,
  FormMode,
} from '../../ui-generator.types';

const makeComponent = (overrides: Partial<Component> = {}): Component =>
  ({
    uiType: FieldType.Text,
    path: 'spec.name',
    fieldParams: { label: 'Name' },
    ...overrides,
  }) as Component;

describe('applyModeOverrides', () => {
  it('returns components unchanged when no modes are defined', () => {
    const sections: Record<string, Section> = {
      basic: {
        label: 'Basic',
        components: { name: makeComponent() },
      },
    };

    const result = applyModeOverrides(sections, FormMode.Edit);
    const comp = result.basic.components.name as Component;
    expect(comp.uiType).toBe(FieldType.Text);
    expect(comp.fieldParams.disabled).toBeUndefined();
  });

  it('disables a component via fieldParams.modes', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent({
            fieldParams: {
              label: 'Name',
              modes: { [FormMode.Edit]: { disabled: true } },
            },
          }),
        },
      },
    };

    const result = applyModeOverrides(sections, FormMode.Edit);
    const comp = result.basic.components.name as Component;
    expect(comp.fieldParams.disabled).toBe(true);
  });

  it('does not disable when a different mode is active', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent({
            fieldParams: {
              label: 'Name',
              modes: { [FormMode.Edit]: { disabled: true } },
            },
          }),
        },
      },
    };

    const result = applyModeOverrides(sections, FormMode.New);
    const comp = result.basic.components.name as Component;
    expect(comp.fieldParams.disabled).toBeUndefined();
  });

  it('hides a component via component.modes uiType override and strips CEL', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          secret: makeComponent({
            modes: { [FormMode.Edit]: { uiType: 'hidden' } },
            validation: {
              required: true,
              celExpressions: [
                { celExpr: 'self.length > 0', message: 'required' },
              ],
            },
          }),
        },
      },
    };

    const result = applyModeOverrides(sections, FormMode.Edit);
    const comp = result.basic.components.secret as Component;
    expect(comp.uiType).toBe('hidden');
    expect(comp.validation?.celExpressions).toBeUndefined();
    expect(comp.validation?.required).toBe(true);
  });

  it('component.modes uiType=hidden takes precedence over fieldParams.modes disabled', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent({
            modes: { [FormMode.Edit]: { uiType: 'hidden' } },
            fieldParams: {
              label: 'Name',
              modes: { [FormMode.Edit]: { disabled: true } },
            },
          }),
        },
      },
    };

    const result = applyModeOverrides(sections, FormMode.Edit);
    const comp = result.basic.components.name as Component;
    expect(comp.uiType).toBe('hidden');
  });

  it('processes nested groups recursively', () => {
    const group: ComponentGroup = {
      uiType: 'group',
      components: {
        inner: makeComponent({
          fieldParams: {
            label: 'Inner',
            modes: { [FormMode.Edit]: { disabled: true } },
          },
        }),
      },
    };
    const sections: Record<string, Section> = {
      resources: { components: { group } },
    };

    const result = applyModeOverrides(sections, FormMode.Edit);
    const g = result.resources.components.group as ComponentGroup;
    const inner = g.components.inner as Component;
    expect(inner.fieldParams.disabled).toBe(true);
  });

  it('applies readOnly from fieldParams.modes', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent({
            fieldParams: {
              label: 'Name',
              modes: { [FormMode.Edit]: { readOnly: true } },
            },
          } as Partial<Component>),
        },
      },
    };

    const result = applyModeOverrides(sections, FormMode.Edit);
    const comp = result.basic.components.name as Component;
    expect((comp.fieldParams as Record<string, unknown>).readOnly).toBe(true);
  });

  it('applies label override from fieldParams.modes', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent({
            fieldParams: {
              label: 'Name',
              modes: { [FormMode.Edit]: { label: 'Name (read-only)' } },
            },
          }),
        },
      },
    };

    const result = applyModeOverrides(sections, FormMode.Edit);
    const comp = result.basic.components.name as Component;
    expect(comp.fieldParams.label).toBe('Name (read-only)');
  });

  it('applies helperText override from fieldParams.modes', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent({
            fieldParams: {
              label: 'Name',
              helperText: 'Enter a name',
              modes: {
                [FormMode.Edit]: {
                  helperText: 'Name cannot be changed after creation',
                },
              },
            },
          }),
        },
      },
    };

    const result = applyModeOverrides(sections, FormMode.Edit);
    const comp = result.basic.components.name as Component;
    expect(comp.fieldParams.helperText).toBe(
      'Name cannot be changed after creation'
    );
  });

  it('applies defaultValue override from fieldParams.modes', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent({
            fieldParams: {
              label: 'Name',
              defaultValue: 'default-new',
              modes: {
                [FormMode.Edit]: { defaultValue: 'default-edit' },
              },
            },
          }),
        },
      },
    };

    const result = applyModeOverrides(sections, FormMode.Edit);
    const comp = result.basic.components.name as Component;
    expect(comp.fieldParams.defaultValue).toBe('default-edit');
  });

  it('applies autoFocus override from fieldParams.modes', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeComponent({
            fieldParams: {
              label: 'Name',
              modes: { [FormMode.Edit]: { autoFocus: true } },
            },
          }),
        },
      },
    };

    const result = applyModeOverrides(sections, FormMode.Edit);
    const comp = result.basic.components.name as Component;
    expect(comp.fieldParams.autoFocus).toBe(true);
  });
});
