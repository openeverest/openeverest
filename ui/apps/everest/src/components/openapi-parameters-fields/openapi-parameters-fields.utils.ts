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

import { z } from 'zod';
import { KeyValuePair } from '@percona/ui-lib';
import {
  OpenApiParameterField,
  OpenApiParameterFields,
  OpenApiParameterFormValues,
  OpenApiParameterKind,
} from './openapi-parameters-fields.types';
import { Messages } from './openapi-parameters-fields.messages';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

const isStringMapSchema = (property: Record<string, unknown>) =>
  property.type === 'object' &&
  !isRecord(property.properties) &&
  isRecord(property.additionalProperties) &&
  property.additionalProperties.type === 'string';

const toKind = (property: unknown): OpenApiParameterKind | undefined => {
  if (!isRecord(property)) {
    return undefined;
  }
  if (property.type === 'string') {
    return OpenApiParameterKind.Text;
  }
  return isStringMapSchema(property)
    ? OpenApiParameterKind.StringMap
    : undefined;
};

// The top-level properties of an OpenAPI v3 object schema a simple form can
// edit: strings as text and string maps as key/value rows.
export const getOpenApiParameterFields = (
  openAPIV3Schema: unknown
): OpenApiParameterFields => {
  if (!isRecord(openAPIV3Schema) || !isRecord(openAPIV3Schema.properties)) {
    return { fields: [], skipped: [] };
  }
  const required = Array.isArray(openAPIV3Schema.required)
    ? openAPIV3Schema.required
    : [];

  return Object.entries(
    openAPIV3Schema.properties
  ).reduce<OpenApiParameterFields>(
    ({ fields, skipped }, [name, property]) => {
      const kind = toKind(property);
      if (!kind) {
        return { fields, skipped: [...skipped, name] };
      }
      const description =
        isRecord(property) && typeof property.description === 'string'
          ? property.description
          : undefined;
      return {
        fields: [
          ...fields,
          { name, kind, required: required.includes(name), description },
        ],
        skipped,
      };
    },
    { fields: [], skipped: [] }
  );
};

const toPairs = (value: unknown): KeyValuePair[] =>
  isRecord(value)
    ? Object.entries(value).map(([key, entryValue]) => ({
        key,
        value: typeof entryValue === 'string' ? entryValue : '',
      }))
    : [];

export const toOpenApiParameterFormValues = (
  parameters: unknown,
  { fields }: OpenApiParameterFields
): OpenApiParameterFormValues => {
  const source = isRecord(parameters) ? parameters : {};
  return Object.fromEntries(
    fields.map((field) => {
      const value = source[field.name];
      return [
        field.name,
        field.kind === OpenApiParameterKind.Text
          ? typeof value === 'string'
            ? value
            : ''
          : toPairs(value),
      ];
    })
  );
};

const isKeyValuePair = (value: unknown): value is KeyValuePair =>
  isRecord(value) &&
  typeof value.key === 'string' &&
  typeof value.value === 'string';

// Empty values are left out, so an unset parameter keeps the class default.
export const fromOpenApiParameterFormValues = (
  values: unknown,
  { fields }: OpenApiParameterFields
): Record<string, unknown> => {
  const source = isRecord(values) ? values : {};
  return Object.fromEntries(
    fields.flatMap((field): [string, unknown][] => {
      const value = source[field.name];
      if (field.kind === OpenApiParameterKind.Text) {
        return typeof value === 'string' && value !== ''
          ? [[field.name, value]]
          : [];
      }
      const pairs = (Array.isArray(value) ? value : []).filter(isKeyValuePair);
      return pairs.length > 0
        ? [[field.name, Object.fromEntries(pairs.map((p) => [p.key, p.value]))]]
        : [];
    })
  );
};

const noDuplicateKeys = (pairs: KeyValuePair[], ctx: z.RefinementCtx) =>
  pairs.forEach(({ key }, index) => {
    if (key && pairs.findIndex((pair) => pair.key === key) !== index) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: [index, 'key'],
        message: Messages.duplicateKey(key),
      });
    }
  });

const pairsSchema = (required: boolean, fieldName: string) => {
  const pairs = z.array(
    z.object({
      key: z.string().min(1, Messages.keyRequired),
      value: z.string(),
    })
  );
  return (
    required ? pairs.min(1, Messages.required(fieldName)) : pairs
  ).superRefine(noDuplicateKeys);
};

const toFieldSchema = (field: OpenApiParameterField): z.ZodTypeAny => {
  if (field.kind === OpenApiParameterKind.StringMap) {
    return pairsSchema(field.required, field.name);
  }
  return field.required
    ? z.string().min(1, Messages.required(field.name))
    : z.string();
};

export const buildOpenApiParametersSchema = ({
  fields,
}: OpenApiParameterFields): z.ZodTypeAny =>
  z.object(
    Object.fromEntries(
      fields.map((field) => [field.name, toFieldSchema(field)])
    )
  );
