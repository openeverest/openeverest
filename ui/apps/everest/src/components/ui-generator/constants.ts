import { SelectInput, TextInput } from '@percona/ui-lib';
import { FieldType, GroupType } from './ui-generator.types';
import AccordionWrapper from './ui-group-wrappers/accordion-wrapper';
import StackWrapper from './ui-group-wrappers/stack-wrapper';
import { z } from 'zod';

export const UI_TYPE_DEFAULT_VALUE: Partial<Record<FieldType, unknown>> = {
  //   [FieldType.Switch]: false,
  //   [FieldType.Checkbox]: false,
  //   [FieldType.Toggle]: false,
  //   [FieldType.TextArea]: 'lorem ipsum',
  //   [FieldType.Input]: '',
  //   [FieldType.StorageClassSelect]: 'lorem ipsum',
  //   [FieldType.SecretSelector]: '',
  //   [FieldType.String]: '',
  [FieldType.Select]: '',
  [FieldType.Hidden]: undefined,
};

export const componentGroupMap: Record<string, React.ElementType> = {
  [GroupType.Accordion]: AccordionWrapper,
  [GroupType.Line]: StackWrapper,
};
export const muiComponentMap: Record<FieldType, React.ElementType> = {
  [FieldType.Number]: TextInput,
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
  [FieldType.Select]: z.string(),
  [FieldType.Hidden]: z.any(),
  // [FieldType.Input]: z.string(),
  // [FieldType.Switch]: z.boolean(),
  // [FieldType.Checkbox]: z.boolean(),
  // [FieldType.String]: z.string().min(5),
  // [FieldType.TextArea]: z.string().min(5),
};
