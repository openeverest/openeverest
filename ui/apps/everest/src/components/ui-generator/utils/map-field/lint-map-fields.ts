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
  type MapComponent,
  type Section,
  TemplateWidgetType,
  isMapComponent,
  isWidgetComponent,
} from '../../ui-generator.types';
import { isPlainObject } from '../object-path';
import { walkLeafComponents } from '../schema-walker';
import { isStringMap } from './map-field';
import { IGNORED_ROW_FIELD_PARAMS } from './map-field.constants';

// Named like the Go validator's rules (provider-runtime/uischema), which
// reports the same mistakes when a provider is built.
export type MapLintRule =
  | 'map-ignored-option'
  | 'map-invalid-default'
  | 'map-invalid-limits'
  | 'template-widget-misplaced';

export interface MapLintIssue {
  // Section key followed by group keys and the field key, e.g. `advanced.labels`.
  keyPath: string;
  rule: MapLintRule;
  message: string;
}

type Branch = [at: string, validation: Record<string, unknown>];

const TEMPLATE_WIDGET_TYPES: ReadonlySet<string> = new Set(
  Object.values(TemplateWidgetType)
);
const ROW_FIELDS: ReadonlyArray<'key' | 'value'> = ['key', 'value'];
const LIMITS: ReadonlyArray<'minItems' | 'maxItems'> = ['minItems', 'maxItems'];

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

const findIgnoredOptions = ({
  validation,
  itemTemplate,
}: MapComponent): string[] => [
  ...getDeclaredBranches(validation, 'validation').flatMap(([at, branch]) => [
    ...(branch.regex !== undefined ? [`${at}.regex`] : []),
    ...(hasCel(branch) ? [`${at}.celExpressions`] : []),
  ]),
  ...ROW_FIELDS.flatMap((field) => {
    const params: unknown = itemTemplate.fieldParams?.[field];
    const rules: unknown = itemTemplate.validation?.[field];
    return [
      ...IGNORED_ROW_FIELD_PARAMS.filter(
        (param) => isPlainObject(params) && params[param] !== undefined
      ).map((param) => `itemTemplate.fieldParams.${field}.${param}`),
      ...getDeclaredBranches(rules, `itemTemplate.validation.${field}`)
        .filter(([, branch]) => hasCel(branch))
        .map(([at]) => `${at}.celExpressions`),
    ];
  }),
];

const findInvalidDefaults = ({ fieldParams }: MapComponent): string[] => {
  const modes = isPlainObject(fieldParams.modes) ? fieldParams.modes : {};
  const defaults: [string, unknown][] = [
    ['fieldParams.defaultValue', fieldParams.defaultValue],
    ...Object.entries(modes).map(([mode, params]): [string, unknown] => [
      `fieldParams.modes.${mode}.defaultValue`,
      isPlainObject(params) ? params.defaultValue : undefined,
    ]),
  ];
  return defaults
    .filter(([, value]) => value !== undefined && !isStringMap(value))
    .map(([at]) => at);
};

const isCount = (value: unknown): value is number =>
  typeof value === 'number' && Number.isInteger(value) && value >= 0;

const describeLimits = ([at, branch]: Branch): string | undefined => {
  const { minItems, maxItems } = branch;
  const notCount = LIMITS.find(
    (limit) => branch[limit] !== undefined && !isCount(branch[limit])
  );
  if (notCount) {
    return `${at}.${notCount} is not a whole number of 0 or more`;
  }
  if (isCount(minItems) && isCount(maxItems) && minItems > maxItems) {
    return `${at} needs at least ${minItems} entries but allows at most ${maxItems}, so no map passes`;
  }
  return undefined;
};

// Mistakes in a schema the UI renders anyway, ignoring the part it can't use.
// Run on sections whose unsupported maps are already dropped.
export const lintMapFields = (
  sections: Record<string, Section>
): MapLintIssue[] => {
  const issues: MapLintIssue[] = [];
  Object.entries(sections).forEach(([sectionKey, section]) => {
    if (!section?.components) return;
    walkLeafComponents(
      section.components,
      ({ component, generatedName: keyPath }) => {
        if (
          isWidgetComponent(component) &&
          TEMPLATE_WIDGET_TYPES.has(component.widgetType)
        ) {
          issues.push({
            keyPath,
            rule: 'template-widget-misplaced',
            message: `is the "${component.widgetType}" template widget, which stands only for a map's row in itemTemplate; anywhere else it renders nothing`,
          });
          return;
        }
        if (!isMapComponent(component)) return;
        const ignored = findIgnoredOptions(component);
        if (ignored.length > 0) {
          issues.push({
            keyPath,
            rule: 'map-ignored-option',
            message: `sets ${ignored.join(', ')}, which a map field ignores`,
          });
        }
        const defaults = findInvalidDefaults(component);
        if (defaults.length > 0) {
          issues.push({
            keyPath,
            rule: 'map-invalid-default',
            message: `sets ${defaults.join(', ')} to something other than a map of strings, so the field starts empty`,
          });
        }
        const limits = getEffectiveBranches(component.validation)
          .map(describeLimits)
          .find((message) => message !== undefined);
        if (limits) {
          issues.push({ keyPath, rule: 'map-invalid-limits', message: limits });
        }
      },
      sectionKey
    );
  });
  return issues;
};
