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
  type TopologyUISchemas,
  FormMode,
} from 'components/ui-generator/ui-generator.types';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { buildZodSchema } from '../schema-builder';
import { TOGGLEABLE_SWITCHES_KEY } from '../toggleable/toggleable';

const RULE_FAILED = 'rule failed';
const CEL_ERROR = expect.stringContaining('CEL expression error');

const rule = (celExpr: string) =>
  `{ celExpressions: [{ celExpr: ${JSON.stringify(celExpr)}, message: ${RULE_FAILED} }] }`;

const rowsOf = (...values: string[]) => values.map((value) => ({ value }));

interface Check {
  rows?: { value: string }[];
  // `original` as the edit form gets it.
  original?: Record<string, unknown>;
  name?: string;
}

// Each issue with where the form shows it: a path's segments joined.
const issuesOf = (
  schema: TopologyUISchemas,
  { rows, original, name = 'db' }: Check
) => {
  const result = buildZodSchema(schema, 'replicaSet', {
    formMode: original ? FormMode.Edit : FormMode.New,
    originalData: original,
  }).schema.safeParse({ spec: { name, ...(rows && { ranges: rows }) } });
  return result.success
    ? []
    : result.error.issues.map(({ path, message }) => ({
        path: path.join('.'),
        message,
      }));
};

// A list and a text field; `rules` go on the one named, in YAML flow style.
const schemaWith = (
  rules: { name?: string; ranges?: string },
  itemRules = '{}',
  listParams = ''
): TopologyUISchemas =>
  preprocessSchema(
    parse(`
      replicaSet:
        sections:
          advanced:
            components:
              ranges:
                uiType: list
                path: spec.ranges
                fieldParams: { label: Source ranges${listParams} }
                validation: ${rules.ranges ?? '{}'}
                itemTemplate: { uiType: text, validation: ${itemRules} }
              name:
                uiType: text
                path: spec.name
                fieldParams: { label: Name }
                validation: ${rules.name ?? '{}'}
    `)
  );

const OFFICE = '10.0.0.0/8';
const VPN = '192.168.0.0/16';
const UNDER_NAME = 'spec.name';
// Under the list, where its count errors go, not on a row.
const UNDER_THE_LIST = 'spec.ranges.root';

