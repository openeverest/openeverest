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

import {
  type RowsComponent,
  type RowsFieldType,
  type Section,
  FormMode,
  isRowsComponent,
} from '../../ui-generator.types';
import { isPlainObject } from '../object-path';
import { walkLeafComponents } from '../schema-walker';
import {
  IGNORED_ROW_FIELD_OPTIONS,
  IGNORED_ROW_FIELD_PARAMS,
} from './rows-field.constants';
import { isEmptyRow } from './rows';
import { getDeclaredRowFields, getRowsFieldKind } from './rows-field-kinds';
import type { RowsFieldKind } from './rows-field.types';

// Named like the Go validator's rules (provider-runtime/uischema), which
// reports the same mistakes when a provider is built: `map-ignored-option`,
// `list-invalid-default`, …
export type RowsLintRule =
  `${RowsFieldType}-${'ignored-option' | 'invalid-default' | 'invalid-limits' | 'unsatisfiable-limits'}`;

export interface RowsLintIssue {
  // Section key followed by group keys and the field key, e.g. `advanced.labels`.
  keyPath: string;
  rule: RowsLintRule;
  message: string;
}

type Branch = [at: string, validation: Record<string, unknown>];

const LIMITS: ReadonlyArray<'min' | 'max'> = ['min', 'max'];

const hasCel = (validation: Record<string, unknown>): boolean =>
  Array.isArray(validation.celExpressions) &&
  validation.celExpressions.length > 0;

// A `validation` block and each of its mode branches, as written.
const getDeclaredBranches = (validation: unknown, at: string): Branch[] => {
  if (!isPlainObject(validation)) return [];
  const modes = isPlainObject(validation.modes) ? validation.modes : {};
  return [
    [at, validation],
    ...Object.entries(modes).flatMap(([mode, branch]): Branch[] =>
      isPlainObject(branch) ? [[`${at}.modes.${mode}`, branch]] : []
    ),
  ];
};

// The rules each form mode applies, merged like resolveValidationForMode.
const getEffectiveBranches = (validation: unknown): Branch[] => {
  if (!isPlainObject(validation)) return [];
  const { modes, ...shared } = validation;
  return [
    ['validation', shared],
    ...Object.entries(isPlainObject(modes) ? modes : {}).flatMap(
      ([mode, branch]): Branch[] =>
        isPlainObject(branch)
          ? [
              [
                `validation.modes.${mode}`,
                branch.inheritShared === false
                  ? branch
                  : { ...shared, ...branch },
              ],
            ]
          : []
    ),
  ];
};

const has = (options: unknown, option: string): boolean =>
  isPlainObject(options) && options[option] !== undefined;

// The rules of a `validation` block, written at `at`, that do nothing: each in
// the block itself and in its mode branches.
const findIgnoredRules = (
  validation: unknown,
  at: string,
  ignoredRules: readonly string[]
): string[] =>
  getDeclaredBranches(validation, at).flatMap(([branchAt, branch]) =>
    ignoredRules
      .filter((rule) => branch[rule] !== undefined)
      .map((rule) => `${branchAt}.${rule}`)
  );

// What a row's field, written at `at`, sets that a row can't act on yet. A
// default counts in a mode too.
const findIgnoredRowOptions = (
  at: string,
  field: unknown,
  { ignoredRowFieldRules }: RowsFieldKind
): string[] => {
  if (!isPlainObject(field)) return [];
  const { fieldParams, validation } = field;
  const modes =
    isPlainObject(fieldParams) && isPlainObject(fieldParams.modes)
      ? fieldParams.modes
      : {};
  return [
    ...IGNORED_ROW_FIELD_OPTIONS.filter((option) => has(field, option)).map(
      (option) => `${at}.${option}`
    ),
    ...IGNORED_ROW_FIELD_PARAMS.filter((param) => has(fieldParams, param)).map(
      (param) => `${at}.fieldParams.${param}`
    ),
    ...Object.entries(modes)
      .filter(([, params]) => has(params, 'defaultValue'))
      .map(([mode]) => `${at}.fieldParams.modes.${mode}.defaultValue`),
    ...getDeclaredBranches(validation, `${at}.validation`)
      .filter(([, branch]) => hasCel(branch))
      .map(([branchAt]) => `${branchAt}.celExpressions`),
    ...findIgnoredRules(validation, `${at}.validation`, ignoredRowFieldRules),
  ];
};

const findIgnoredOptions = (
  component: RowsComponent,
  kind: RowsFieldKind
): string[] => [
  ...findIgnoredRules(component.validation, 'validation', [
    'regex',
    ...kind.ignoredRules,
  ]),
  ...getDeclaredRowFields(component).flatMap(([at, field]) =>
    findIgnoredRowOptions(at, field, kind)
  ),
];

const findInvalidDefaults = (
  { fieldParams }: RowsComponent,
  { isValidDefault }: RowsFieldKind
): string[] => {
  // A schema comes from YAML, so a field may have no `fieldParams` at all.
  if (!isPlainObject(fieldParams)) return [];
  const modes = isPlainObject(fieldParams.modes) ? fieldParams.modes : {};
  const defaults: [string, unknown][] = [
    ['fieldParams.defaultValue', fieldParams.defaultValue],
    ...Object.entries(modes).map(([mode, params]): [string, unknown] => [
      `fieldParams.modes.${mode}.defaultValue`,
      isPlainObject(params) ? params.defaultValue : undefined,
    ]),
  ];
  return defaults
    .filter(([, value]) => value !== undefined && !isValidDefault(value))
    .map(([at]) => at);
};

