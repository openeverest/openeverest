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

import type { Section } from '../../ui-generator.types';
import { postprocessSchemaData } from './postprocess-schema';
import {
  deepClone,
  deepMerge,
  deleteByPathAndEmptyParents,
  isPlainObject,
} from '../object-path';
import { getInactiveToggleablePaths } from '../toggleable/toggleable';

export interface MergeSectionEditParams {
  spec: Record<string, unknown>;
  formData: Record<string, unknown>;
  // Preprocessed sections of the instance's topology.
  sections: Record<string, Section>;
  sectionKey: string;
  topology?: string;
}

// Applies one section's form values to a saved spec for a full update. The
// merge keeps saved values, so a switched-off group's paths are deleted
// explicitly (with parents left empty).
export const mergeSectionEdit = ({
  spec,
  formData,
  sections,
  sectionKey,
  topology,
}: MergeSectionEditParams): Record<string, unknown> => {
  const processed = postprocessSchemaData(
    formData,
    topology
      ? { schema: { [topology]: { sections } }, selectedTopology: topology }
      : undefined
  );
  const updates = processed.spec;
  const root: Record<string, unknown> = {
    spec: isPlainObject(updates)
      ? deepMerge(deepClone(spec), updates)
      : deepClone(spec),
  };

  getInactiveToggleablePaths(
    { [sectionKey]: sections[sectionKey] },
    formData
  ).forEach((path) => deleteByPathAndEmptyParents(root, path));

  return isPlainObject(root.spec) ? root.spec : {};
};