describe('a list field read by the CEL rule of another field', () => {
  const messagesOf = (celExpr: string, check: Check = {}, itemRules = '{}') =>
    issuesOf(schemaWith({ name: rule(celExpr) }, itemRules), check).map(
      ({ path, message }) => {
        expect(path).toBe(UNDER_NAME);
        return message;
      }
    );

  it.each([
    ['has(spec.ranges)'],
    [`'${OFFICE}' in spec.ranges`],
    ['size(spec.ranges) == 2'],
    ["spec.ranges.all(v, v.contains('/'))"],
    ["spec.ranges.exists(v, v.startsWith('192.168.'))"],
    [`spec.ranges[0] == '${OFFICE}' && spec.ranges[1] == '${VPN}'`],
    [`spec.ranges == ['${OFFICE}', '${VPN}']`],
  ])(
    'reads the list the field writes, in the order of its rows: %s',
    (celExpr) => {
      // The row in the middle is not an entry, so the rule doesn't count it.
      expect(messagesOf(celExpr, { rows: rowsOf(OFFICE, '', VPN) })).toEqual(
        []
      );
    }
  );

  it('fails a rule that reads an entry the list does not have', () => {
    expect(
      messagesOf(`spec.ranges[1] == '${VPN}'`, { rows: rowsOf(OFFICE) })
    ).toEqual([CEL_ERROR]);
    expect(
      messagesOf(`size(spec.ranges) > 1 && spec.ranges[1] == '${VPN}'`, {
        rows: rowsOf(OFFICE),
      })
    ).toEqual([RULE_FAILED]);
  });

  describe.each([[[]], [undefined], [rowsOf('')]])(
    'with no entries, given %j',
    (rows) => {
      it('leaves the list unset, as the API gets it', () => {
        expect(messagesOf('!has(spec.ranges)', { rows })).toEqual([]);
        expect(messagesOf('has(spec.ranges)', { rows })).toEqual([RULE_FAILED]);
      });

      it('fails a rule that reads the list without asking for it first', () => {
        expect(messagesOf(`'${OFFICE}' in spec.ranges`, { rows })).toEqual([
          CEL_ERROR,
        ]);
        expect(messagesOf('size(spec.ranges) <= 5', { rows })).toEqual([
          CEL_ERROR,
        ]);
      });

      it('answers a rule that asks for the list first with its own message', () => {
        expect(
          messagesOf(`has(spec.ranges) && '${OFFICE}' in spec.ranges`, { rows })
        ).toEqual([RULE_FAILED]);
        expect(
          messagesOf('!has(spec.ranges) || size(spec.ranges) <= 5', { rows })
        ).toEqual([]);
      });
    }
  );

  it('reads a value the way the item writes it', () => {
    expect(
      messagesOf(
        "'example.com' in spec.ranges",
        { rows: rowsOf(' Example.COM ') },
        '{ trim: true, toLowerCase: true }'
      )
    ).toEqual([]);
  });

  describe('against the saved list in edit mode', () => {
    const KEEPS_SAVED_ENTRIES =
      '!has(original.spec.ranges) || (has(spec.ranges) && original.spec.ranges.all(v, v in spec.ranges))';

    it.each([
      // The section edit modal lays the form's values over the instance.
      ['rows', { spec: { ranges: rowsOf(OFFICE) } }],
      ['the saved list', { spec: { ranges: [OFFICE] } }],
    ])('reads `original` the same way, given %s', (_, original) => {
      expect(
        messagesOf(KEEPS_SAVED_ENTRIES, { rows: rowsOf(VPN, OFFICE), original })
      ).toEqual([]);
      expect(
        messagesOf(KEEPS_SAVED_ENTRIES, { rows: rowsOf(VPN), original })
      ).toEqual([RULE_FAILED]);
      expect(messagesOf(KEEPS_SAVED_ENTRIES, { rows: [], original })).toEqual([
        RULE_FAILED,
      ]);
    });

    it.each([
      ['without the list', { spec: { name: 'db' } }],
      ['with no row', { spec: { name: 'db', ranges: rowsOf() } }],
      // Restore passes the instance as saved, where kubectl may have left this.
      ['with an empty list', { spec: { name: 'db', ranges: [] } }],
    ])(
      'leaves the list unset in `original` for an instance saved %s',
      (_, original) => {
        expect(messagesOf(KEEPS_SAVED_ENTRIES, { rows: [], original })).toEqual(
          []
        );
      }
    );
  });

  it.each([
    [`'${OFFICE}' in spec.ranges`],
    ["spec.ranges.all(v, v.contains('/'))"],
    [`spec.ranges[0] == '${OFFICE}'`],
    ['spec.ranges.size() > 0'],
  ])(
    'runs the rule again when the list changes, whatever it reads inside: %s',
    (celExpr) => {
      const [watched] = buildZodSchema(
        schemaWith({ name: rule(celExpr) }),
        'replicaSet'
      ).celDependencyGroups;

      // The form has one field for the whole list, so a path into it, such
      // as `spec.ranges.all`, is the list.
      expect(watched.filter((path) => path.startsWith('spec.'))).toEqual([
        'spec.name',
        'spec.ranges',
      ]);
    }
  );
});