const isCount = (value: unknown): value is number =>
  typeof value === 'number' && Number.isInteger(value) && value >= 0;

const describeLimits = (
  [at, branch]: Branch,
  { uiType }: RowsFieldKind
): string | undefined => {
  const { min, max } = branch;
  const notCount = LIMITS.find(
    (limit) => branch[limit] !== undefined && !isCount(branch[limit])
  );
  if (notCount) {
    return `${at}.${notCount} is not a whole number of 0 or more`;
  }
  if (isCount(min) && isCount(max) && min > max) {
    return `${at} needs at least ${min} entries but allows at most ${max}, so no ${uiType} passes`;
  }
  return undefined;
};

// The forms that start from the schema's default: a new instance, and one
// imported from a backup. An edit and a restore start from a saved instance,
// which may hold any entries. A preset can fill a new form instead, and it is
// sent unchecked by the form; with none picked the form starts from the
// default, and that it can't pass then is enough to report.
const MODES_STARTING_FROM_DEFAULT = [FormMode.New, FormMode.Import];

// The params a form mode draws the field with, merged like applyModeOverrides.
const getParamsInMode = (
  fieldParams: unknown,
  mode: FormMode
): Record<string, unknown> => {
  if (!isPlainObject(fieldParams)) return {};
  const { modes, ...shared } = fieldParams;
  const override = isPlainObject(modes) ? modes[mode] : undefined;
  return isPlainObject(override) ? { ...shared, ...override } : shared;
};

// The rules a form mode applies: its own branch over the shared ones, or the
// shared ones alone.
const getRulesInMode = (
  validation: unknown,
  mode: FormMode
): Record<string, unknown> => {
  const branches = getEffectiveBranches(validation);
  const [, rules = {}] =
    branches.find(([at]) => at === `validation.modes.${mode}`) ??
    branches[0] ??
    [];
  return rules;
};

// Limits a form can never meet, in a mode whose form starts from the default:
// the default's entries are too few and none can be added, or too many and
// none can be removed. A field the mode hides or disables skips its
// rules, so it is left alone.
const describeUnsatisfiableLimits = (
  component: RowsComponent,
  kind: RowsFieldKind
): string | undefined => {
  for (const mode of MODES_STARTING_FROM_DEFAULT) {
    const params = getParamsInMode(component.fieldParams, mode);
    if (
      component.modes?.[mode]?.uiType === 'hidden' ||
      params.disabled === true
    ) {
      continue;
    }
    const { required, min, max } = getRulesInMode(component.validation, mode);
    const entries = kind.isValidDefault(params.defaultValue)
      ? kind.toRows(params.defaultValue).filter((row) => !isEmptyRow(row, kind))
          .length
      : 0;
    const needed = Math.max(required === true ? 1 : 0, isCount(min) ? min : 0);
    if (params.disableAdd === true && entries < needed) {
      return `fieldParams.disableAdd lets no entry be added in the "${mode}" form, which starts from ${entries} of the ${needed} it needs, so the form never passes`;
    }
    if (params.disableRemove === true && isCount(max) && max < entries) {
      return `fieldParams.disableRemove lets no entry be removed in the "${mode}" form, which starts from ${entries} while it allows at most ${max}, so the form never passes`;
    }
  }
  return undefined;
};

// Mistakes in a schema the UI renders anyway, ignoring the part it can't use.
// Run on sections whose unsupported rows fields are already dropped.
export const lintRowsFields = (
  sections: Record<string, Section>
): RowsLintIssue[] => {
  const issues: RowsLintIssue[] = [];
  Object.entries(sections).forEach(([sectionKey, section]) => {
    if (!section?.components) return;
    walkLeafComponents(
      section.components,
      ({ component, generatedName: keyPath }) => {
        if (!isRowsComponent(component)) return;
        const kind = getRowsFieldKind(component);
        const { uiType, valueShape } = kind;
        const ignored = findIgnoredOptions(component, kind);
        if (ignored.length > 0) {
          issues.push({
            keyPath,
            rule: `${uiType}-ignored-option`,
            message: `sets ${ignored.join(', ')}, which a ${uiType} field ignores`,
          });
        }
        const defaults = findInvalidDefaults(component, kind);
        if (defaults.length > 0) {
          issues.push({
            keyPath,
            rule: `${uiType}-invalid-default`,
            message: `sets ${defaults.join(', ')} to something other than a ${valueShape}, so the field starts empty`,
          });
        }
        const limits = getEffectiveBranches(component.validation)
          .map((branch) => describeLimits(branch, kind))
          .find((message) => message !== undefined);
        if (limits) {
          issues.push({
            keyPath,
            rule: `${uiType}-invalid-limits`,
            message: limits,
          });
        }
        const unsatisfiable = describeUnsatisfiableLimits(component, kind);
        if (unsatisfiable) {
          issues.push({
            keyPath,
            rule: `${uiType}-unsatisfiable-limits`,
            message: unsatisfiable,
          });
        }
      },
      sectionKey
    );
  });
  return issues;
};
