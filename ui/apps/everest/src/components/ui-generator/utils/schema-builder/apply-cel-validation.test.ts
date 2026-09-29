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
              },
            },
          },
          retention: {
            uiType: FieldType.Number,
            path: 'spec.backup.retention',
            fieldParams: { label: 'Retention' },
            validation: {
              celExpressions: [
                {
                  celExpr:
                    '!has(spec.monitoring.interval) || spec.backup.retention >= spec.monitoring.interval',
                  message: 'Retention must cover the monitoring interval',
                },
              ],
            },
          },
        },
      },
    },
  },
});

const parse = (switchOn: boolean) =>
  buildZodSchema(schema, 'replicaSet').schema.safeParse({
    [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~monitoring': switchOn },
    spec: { monitoring: { interval: 100 }, backup: { retention: 10 } },
  });

describe('CEL on a field outside a toggleable group', () => {
  it('sees a switched-off group as absent, like the payload', () => {
    expect(parse(false).success).toBe(true);
  });

  it('sees the group values once it is switched on', () => {
    const result = parse(true);
    expect(result.success).toBe(false);
    expect(
      result.success ? [] : result.error.issues.map((issue) => issue.message)
    ).toContain('Retention must cover the monitoring interval');
  });
});
