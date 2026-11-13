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

export {
  formatMapFieldEntries,
  getMapFieldPaths,
  getMapFieldTargetPaths,
  isStringMap,
  objectToRows,
  rowsToObject,
  writeMapFieldsInApiShape,
} from './map-field';
export { buildMapFieldSchema } from './map-field-schema';
export { expandKeyValueWidget, getMapRow } from './key-value-widget';
export { type IgnoredMapField, resolveMapFields } from './resolve-map-fields';
export { type MapLintIssue, lintMapFields } from './lint-map-fields';
export type { MapRow } from './map-field.types';
