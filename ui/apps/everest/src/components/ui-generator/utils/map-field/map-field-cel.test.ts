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
import { parse } from 'yaml';
import {
  type MapComponent,
  type TopologyUISchemas,
  FieldType,
  FormMode,
  GroupType,
} from 'components/ui-generator/ui-generator.types';
import {
  KEY_VALUE_ROW_YAML,
  keyValueRow,
} from 'components/ui-generator/test-fixtures/key-value-row.fixture';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { buildZodSchema } from '../schema-builder';
import { TOGGLEABLE_SWITCHES_KEY } from '../toggleable/toggleable';

const RULE_FAILED = 'rule failed';

// A text field whose rule reads the map next to it.
const schemaWith = (celExpr: string, keyRules = '{}'): TopologyUISchemas =>
  preprocessSchema(
    parse(`
      replicaSet:
        sections:
          advanced:
            components:
              labels:
                uiType: map
                path: spec.labels
                fieldParams: { label: Labels }
                itemTemplate:
                  uiType: group
                  groupType: line
                  components:
                    key: { uiType: text, path: key, validation: ${keyRules} }
                    value: { uiType: text, path: value }
              name:
                uiType: text
                path: spec.name
                fieldParams: { label: Name }
                validation:
                  celExpressions:
                    - celExpr: ${JSON.stringify(celExpr)}
                      message: ${RULE_FAILED}
    `)
  );

interface Check {
  rows?: { key: string; value: string }[];
  // `original` as the edit form gets it.
  original?: Record<string, unknown>;
  keyRules?: string;
}

const issuesOf = (
  celExpr: string,
  { rows, original, keyRules }: Check = {}
): string[] => {
  const { schema } = buildZodSchema(
    schemaWith(celExpr, keyRules),
    'replicaSet',
    {
      formMode: original ? FormMode.Edit : FormMode.New,
      originalData: original,
    }
  );
  const result = schema.safeParse({
    spec: { name: 'db', ...(rows && { labels: rows }) },
  });
  return result.success
    ? []
    : result.error.issues.map((issue) => issue.message);
};

const TEAM = { key: 'team', value: 'db' };
const LB_TYPE = 'service.beta.kubernetes.io/aws-load-balancer-type';

const CEL_ERROR = expect.stringContaining('CEL expression error');

