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

import { type Section, isWidgetComponent } from '../../ui-generator.types';
import { postprocessSchemaData } from './postprocess-schema';
import {
  type FieldChangeInput,
  groupFieldPathsByChange,
} from './field-changes';
import {
  deepClone,
  deleteByPathAndEmptyParents,
  getByPath,
  isPlainObject,
  setByPath,
} from '../object-path';
import { getInactiveToggleablePaths } from '../toggleable/toggleable';
import { walkLeafComponents } from '../schema-walker';
import { getComponentSourcePath } from '../preprocess/normalized-component';
import { getWidgetTargetPaths } from '../widget-targets';

const collectWidgetPaths = (section: Section | undefined): string[] => {
  const paths: string[] = [];
  if (!section?.components) return paths;
  walkLeafComponents(section.components, ({ component }) => {
    const path = isWidgetComponent(component)
      ? getComponentSourcePath(component)
      : undefined;
    if (path) paths.push(path);
    paths.push(...getWidgetTargetPaths(component));
  });
  return paths;
};

export interface MergeSectionEditParams extends FieldChangeInput {
  spec: Record<string, unknown>;
  sectionKey: string;
  topology?: string;
}

// Applies the fields the user changed to a saved spec for a full update. An
// untouched field keeps its saved form ("600m", not the 0.6 it reads as), a
// switched-off group's paths are deleted explicitly (with parents left empty),
// and widget values are replaced whole.
export const mergeSectionEdit = (
  params: MergeSectionEditParams
): Record<string, unknown> => {
  const { spec, formData, sections, sectionKey, topology } = params;
  const processed = postprocessSchemaData(
    formData,
    topology
      ? { schema: { [topology]: { sections } }, selectedTopology: topology }
      : undefined
  );
  const root: Record<string, unknown> = { spec: deepClone(spec) };
  groupFieldPathsByChange(params).changed.forEach((path) => {
    const value = getByPath(processed, path);
    if (value !== undefined) setByPath(root, path, value);
  });

  getInactiveToggleablePaths(
    { [sectionKey]: sections[sectionKey] },
    formData
  ).forEach((path) => deleteByPathAndEmptyParents(root, path));

  collectWidgetPaths(sections[sectionKey]).forEach((path) => {
    const value = getByPath(processed, path);
    if (value === undefined) {
      deleteByPathAndEmptyParents(root, path);
    } else {
      setByPath(root, path, value);
    }
  });

  return isPlainObject(root.spec) ? root.spec : {};
};
