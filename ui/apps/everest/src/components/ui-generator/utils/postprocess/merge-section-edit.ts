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
  deepClone,
  deleteByPathAndEmptyParents,
  getByPath,
  isEmptyFieldValue,
  isPlainObject,
  setByPath,
} from '../object-path';
import { getInactiveToggleablePaths } from '../toggleable/toggleable';
import { walkLeafComponents } from '../schema-walker';
import {
  getComponentSourcePath,
  getComponentTargetPaths,
} from '../preprocess/normalized-component';
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

export interface MergeSectionEditParams {
  spec: Record<string, unknown>;
  formData: Record<string, unknown>;
  // The form's values when it was opened (read from the saved spec).
  initialValues: Record<string, unknown>;
  // Preprocessed sections of the instance's topology.
  sections: Record<string, Section>;
  sectionKey: string;
  topology?: string;
}

const isSameFieldValue = (a: unknown, b: unknown): boolean =>
  (isEmptyFieldValue(a) && isEmptyFieldValue(b)) ||
  JSON.stringify(a) === JSON.stringify(b);

// Reading a value can change its form ("600m" → 0.6), so an untouched field must
// keep the saved value instead of being written back.
const applyChangedFields = (
  root: Record<string, unknown>,
  processed: Record<string, unknown>,
  { formData, initialValues, sections }: MergeSectionEditParams
) => {
  Object.values(sections).forEach((section) => {
    if (!section?.components) return;
    walkLeafComponents(section.components, ({ component }) => {
      const sourcePath = getComponentSourcePath(component);
      if (
        isWidgetComponent(component) ||
        !sourcePath ||
        isSameFieldValue(
          getByPath(formData, sourcePath),
          getByPath(initialValues, sourcePath)
        )
      ) {
        return;
      }
      getComponentTargetPaths(component).forEach((path) => {
        const value = getByPath(processed, path);
        if (value !== undefined) setByPath(root, path, value);
      });
    });
  });
};

// Applies the fields the user changed to a saved spec for a full update. The
// rest of the spec is kept as saved, so a switched-off group's paths are deleted
// explicitly (with parents left empty), and widget values are replaced whole.
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
  applyChangedFields(root, processed, params);

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