describe('a map field read by the CEL rule of another field', () => {
  it.each([
    ['has(spec.labels)'],
    ["'team' in spec.labels"],
    ["has(spec.labels.team) && spec.labels.team == 'db'"],
    [`'${LB_TYPE}' in spec.labels && spec.labels['${LB_TYPE}'] == 'nlb'`],
    ['size(spec.labels) == 2'],
    ["spec.labels.all(k, spec.labels[k] != '')"],
  ])('reads the map the field writes: %s', (celExpr) => {
    expect(
      issuesOf(celExpr, {
        // The last row is not an entry yet, so the rule doesn't count it.
        rows: [TEAM, { key: LB_TYPE, value: 'nlb' }, { key: '', value: '' }],
      })
    ).toEqual([]);
  });

  describe.each([[[]], [undefined], [[{ key: '', value: '' }]]])(
    'with no entries, given %j',
    (rows) => {
      it('leaves the map unset, as the API gets it', () => {
        expect(issuesOf('!has(spec.labels)', { rows })).toEqual([]);
        expect(issuesOf('has(spec.labels)', { rows })).toEqual([RULE_FAILED]);
      });

      it('fails a rule that reads the map without asking for it first', () => {
        expect(issuesOf("'team' in spec.labels", { rows })).toEqual([
          CEL_ERROR,
        ]);
        expect(issuesOf('size(spec.labels) <= 5', { rows })).toEqual([
          CEL_ERROR,
        ]);
      });

      it('answers a rule that asks for the map first with its own message', () => {
        expect(
          issuesOf("has(spec.labels) && 'team' in spec.labels", { rows })
        ).toEqual([RULE_FAILED]);
        expect(
          issuesOf('!has(spec.labels) || size(spec.labels) <= 5', { rows })
        ).toEqual([]);
      });
    }
  );

  it('reads a key the way its field writes it', () => {
    expect(
      issuesOf("'team' in spec.labels", {
        rows: [{ key: ' Team ', value: 'db' }],
        keyRules: '{ trim: true, toLowerCase: true }',
      })
    ).toEqual([]);
  });

  describe('against the saved map in edit mode', () => {
    const KEEPS_SAVED_ENTRIES =
      '!has(original.spec.labels) || (has(spec.labels) && original.spec.labels.all(k, k in spec.labels))';

    it.each([
      // The section edit modal lays the form's values over the instance.
      ['rows', { spec: { labels: [TEAM] } }],
      ['the saved map', { spec: { labels: { team: 'db' } } }],
    ])('reads `original` the same way, given %s', (_, original) => {
      expect(issuesOf(KEEPS_SAVED_ENTRIES, { rows: [TEAM], original })).toEqual(
        []
      );
      expect(
        issuesOf(KEEPS_SAVED_ENTRIES, {
          rows: [{ key: 'tier', value: 'db' }],
          original,
        })
      ).toEqual([RULE_FAILED]);
      expect(issuesOf(KEEPS_SAVED_ENTRIES, { rows: [], original })).toEqual([
        RULE_FAILED,
      ]);
    });

    it.each([
      ['without the map', { spec: { name: 'db' } }],
      ['with no row', { spec: { name: 'db', labels: [] } }],
      // Restore passes the instance as saved, where kubectl may have left this.
      ['with an empty map', { spec: { name: 'db', labels: {} } }],
    ])(
      'leaves the map unset in `original` for an instance saved %s',
      (_, original) => {
        expect(issuesOf(KEEPS_SAVED_ENTRIES, { rows: [], original })).toEqual(
          []
        );
      }
    );
  });

  describe('with the map inside an object', () => {
    const issuesWith = (
      celExpr: string,
      spec: Record<string, unknown>,
      original?: Record<string, unknown>
    ) => {
      const schema = preprocessSchema(
        parse(`
          replicaSet:
            sections:
              advanced:
                components:
                  labels:
                    uiType: map
                    path: spec.proxy.labels
                    fieldParams: { label: Labels }
                    itemTemplate: ${KEY_VALUE_ROW_YAML}
                  name:
                    uiType: text
                    path: spec.name
                    fieldParams: { label: Name }
                    validation:
                      celExpressions:
                        - celExpr: ${JSON.stringify(celExpr)}
                          message: ${RULE_FAILED}
        `)
      );
      const result = buildZodSchema(schema, 'replicaSet', {
        originalData: original,
      }).schema.safeParse({ spec });
      return result.success ? [] : result.error.issues.map((i) => i.message);
    };

    it('adds no object around the map, so a rule asking for one answers as without the map', () => {
      expect(issuesWith('!has(spec.proxy)', { name: 'db' })).toEqual([]);
      expect(
        issuesWith('!has(original.spec.proxy)', { name: 'db' }, { spec: {} })
      ).toEqual([]);
    });

    it('leaves no empty object behind once every row is removed', () => {
      expect(
        issuesWith('!has(spec.proxy)', { name: 'db', proxy: { labels: [] } })
      ).toEqual([]);
      expect(
        issuesWith('has(spec.proxy) && !has(spec.proxy.labels)', {
          name: 'db',
          proxy: { replicas: 1, labels: [] },
        })
      ).toEqual([]);
    });
  });

  it('keeps a map with no parent as the form holds it, since has() cannot ask for it', () => {
    const schema = preprocessSchema({
      replicaSet: {
        sections: {
          advanced: {
            components: {
              labels: {
                uiType: FieldType.Map,
                path: 'labels',
                fieldParams: { label: 'Labels' },
                itemTemplate: keyValueRow(),
              },
              name: {
                uiType: FieldType.Text,
                path: 'name',
                fieldParams: { label: 'Name' },
                validation: {
                  celExpressions: [
                    { celExpr: 'size(labels) == 0', message: RULE_FAILED },
                  ],
                },
              },
            },
          },
        },
      },
    });

    expect(
      buildZodSchema(schema, 'replicaSet').schema.safeParse({
        name: 'db',
        labels: [],
      }).success
    ).toBe(true);
  });

  describe('with a toggleable group', () => {
    const SWITCH = 'advanced~monitoring';

    const labels: MapComponent = {
      uiType: FieldType.Map,
      path: 'spec.monitoring.labels',
      fieldParams: { label: 'Labels' },
      itemTemplate: keyValueRow(),
    };

    // `inGroup` sets where the map sits: inside the group, or beside it.
    const issuesWith = (
      celExpr: string,
      inGroup: boolean,
      values: Record<string, unknown>
    ) => {
      const schema = preprocessSchema({
        replicaSet: {
          sections: {
            advanced: {
              components: {
                monitoring: {
                  uiType: 'group',
                  groupType: GroupType.Toggleable,
                  label: 'Monitoring',
                  components: {
                    interval: {
                      uiType: FieldType.Number,
                      path: 'spec.monitoring.interval',
                      fieldParams: { label: 'Interval' },
                    },
                    ...(inGroup && { labels }),
                  },
                },
                ...(!inGroup && { labels }),
                name: {
                  uiType: FieldType.Text,
                  path: 'spec.name',
                  fieldParams: { label: 'Name' },
                  validation: {
                    celExpressions: [{ celExpr, message: RULE_FAILED }],
                  },
                },
              },
            },
          },
        },
      });
      const result = buildZodSchema(schema, 'replicaSet').schema.safeParse(
        values
      );
      return result.success ? [] : result.error.issues.map((i) => i.message);
    };

    const withSwitch = (on: boolean, monitoring: Record<string, unknown>) => ({
      spec: { name: 'db', monitoring },
      [TOGGLEABLE_SWITCHES_KEY]: { [SWITCH]: on },
    });

    it('leaves the map unset inside a switched-off group, whatever its rows hold', () => {
      const guarded =
        "!has(spec.monitoring.labels) || 'team' in spec.monitoring.labels";
      const monitoring = {
        interval: 5,
        labels: [{ key: 'tier', value: 'db' }],
      };

      expect(issuesWith(guarded, true, withSwitch(false, monitoring))).toEqual(
        []
      );
      expect(issuesWith(guarded, true, withSwitch(true, monitoring))).toEqual([
        RULE_FAILED,
      ]);
    });

    it('leaves no object behind that the form held only for a switched-off field and an empty map', () => {
      expect(
        issuesWith(
          '!has(spec.monitoring)',
          false,
          withSwitch(false, { interval: 5, labels: [] })
        )
      ).toEqual([]);
    });
  });

  it.each([
    ["spec.labels.team == 'db'"],
    [`spec.labels['${LB_TYPE}'] == 'nlb'`],
    ["'team' in spec.labels"],
  ])(
    'runs the rule again when the map changes, whatever it reads inside: %s',
    (celExpr) => {
      const { celDependencyGroups } = buildZodSchema(
        schemaWith(celExpr),
        'replicaSet'
      );

      // The form has one field for the whole map, and a quoted key is no field.
      expect(celDependencyGroups).toEqual([['spec.name', 'spec.labels']]);
    }
  );
});
