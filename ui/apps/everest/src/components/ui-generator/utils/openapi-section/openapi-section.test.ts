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
  ArrayItemType,
  ArrayOutputType,
  FieldType,
} from 'components/ui-generator/ui-generator.types';
import { sectionFromOpenApiSchema } from './openapi-section';

// The Prometheus class's destinationParametersSchema, plus shapes the
// generator can't edit.
const prometheusSchema = {
  type: 'object',
  required: ['interval'],
  properties: {
    podMonitorLabels: {
      type: 'object',
      additionalProperties: { type: 'string' },
    },
    interval: { type: 'string', description: 'Scrape interval' },
    replicas: { type: 'integer' },
    tls: { type: 'object', properties: { ca: { type: 'string' } } },
  },
};

describe('sectionFromOpenApiSchema', () => {
  it('maps strings to text, string maps to arrays, the rest to skipped', () => {
    expect(sectionFromOpenApiSchema(prometheusSchema, 'parameters')).toEqual({
      section: {
        components: {
          podMonitorLabels: {
            uiType: FieldType.Array,
            outputType: ArrayOutputType.Object,
            path: 'parameters.podMonitorLabels',
            fieldParams: { label: 'podMonitorLabels', helperText: undefined },
            validation: undefined,
            itemTemplate: { uiType: ArrayItemType.KeyValue },
          },
          interval: {
            uiType: FieldType.Text,
            path: 'parameters.interval',
            fieldParams: { label: 'interval', helperText: 'Scrape interval' },
            validation: { required: true },
          },
        },
        componentsOrder: ['podMonitorLabels', 'interval'],
      },
      skippedProperties: ['replicas', 'tls'],
    });
  });

  it('has no section without an object schema or editable properties', () => {
    expect(sectionFromOpenApiSchema(undefined, 'parameters')).toEqual({
      section: undefined,
      skippedProperties: [],
    });
    expect(
      sectionFromOpenApiSchema(
        { type: 'object', properties: { replicas: { type: 'integer' } } },
        'parameters'
      )
    ).toEqual({ section: undefined, skippedProperties: ['replicas'] });
  });
});
