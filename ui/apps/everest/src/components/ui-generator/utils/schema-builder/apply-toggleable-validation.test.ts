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
  GroupType,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { TOGGLEABLE_SWITCHES_KEY } from '../toggleable/toggleable';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { buildZodSchema } from './build-zod-schema';

const SWITCH = 'advanced~monitoring';

const schema: TopologyUISchemas = preprocessSchema({
  replicaSet: {
    sections: {
      advanced: {
        components: {
          monitoring: {
            uiType: 'group',
            groupType: GroupType.Toggleable,
            components: {
              interval: {
                uiType: FieldType.Number,
                path: 'spec.monitoring.interval',
                fieldParams: { label: 'Interval' },
                validation: {
                  required: true,
                  celExpressions: [
                    {
                      celExpr: 'spec.monitoring.interval >= 10',
                      message: 'Interval must be at least 10',
                    },
                  ],
                },
              },
            },
          },
        },
      },
    },
  },
});

const parse = (switchOn: boolean, interval: unknown) =>
  buildZodSchema(schema, 'replicaSet').schema.safeParse({
    [TOGGLEABLE_SWITCHES_KEY]: { [SWITCH]: switchOn },
    spec: { monitoring: { interval } },
  });

const messagesOf = (result: ReturnType<typeof parse>) =>
  result.success ? [] : result.error.issues.map((issue) => issue.message);

describe('toggleable group validation', () => {
  it('pauses required and CEL checks while the section is off', () => {
    expect(parse(false, undefined).success).toBe(true);
    expect(parse(false, 3).success).toBe(true);
  });

  it('runs CEL checks while the section is on', () => {
    expect(messagesOf(parse(true, 3))).toContain(
      'Interval must be at least 10'
    );
  });

  it('keeps field transforms for valid values', () => {
    const result = parse(true, '30');
    expect(result.success).toBe(true);
    expect(result.success && result.data.spec.monitoring.interval).toBe(30);
  });
});
