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
import { FieldType, Section } from '../../ui-generator.types';
import {
  readValuesForCel,
  withEmptyFormValues,
  withoutEmptyFormValues,
} from './field-codecs';

describe('readValuesForCel', () => {
  it('adds no value where none is held, whatever the field would read it as', () => {
    const values = { spec: { name: 'db' } };

    expect(
      readValuesForCel(values, [
        { path: 'spec.proxy.labels', read: () => ({ team: 'db' }) },
      ])
    ).toEqual(values);
  });

  it('leaves the values it was given untouched', () => {
    const values = { spec: { labels: [{ key: 'team', value: 'db' }] } };

    readValuesForCel(values, [{ path: 'spec.labels', read: () => undefined }]);

    expect(values).toEqual({
      spec: { labels: [{ key: 'team', value: 'db' }] },
    });
  });
});

describe('the values with and without what a field with no value holds', () => {
  const sections: Record<string, Section> = {
    advanced: {
      components: {
        args: {
          uiType: FieldType.List,
          path: 'spec.args',
          fieldParams: {},
          itemTemplate: { uiType: FieldType.Text, fieldParams: {} },
        },
      },
    },
  };

  it('are new values: the ones given stay as they were', () => {
    const unset = { spec: { name: 'db' } };
    const held = withEmptyFormValues(unset, sections);

    expect(unset).toEqual({ spec: { name: 'db' } });

    const heldBefore = structuredClone(held);
    expect(withoutEmptyFormValues(held, sections)).toEqual(unset);
    expect(held).toEqual(heldBefore);
  });
});
