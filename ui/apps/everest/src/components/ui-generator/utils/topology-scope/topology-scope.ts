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
import { getDefaultValues } from '../default-values';
import { mergeTopologyDefaults } from '../default-values/merge-topology-defaults';
import { withoutEmptyFormValues } from '../field-codecs';
import { deepClone, deleteByPath, isSameOrNestedPath } from '../object-path';
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
        !selectedPaths.some((selected) => isSameOrNestedPath(path, selected))
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

// The form's values once the user switches to `topology`: what only the other
// topologies bind is dropped, and the new topology's defaults fill what is
// then unset. A map or a list with no entry counts as unset, though the form
// holds an empty row for it: the row a field showed before the switch must
// not keep out the default the new topology gives it. Rows the user filled
// stay.
export const switchTopologyValues = (
  values: Record<string, unknown>,
  schema: TopologyUISchemas,
  topology: string
): Record<string, unknown> => {
  const kept = withoutEmptyFormValues(
    dropOtherTopologyValues(values, schema, topology),
    schema[topology]?.sections ?? {}
  );
  return mergeTopologyDefaults(kept, getDefaultValues(schema, topology));
};
