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
  isRowsComponent,
} from '../../ui-generator.types';
import { isPlainObject } from '../object-path';
import { walkLeafComponents } from '../schema-walker';
import {
  IGNORED_ROW_FIELD_OPTIONS,
  IGNORED_ROW_FIELD_PARAMS,
} from './rows-field.constants';
import { getDeclaredRowFields, getRowsFieldKind } from './rows-field-kinds';
import type { RowsFieldKind } from './rows-field.types';

// Named like the Go validator's rules (provider-runtime/uischema), which
// reports the same mistakes when a provider is built: `map-ignored-option`,
// `list-invalid-default`, …
export type RowsLintRule =
  `${RowsFieldType}-${'ignored-option' | 'invalid-default' | 'invalid-limits'}`;

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

// What a row's field, written at `at`, sets that a row can't act on yet. A
// default counts in a mode too.
const findIgnoredRowOptions = (at: string, field: unknown): string[] => {
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
  ];
};

const findIgnoredOptions = (component: RowsComponent): string[] => [
  ...getDeclaredBranches(component.validation, 'validation').flatMap(
    ([at, branch]) => (branch.regex !== undefined ? [`${at}.regex`] : [])
  ),
  ...getDeclaredRowFields(component).flatMap(([at, field]) =>
    findIgnoredRowOptions(at, field)
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
        const ignored = findIgnoredOptions(component);
        if (ignored.length > 0) {
          issues.push({
            keyPath,
            rule: `${uiType}-ignored-option`,
            message: `sets ${ignored.join(', ')}, which a ${uiType} field doesn't support yet and ignores`,
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
      },
      sectionKey
    );
  });
  return issues;
};
