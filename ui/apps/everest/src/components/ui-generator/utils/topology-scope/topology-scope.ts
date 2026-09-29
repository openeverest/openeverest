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

import { TopologyUISchemas } from '../../ui-generator.types';
import { deepClone, deleteByPath } from '../object-path';
import { collectAllSchemaPaths } from '../schema-walker';

const collectTopologyPaths = (
  schema: TopologyUISchemas,
  topology: string
): string[] =>
  Array.from(collectAllSchemaPaths(schema[topology]?.sections ?? {}));

const isSameOrNested = (a: string, b: string): boolean =>
  a === b || a.startsWith(`${b}.`) || b.startsWith(`${a}.`);

// Values bound only by other topologies are leftovers from a topology switch.
export const dropOtherTopologyValues = (
  input: Record<string, unknown>,
  schema: TopologyUISchemas,
  selectedTopology: string
): Record<string, unknown> => {
  const selectedPaths = collectTopologyPaths(schema, selectedTopology);
  const result = deepClone(input);

  Object.keys(schema)
    .filter((topology) => topology !== selectedTopology)
    .flatMap((topology) => collectTopologyPaths(schema, topology))
    .filter(
      (path) =>
        !selectedPaths.some((selected) => isSameOrNested(path, selected))
    )
    .forEach((path) => deleteByPath(result, path));

  return result;
};
