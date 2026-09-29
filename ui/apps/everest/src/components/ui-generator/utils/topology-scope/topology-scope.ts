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
import { collectToggleableMetas } from '../toggleable/toggleable';

const collectTopologyPaths = (
  schema: TopologyUISchemas,
  topology: string
): string[] =>
  Array.from(collectAllSchemaPaths(schema[topology]?.sections ?? {}));

const collectTopologySwitches = (
  schema: TopologyUISchemas,
  topology: string
): string[] =>
  collectToggleableMetas(schema[topology]?.sections ?? {}).map(
    (meta) => meta.switchName
  );

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

  // A switch shared by name keeps the user's choice; the rest start off again.
  const selectedSwitches = collectTopologySwitches(schema, selectedTopology);
  Object.keys(schema)
    .filter((topology) => topology !== selectedTopology)
    .flatMap((topology) => collectTopologySwitches(schema, topology))
    .filter((switchName) => !selectedSwitches.includes(switchName))
    .forEach((switchName) => deleteByPath(result, switchName));

  return result;
};
