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

import { renderHook } from '@testing-library/react';
import { FieldType, FormMode, TopologyUISchemas } from '../ui-generator.types';
import { preprocessSchema } from '../utils/preprocess/preprocess-schema';
import { useFormEngine } from './use-form-engine';

const SCHEMA: TopologyUISchemas = {
  single: {
    sections: {
      base: {
        components: {
          users: {
            uiType: FieldType.Text,
            path: 'spec.userSecretRef.name',
            fieldParams: { label: 'Users' },
            validation: { modes: { [FormMode.Import]: { required: true } } },
          },
        },
      },
    },
  },
};

describe('useFormEngine', () => {
  it.each([
    [FormMode.Import, false],
    [FormMode.New, true],
  ])('validates with the rules of the %s mode', (formMode, valid) => {
    const uiSchema = preprocessSchema(SCHEMA, undefined, formMode);
    const { result } = renderHook(() =>
      useFormEngine({ uiSchema, selectedTopology: 'single', formMode })
    );

    expect(result.current.zodSchema.safeParse({}).success).toBe(valid);
  });
});
