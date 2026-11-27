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
  type Component,
  type ComponentGroup,
  type MapComponent,
  type MapRowRole,
  FieldType,
  MAP_ROW_ROLES,
} from '../../ui-generator.types';
import {
  IGNORED_ROW_FIELD_OPTIONS,
  IGNORED_ROW_FIELD_PARAMS,
  IGNORED_ROW_FIELD_RULES,
} from './map-field.constants';
import { Messages } from './map-field.messages';

const without = <T extends object>(options: T, ignored: string[]): T => {
  const copy = { ...options };
  ignored.forEach((option) => Reflect.deleteProperty(copy, option));
  return copy;
};

// A row field is the key or the value by the path it writes in the row; the
// name of the component is the author's.
export const getRowFieldRole = (
  field: Component | ComponentGroup
): MapRowRole | undefined =>
  'components' in field
    ? undefined
    : MAP_ROW_ROLES.find((role) => field.path === role);

export const getRowField = (
  row: ComponentGroup,
  role: MapRowRole
): Component | ComponentGroup | undefined =>
  Object.values(row.components).find(
    (field) => getRowFieldRole(field) === role
  );

// A row field as the form draws it: named `Key` or `Value` until the schema
// says otherwise, and without what a row can't act on yet.
const toRowField = (
  field: Component | ComponentGroup
): Component | ComponentGroup => {
  const role = getRowFieldRole(field);
  if (!role || 'components' in field || field.uiType !== FieldType.Text) {
    return field;
  }
  return {
    ...without(field, IGNORED_ROW_FIELD_OPTIONS),
    fieldParams: {
      placeholder: Messages[role],
      ...without(field.fieldParams, IGNORED_ROW_FIELD_PARAMS),
    },
    ...(field.validation && {
      validation: without(field.validation, IGNORED_ROW_FIELD_RULES),
    }),
  };
};

// The row a map's item template declares, as the form draws it.
export const toMapRow = (itemTemplate: ComponentGroup): ComponentGroup => ({
  ...itemTemplate,
  components: Object.fromEntries(
    Object.entries(itemTemplate.components).map(([name, field]) => [
      name,
      toRowField(field),
    ])
  ),
});

// Preprocess sets `_row`; a form that skips it gets the row prepared here.
export const getMapRow = (component: MapComponent): ComponentGroup =>
  component._row ?? toMapRow(component.itemTemplate);
