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

// Synthesizes a Section from an OpenAPI v3 object schema, for a class that
// ships a parameters schema but no uiSchema. Only shapes the generator can
// edit become components: strings as text and string maps as key/value rows.

import {
  ArrayItemType,
  ArrayOutputType,
  Component,
  FieldType,
  Section,
} from 'components/ui-generator/ui-generator.types';
import { isPlainObject } from '../object-path';

export interface OpenApiSection {
  section: Section | undefined;
  // Top-level properties of a shape the generator can't edit.
  skippedProperties: string[];
}

const isStringMapSchema = (property: Record<string, unknown>): boolean =>
  property.type === 'object' &&
  !isPlainObject(property.properties) &&
  isPlainObject(property.additionalProperties) &&
  property.additionalProperties.type === 'string';

const toComponent = (
  name: string,
  property: Record<string, unknown>,
  path: string,
  required: boolean
): Component | undefined => {
  const helperText =
    typeof property.description === 'string'
      ? property.description
      : undefined;
  const fieldParams = { label: name, helperText };
  const validation = required ? { required } : undefined;

  if (property.type === 'string') {
    return { uiType: FieldType.Text, path, fieldParams, validation };
  }
  return isStringMapSchema(property)
    ? {
        uiType: FieldType.Array,
        outputType: ArrayOutputType.Object,
        path,
        fieldParams,
        validation,
        itemTemplate: { uiType: ArrayItemType.KeyValue },
      }
    : undefined;
};

export const sectionFromOpenApiSchema = (
  openAPIV3Schema: unknown,
  pathPrefix: string
): OpenApiSection => {
  if (
    !isPlainObject(openAPIV3Schema) ||
    !isPlainObject(openAPIV3Schema.properties)
  ) {
    return { section: undefined, skippedProperties: [] };
  }
  const required = Array.isArray(openAPIV3Schema.required)
    ? openAPIV3Schema.required
    : [];

  const components: Section['components'] = {};
  const skippedProperties: string[] = [];
  Object.entries(openAPIV3Schema.properties).forEach(([name, property]) => {
    const component = isPlainObject(property)
      ? toComponent(
          name,
          property,
          `${pathPrefix}.${name}`,
          required.includes(name)
        )
      : undefined;
    if (component) {
      components[name] = component;
    } else {
      skippedProperties.push(name);
    }
  });

  return {
    section:
      Object.keys(components).length > 0
        ? { components, componentsOrder: Object.keys(components) }
        : undefined,
    skippedProperties,
  };
};
