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
import { getByPath, isEmptyFieldValue, setByPath } from '../object-path';
import { walkLeafComponents } from '../schema-walker';
import {
  getComponentSourcePath,
  getComponentTargetPaths,
} from '../preprocess/normalized-component';

export interface FieldChangeInput {
  sections: Record<string, Section>;
  formData: Record<string, unknown>;
  // The form's values when it was opened, read from a stored object.
  initialValues: Record<string, unknown>;
}

const isSameFieldValue = (a: unknown, b: unknown): boolean =>
  (isEmptyFieldValue(a) && isEmptyFieldValue(b)) ||
  JSON.stringify(a) === JSON.stringify(b);

// Target paths of path-bound fields (widgets excluded), split by whether the
// user changed the field from the value the form opened with.
export const groupFieldPathsByChange = ({
  sections,
  formData,
  initialValues,
}: FieldChangeInput): { changed: string[]; untouched: string[] } => {
  const changed: string[] = [];
  const untouched: string[] = [];
  Object.values(sections).forEach((section) => {
    if (!section?.components) return;
    walkLeafComponents(section.components, ({ component }) => {
      const sourcePath = getComponentSourcePath(component);
      if (isWidgetComponent(component) || !sourcePath) return;
      const isChanged = !isSameFieldValue(
        getByPath(formData, sourcePath),
        getByPath(initialValues, sourcePath)
      );
      (isChanged ? changed : untouched).push(
        ...getComponentTargetPaths(component)
      );
    });
  });
  return { changed, untouched };
};

// Reading can change a value's form ("600m" → 0.6), so a field the user left
// as read is sent in its source form. Paths the payload dropped stay dropped.
export const keepUntouchedSourceValues = (
  payload: Record<string, unknown>,
  source: Record<string, unknown>,
  input: FieldChangeInput
): void => {
  groupFieldPathsByChange(input).untouched.forEach((path) => {
    const sourceValue = getByPath(source, path);
    if (sourceValue !== undefined && getByPath(payload, path) !== undefined) {
      setByPath(payload, path, sourceValue);
    }
  });
};
