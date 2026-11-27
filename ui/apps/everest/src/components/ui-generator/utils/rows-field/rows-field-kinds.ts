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

// The only place that tells a map from a list. Everything else in this folder
// works on rows and takes what differs from the field's kind.

import {
  type Component,
  type ComponentGroup,
  type RowsComponent,
  type RowsFieldType,
  FieldType,
} from '../../ui-generator.types';
import { LIST_FIELD_KIND, toListRow } from '../list-field/list-field-kind';
import { MAP_FIELD_KIND } from '../map-field/map-field-kind';
import { prepareRow } from './row-template';
import type { RowsFieldKind } from './rows-field.types';

const KINDS: Record<RowsFieldType, RowsFieldKind> = {
  [FieldType.Map]: MAP_FIELD_KIND,
  [FieldType.List]: LIST_FIELD_KIND,
};

export const getRowsFieldKind = ({ uiType }: RowsComponent): RowsFieldKind =>
  KINDS[uiType];

// The kind a field's `uiType` names, also in a shape this version doesn't
// implement. Undefined for a field that isn't edited as rows.
export const findRowsFieldKind = (
  component: Component
): RowsFieldKind | undefined =>
  component.uiType === FieldType.Map || component.uiType === FieldType.List
    ? KINDS[component.uiType]
    : undefined;

// Each field of a row as the schema writes it, with where it is written: a
// map declares a group of them, a list the one field itself.
export const getDeclaredRowFields = (
  component: RowsComponent
): [at: string, field: unknown][] =>
  component.uiType === FieldType.Map
    ? Object.entries(component.itemTemplate.components).map(([name, field]) => [
        `itemTemplate.components.${name}`,
        field,
      ])
    : [['itemTemplate', component.itemTemplate]];

// The row a field's item template declares, as the form draws it.
const toRow = (component: RowsComponent): ComponentGroup =>
  prepareRow(
    component.uiType === FieldType.Map
      ? component.itemTemplate
      : toListRow(component.itemTemplate)
  );

// The field as the stages after preprocess read it: with its row prepared, and
// with the `fieldParams` a schema may leave out. Written per kind, so that
// each keeps the type of its own params.
export const withRow = (component: RowsComponent): RowsComponent => {
  const _row = toRow(component);
  return component.uiType === FieldType.Map
    ? { ...component, fieldParams: component.fieldParams ?? {}, _row }
    : { ...component, fieldParams: component.fieldParams ?? {}, _row };
};

// Preprocess sets `_row`; a form that skips it gets the row prepared here.
export const getRow = (component: RowsComponent): ComponentGroup =>
  component._row ?? toRow(component);