describe('a CEL rule on the list itself', () => {
  const NO_OPEN_RANGE = rule(
    "!has(spec.ranges) || !('0.0.0.0/0' in spec.ranges)"
  );
  const issuesOfOwnRule = (rules: string, check: Check, listParams = '') =>
    issuesOf(schemaWith({ ranges: rules }, '{}', listParams), check);

  it('reads the list by its path and reports under the list', () => {
    expect(issuesOfOwnRule(NO_OPEN_RANGE, { rows: rowsOf(OFFICE) })).toEqual(
      []
    );
    expect(
      issuesOfOwnRule(NO_OPEN_RANGE, { rows: rowsOf(OFFICE, '0.0.0.0/0') })
    ).toEqual([{ path: UNDER_THE_LIST, message: RULE_FAILED }]);
  });

  it('finds the list unset while it has no entries, so the rule asks for it first', () => {
    expect(issuesOfOwnRule(NO_OPEN_RANGE, { rows: [] })).toEqual([]);
    expect(
      issuesOfOwnRule(rule("!('0.0.0.0/0' in spec.ranges)"), { rows: [] })
    ).toEqual([{ path: UNDER_THE_LIST, message: CEL_ERROR }]);
  });

  it('reads another field, and is checked again when that field changes', () => {
    const PROD_NEEDS_RANGES = rule("spec.name != 'prod' || has(spec.ranges)");

    expect(issuesOfOwnRule(PROD_NEEDS_RANGES, { rows: [] })).toEqual([]);
    expect(
      issuesOfOwnRule(PROD_NEEDS_RANGES, { rows: [], name: 'prod' })
    ).toEqual([{ path: UNDER_THE_LIST, message: RULE_FAILED }]);
    expect(
      buildZodSchema(schemaWith({ ranges: PROD_NEEDS_RANGES }), 'replicaSet')
        .celDependencyGroups
    ).toContainEqual(expect.arrayContaining(['spec.ranges', 'spec.name']));
  });

  it('compares with the saved list in the mode that has one', () => {
    const KEEPS_SAVED_ENTRIES = `{ modes: { edit: ${rule(
      '!has(original.spec.ranges) || (has(spec.ranges) && original.spec.ranges.all(v, v in spec.ranges))'
    )} } }`;
    const original = { spec: { ranges: [OFFICE] } };

    expect(issuesOfOwnRule(KEEPS_SAVED_ENTRIES, { rows: [] })).toEqual([]);
    expect(
      issuesOfOwnRule(KEEPS_SAVED_ENTRIES, { rows: rowsOf(OFFICE), original })
    ).toEqual([]);
    expect(
      issuesOfOwnRule(KEEPS_SAVED_ENTRIES, { rows: rowsOf(VPN), original })
    ).toEqual([{ path: UNDER_THE_LIST, message: RULE_FAILED }]);
  });

  it('reports after the count of entries, whose error the form then shows first', () => {
    expect(
      issuesOfOwnRule(NO_OPEN_RANGE.replace('{ ', '{ max: 1, '), {
        rows: rowsOf(OFFICE, '0.0.0.0/0'),
      })
    ).toEqual([
      { path: UNDER_THE_LIST, message: expect.stringContaining('at most 1') },
      { path: UNDER_THE_LIST, message: RULE_FAILED },
    ]);
  });

  it('skips the rule of a disabled list, which the user cannot change', () => {
    expect(
      issuesOfOwnRule(
        NO_OPEN_RANGE,
        { rows: rowsOf('0.0.0.0/0') },
        ', disabled: true'
      )
    ).toEqual([]);
  });
});

describe('a list field in a toggleable group', () => {
  const SWITCH = 'advanced~access';
  // `ownRule` sits on the list; `nameRule` on a field outside the group.
  const issuesWhen = (
    on: boolean,
    { ownRule = '{}', nameRule = '{}' }: { ownRule?: string; nameRule?: string }
  ) => {
    const schema = preprocessSchema(
      parse(`
        replicaSet:
          sections:
            advanced:
              components:
                access:
                  uiType: group
                  groupType: toggleable
                  label: Access
                  components:
                    ranges:
                      uiType: list
                      path: spec.access.ranges
                      fieldParams: { label: Source ranges }
                      validation: ${ownRule}
                      itemTemplate: { uiType: text }
                name:
                  uiType: text
                  path: spec.name
                  fieldParams: { label: Name }
                  validation: ${nameRule}
      `)
    );
    const result = buildZodSchema(schema, 'replicaSet').schema.safeParse({
      spec: { name: 'db', access: { ranges: rowsOf('0.0.0.0/0') } },
      [TOGGLEABLE_SWITCHES_KEY]: { [SWITCH]: on },
    });
    return result.success
      ? []
      : result.error.issues.map(({ path, message }) => ({
          path: path.join('.'),
          message,
        }));
  };

  it('skips the rule on the list while the group is off', () => {
    const ownRule = rule("!('0.0.0.0/0' in spec.access.ranges)");

    expect(issuesWhen(false, { ownRule })).toEqual([]);
    expect(issuesWhen(true, { ownRule })).toEqual([
      { path: 'spec.access.ranges.root', message: RULE_FAILED },
    ]);
  });

  it('leaves the list unset for a rule outside the group while it is off, whatever its rows hold', () => {
    const nameRule = rule(
      "!has(spec.access.ranges) || !('0.0.0.0/0' in spec.access.ranges)"
    );

    expect(issuesWhen(false, { nameRule })).toEqual([]);
    expect(issuesWhen(true, { nameRule })).toEqual([
      { path: UNDER_NAME, message: RULE_FAILED },
    ]);
  });
});
