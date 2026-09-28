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
import { z } from 'zod';
import {
  Component,
  ComponentGroup,
  FieldType,
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { buildZodSchema, BuildSchemaOptions } from './build-zod-schema';

const TOPOLOGY = 'replicaSet';

const buildSchema = (
  components: Record<string, Component | ComponentGroup>,
  options?: BuildSchemaOptions
) => {
  const uiSchema: TopologyUISchemas = {
    [TOPOLOGY]: { sections: { main: { components } } },
  };
  return buildZodSchema(uiSchema, TOPOLOGY, options);
};

const issueMessagesAt = (
  result: z.SafeParseReturnType<unknown, unknown>,
  path: string
): string[] =>
  result.success
    ? []
    : result.error.issues
        .filter((issue) => issue.path.join('.') === path)
        .map((issue) => issue.message);

describe('CEL `self`', () => {
  it('resolves to the value of the field that declares the rule', () => {
    const { schema } = buildSchema({
      tier: {
        uiType: FieldType.Select,
        path: 'spec.tier',
        fieldParams: {
          options: [
            { label: 'Free', value: 'free' },
            { label: 'Pro', value: 'pro' },
          ],
        },
        validation: {
          celExpressions: [
            {
              celExpr: "self == 'pro' || spec.users < 10",
              message: 'Free tier is limited to 10 users',
            },
          ],
        },
      },
      users: { uiType: FieldType.Number, path: 'spec.users', fieldParams: {} },
    });

    const free = schema.safeParse({ spec: { tier: 'free', users: 20 } });
    expect(issueMessagesAt(free, 'spec.tier')).toEqual([
      'Free tier is limited to 10 users',
    ]);

    const pro = schema.safeParse({ spec: { tier: 'pro', users: 20 } });
    expect(pro.success).toBe(true);
  });

  it('sees a number field as a number, not the raw input string', () => {
    const { schema } = buildSchema({
      replicas: {
        uiType: FieldType.Number,
        path: 'spec.replicas',
        fieldParams: {},
        validation: {
          celExpressions: [
            { celExpr: 'self % 2 == 1', message: 'Must be odd' },
          ],
        },
      },
    });

    expect(schema.safeParse({ spec: { replicas: '3' } }).success).toBe(true);
    expect(
      issueMessagesAt(
        schema.safeParse({ spec: { replicas: '2' } }),
        'spec.replicas'
      )
    ).toEqual(['Must be odd']);
  });

  it('skips a rule on its own value while the field is empty', () => {
    const { schema } = buildSchema({
      name: {
        uiType: FieldType.Text,
        path: 'spec.name',
        fieldParams: {},
        validation: {
          celExpressions: [
            { celExpr: 'size(self) >= 3', message: 'Too short' },
          ],
        },
      },
    });

    expect(schema.safeParse({ spec: { name: '' } }).success).toBe(true);
    expect(
      issueMessagesAt(schema.safeParse({ spec: { name: 'ab' } }), 'spec.name')
    ).toEqual(['Too short']);
  });

  it('reads a multi-path field from its source path and adds no dependency', () => {
    const { schema, celDependencyGroups } = buildSchema({
      version: {
        uiType: FieldType.Text,
        path: ['spec.engine.version', 'spec.proxy.version'],
        fieldParams: {},
        validation: {
          celExpressions: [
            { celExpr: "self.startsWith('8.')", message: 'Only 8.x' },
          ],
        },
      },
    });

    expect(
      schema.safeParse({ spec: { engine: { version: '8.0' } } }).success
    ).toBe(true);
    expect(
      issueMessagesAt(
        schema.safeParse({ spec: { engine: { version: '5.7' } } }),
        'spec.engine.version'
      )
    ).toEqual(['Only 8.x']);
    expect(celDependencyGroups).toEqual([]);
  });

  it('can be compared with `original` in edit mode', () => {
    const { schema } = buildSchema(
      {
        storage: {
          uiType: FieldType.Number,
          path: 'spec.storage',
          fieldParams: {},
          validation: {
            modes: {
              [FormMode.Edit]: {
                celExpressions: [
                  {
                    celExpr: 'self >= original.spec.storage',
                    message: 'Storage cannot be decreased',
                  },
                ],
              },
            },
          },
        },
      },
      { formMode: FormMode.Edit, originalData: { spec: { storage: 10 } } }
    );

    expect(schema.safeParse({ spec: { storage: 20 } }).success).toBe(true);
    expect(
      issueMessagesAt(
        schema.safeParse({ spec: { storage: 5 } }),
        'spec.storage'
      )
    ).toEqual(['Storage cannot be decreased']);
  });
});
