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

import { SelectInput, TextInput } from '@percona/ui-lib';
import { FieldType, GroupType } from './ui-generator.types';
import AccordionWrapper from './ui-group-wrappers/accordion-wrapper';
import StackWrapper from './ui-group-wrappers/stack-wrapper';
import { z } from 'zod';

export const UI_TYPE_DEFAULT_VALUE: Partial<Record<FieldType, unknown>> = {
  [FieldType.Select]: '',
  [FieldType.Text]: '',
  [FieldType.Hidden]: undefined,
};

export const componentGroupMap: Record<string, React.ElementType> = {
  [GroupType.Accordion]: AccordionWrapper,
  [GroupType.Line]: StackWrapper,
};
export const muiComponentMap: Record<FieldType, React.ElementType> = {
  [FieldType.Number]: TextInput,
  [FieldType.Text]: TextInput,
  [FieldType.Select]: SelectInput,
  [FieldType.Hidden]: () => null,
};

export const zodRuleMapByType: Record<FieldType, Record<string, string>> = {
  [FieldType.Number]: {
    min: 'min',
    max: 'max',
    gt: 'gt',
    lt: 'lt',
    int: 'int',
    multipleOf: 'multipleOf',
    safe: 'safe',
  },
  /**
   * Text field zod string validations.
   * Boolean flags (email, url, uuid, trim, toLowerCase, toUpperCase) are
   * handled separately in buildTextValidationSchema — they are not included
   * here because they must be called without arguments.
   * Value-based rules are listed here for reference / generic fallback use.
   */
  [FieldType.Text]: {
    min: 'min',
    max: 'max',
    length: 'length',
    // Boolean validations (email, url, uuid, trim, toLowerCase, toUpperCase)
    // are handled explicitly in buildTextValidationSchema.
  },
  [FieldType.Select]: {
    // Select fields typically don't have Zod-level validations beyond type checking
    // Custom validations could be added here in the future
  },
  [FieldType.Hidden]: {
    // Hidden fields don't need validation
  },
};

export const getZodRulesForFieldType = (
  fieldType: FieldType
): Record<string, string> => {
  return zodRuleMapByType[fieldType] || {};
};

/*
 Universal helper encapsulates type assertions in one place, making it easy to support
 any field type (number, string, date, etc.) without code duplication.
 */
export const applyZodValidation = (
  schema: z.ZodTypeAny,
  methodName: string,
  value?: unknown
): z.ZodTypeAny => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const method = (schema as any)[methodName];

  if (typeof method !== 'function') {
    return schema;
  }

  return value !== undefined ? method.call(schema, value) : method.call(schema);
};

export const ZOD_SCHEMA_MAP: Record<FieldType, z.ZodTypeAny> = {
  [FieldType.Number]: z
    .union([z.string(), z.number()])
    .optional()
    .transform((val) => {
      // this is needed to handle required param from the schema
      if (val === undefined || val === '') return undefined;
      const num = typeof val === 'string' ? parseFloat(val) : val;
      return isNaN(num) ? undefined : num;
    }),
  [FieldType.Text]: z.string(),
  [FieldType.Select]: z.string(),
  [FieldType.Hidden]: z.any(),
};
