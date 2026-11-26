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
  TemplateWidgetType,
  WIDGET_UI_TYPE,
} from 'components/ui-generator/ui-generator.types';
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
                  uiType: widget
                  widgetType: keyValue
                  validation: { key: ${keyRules} }
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

describe('a map field read by the CEL rule of another field', () => {
  it.each([
    ["'team' in spec.labels"],
    ["'team' in spec.labels && spec.labels['team'] == 'db'"],
    ["'team' in spec.labels && spec.labels.team == 'db'"],
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

  it.each([[[]], [undefined], [[{ key: '', value: '' }]]])(
    'reads a map with no entries as an empty map, so the rule fails with its own message: %j',
    (rows) => {
      expect(issuesOf("'team' in spec.labels", { rows })).toEqual([
        RULE_FAILED,
      ]);
      expect(issuesOf('size(spec.labels) == 0', { rows })).toEqual([]);
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
    const KEEPS_SAVED_ENTRIES = 'original.spec.labels.all(k, k in spec.labels)';

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
    });

    it('reads an instance saved without the map as holding an empty one', () => {
      expect(
        issuesOf('size(spec.labels) >= size(original.spec.labels)', {
          rows: [],
          original: { spec: { name: 'db' } },
        })
      ).toEqual([]);
    });
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
                    itemTemplate: { uiType: widget, widgetType: keyValue }
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

    it('reads an empty map where the object is there', () => {
      const proxy = { replicas: 1 };

      expect(
        issuesWith('size(spec.proxy.labels) == 0', { name: 'db', proxy })
      ).toEqual([]);
      expect(
        issuesWith(
          'size(original.spec.proxy.labels) == 0',
          { name: 'db' },
          { spec: { proxy } }
        )
      ).toEqual([]);
    });
  });

  describe('with a toggleable group', () => {
    const SWITCH = 'advanced~monitoring';

    const labels: MapComponent = {
      uiType: FieldType.Map,
      path: 'spec.monitoring.labels',
      fieldParams: { label: 'Labels' },
      itemTemplate: {
        uiType: WIDGET_UI_TYPE,
        widgetType: TemplateWidgetType.KeyValue,
      },
    };

    // `inGroup` sets where the map sits: inside the group, or beside it.
    const issuesWith = (
      celExpr: string,
      inGroup: boolean,
      values: Record<string, unknown>,
      original?: Record<string, unknown>
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
      const result = buildZodSchema(schema, 'replicaSet', {
        originalData: original,
      }).schema.safeParse(values);
      return result.success ? [] : result.error.issues.map((i) => i.message);
    };

    const withSwitch = (on: boolean, monitoring: Record<string, unknown>) => ({
      spec: { name: 'db', monitoring },
      [TOGGLEABLE_SWITCHES_KEY]: { [SWITCH]: on },
    });

    it('leaves the map unset inside a switched-off group, like the group’s other fields', () => {
      const guarded =
        "!has(spec.monitoring.labels) || 'team' in spec.monitoring.labels";
      // `enabled` is no field of the group, so the object stays with it off.
      const monitoring = { enabled: true, interval: 5, labels: [] };

      expect(issuesWith(guarded, true, withSwitch(false, monitoring))).toEqual(
        []
      );
      expect(issuesWith(guarded, true, withSwitch(true, monitoring))).toEqual([
        RULE_FAILED,
      ]);
    });

    it('reads no map into an object the form holds only for a switched-off field', () => {
      expect(
        issuesWith(
          '!has(spec.monitoring)',
          false,
          withSwitch(false, { interval: 5 })
        )
      ).toEqual([]);
    });

    it('leaves the map unset in `original` for a group saved off', () => {
      expect(
        issuesWith(
          '!has(original.spec.monitoring.labels)',
          true,
          withSwitch(true, { interval: 5 }),
          // Something else saved there keeps the object, with the group off.
          { spec: { name: 'db', monitoring: {} } }
        )
      ).toEqual([]);
      expect(
        issuesWith(
          'size(original.spec.monitoring.labels) == 0',
          true,
          withSwitch(true, { interval: 5 }),
          { spec: { name: 'db', monitoring: { interval: 5 } } }
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
