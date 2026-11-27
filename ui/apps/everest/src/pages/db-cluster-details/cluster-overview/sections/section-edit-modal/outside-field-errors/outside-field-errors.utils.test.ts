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
import {
  FieldType,
  type Section,
} from 'components/ui-generator/ui-generator.types';
import {
  collectOutsideFieldErrors,
  indexSchemaFields,
} from './outside-field-errors.utils';

const sections: Record<string, Section> = {
  resources: {
    label: 'Resources',
    components: {
      size: {
        uiType: FieldType.Number,
        path: 'spec.storage.size',
        fieldParams: { label: 'Disk size' },
      },
    },
  },
  tls: {
    components: {
      mode: {
        uiType: FieldType.Text,
        path: 'spec.tls.mode',
        fieldParams: { label: 'Mode' },
      },
      issuer: {
        uiType: FieldType.Hidden,
        path: 'spec.tls.issuer',
        fieldParams: { label: 'Issuer' },
      },
    },
  },
};

const collect = (errors: unknown) =>
  collectOutsideFieldErrors(errors, indexSchemaFields(sections, 'tls'));

describe('collectOutsideFieldErrors', () => {
  it('keeps only the errors no input of the dialog shows', () => {
    expect(
      collect({
        spec: {
          storage: { size: { type: 'custom', message: 'Cannot shrink' } },
          tls: { mode: { type: 'too_small', message: 'Field is required' } },
        },
      })
    ).toEqual([
      {
        fieldId: 'spec.storage.size',
        message: 'Cannot shrink',
        sectionLabel: 'Resources',
        fieldLabel: 'Disk size',
      },
    ]);
  });

  it("reports a hidden field of the dialog's own section by its path", () => {
    expect(
      collect({
        spec: { tls: { issuer: { type: 'custom', message: 'No issuer' } } },
      })
    ).toEqual([{ fieldId: 'spec.tls.issuer', message: 'No issuer' }]);
  });
});
