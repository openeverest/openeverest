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
  FormMode,
} from 'components/ui-generator/ui-generator.types';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { extractInstanceValues } from '../default-values/extract-instance-values';
import { deepClone, setByPath } from '../object-path';
import { postprocessSchemaData } from './postprocess-schema';
import { keepUntouchedSourceValues } from './field-changes';

const schema = preprocessSchema({
  t: {
    sections: {
      resources: {
        components: {
          cpu: {
            uiType: FieldType.Number,
            path: 'spec.cpu',
            fieldParams: { label: 'CPU' },
          },
          replicas: {
            uiType: FieldType.Number,
            path: 'spec.replicas',
            fieldParams: { label: 'Replicas' },
          },
        },
      },
    },
  },
});
const { sections } = schema.t;
const source = { spec: { cpu: '500m', replicas: 3 } };

const restorePayload = (changes: Record<string, unknown> = {}) => {
  const initialValues = extractInstanceValues(
    sections,
    source,
    FormMode.Restore
  );
  const formData = deepClone(initialValues);
  Object.entries(changes).forEach(([path, value]) =>
    setByPath(formData, path, value)
  );
  const payload = postprocessSchemaData(formData, {
    schema,
    selectedTopology: 't',
  });
  return keepUntouchedSourceValues(payload, source, {
    sections,
    formData,
    initialValues,
  });
};

describe('keepUntouchedSourceValues', () => {
  it('sends a field the user left as read in the source form', () => {
    expect(restorePayload({ 'spec.replicas': 5 })).toEqual({
      spec: { cpu: '500m', replicas: 5 },
    });
  });

  it('sends a changed field as entered', () => {
    expect(restorePayload({ 'spec.cpu': 2 })).toEqual({
      spec: { cpu: 2, replicas: 3 },
    });
  });

  it('does not bring back a value the payload dropped', () => {
    const payload = { spec: { replicas: 3 } };
    const initialValues = extractInstanceValues(
      sections,
      source,
      FormMode.Restore
    );
    expect(
      keepUntouchedSourceValues(payload, source, {
        sections,
        formData: initialValues,
        initialValues,
      })
    ).toEqual({ spec: { replicas: 3 } });
  });
});
