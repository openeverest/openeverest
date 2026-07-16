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
import { isSectionEditable } from './is-section-editable';
import {
  FormMode,
  FieldType,
  Section,
  Component,
  ComponentGroup,
} from '../../ui-generator.types';

const makeComponent = (overrides: Partial<Component> = {}): Component =>
  ({
    uiType: FieldType.Text,
    path: 'spec.name',
    fieldParams: { label: 'Name' },
    ...overrides,
  }) as Component;

describe('isSectionEditable', () => {
  it('returns true when section has editable fields', () => {
    const section: Section = {
      components: { name: makeComponent() },
    };
    expect(isSectionEditable(section, FormMode.Edit)).toBe(true);
  });

  it('returns false when all fields are hidden via component.modes', () => {
    const section: Section = {
      components: {
        name: makeComponent({
          modes: { [FormMode.Edit]: { uiType: 'hidden' } },
        }),
      },
    };
    expect(isSectionEditable(section, FormMode.Edit)).toBe(false);
  });

  it('returns false when all fields are disabled via fieldParams.modes', () => {
    const section: Section = {
      components: {
        name: makeComponent({
          fieldParams: {
            label: 'Name',
            modes: { [FormMode.Edit]: { disabled: true } },
          },
        }),
      },
    };
    expect(isSectionEditable(section, FormMode.Edit)).toBe(false);
  });

  it('returns false when all fields are readOnly via fieldParams.modes', () => {
    const section: Section = {
      components: {
        name: makeComponent({
          fieldParams: {
            label: 'Name',
            modes: { [FormMode.Edit]: { readOnly: true } },
          },
        } as Partial<Component>),
      },
    };
    expect(isSectionEditable(section, FormMode.Edit)).toBe(false);
  });

  it('returns false when field has base disabled=true (no mode override)', () => {
    const section: Section = {
      components: {
        name: makeComponent({
          fieldParams: { label: 'Name', disabled: true },
        }),
      },
    };
    expect(isSectionEditable(section, FormMode.Edit)).toBe(false);
  });

  it('returns false when field has base readOnly=true (no mode override)', () => {
    const section: Section = {
      components: {
        name: makeComponent({
          fieldParams: { label: 'Name', readOnly: true },
        } as Partial<Component>),
      },
    };
    expect(isSectionEditable(section, FormMode.Edit)).toBe(false);
  });

  it('returns true when at least one field is editable in a group', () => {
    const group: ComponentGroup = {
      uiType: 'group',
      components: {
        hidden: makeComponent({
          modes: { [FormMode.Edit]: { uiType: 'hidden' } },
        }),
        editable: makeComponent(),
      },
    };
    const section: Section = {
      components: { group },
    };
    expect(isSectionEditable(section, FormMode.Edit)).toBe(true);
  });

  it('returns true when overrides exist only for a different mode', () => {
    const section: Section = {
      components: {
        name: makeComponent({
          fieldParams: {
            label: 'Name',
            modes: { [FormMode.New]: { disabled: true } },
          },
        }),
      },
    };
    expect(isSectionEditable(section, FormMode.Edit)).toBe(true);
  });

  it('returns false for section with no components', () => {
    const section: Section = { components: {} };
    expect(isSectionEditable(section, FormMode.Edit)).toBe(false);
  });
});
