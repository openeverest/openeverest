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

import type { Section } from 'components/ui-generator/ui-generator.types';
import { generateFieldId } from 'components/ui-generator/utils/component-renderer/generate-field-id';
import { isPlainObject } from 'components/ui-generator/utils/object-path/object-path';
import { walkLeafComponents } from 'components/ui-generator/utils/schema-walker/schema-walker';
import type {
  OutsideFieldError,
  SchemaFieldIndex,
} from './outside-field-errors.types';

export const indexSchemaFields = (
  sections: Record<string, Section>,
  dialogSectionKey: string
): SchemaFieldIndex => {
  const index: SchemaFieldIndex = {
    shownInDialog: new Set(),
    labels: new Map(),
  };

  Object.entries(sections).forEach(([sectionKey, section]) => {
    if (!section?.components) return;

    walkLeafComponents(
      section.components,
      ({ component, generatedName }) => {
        const fieldId = generateFieldId(component, generatedName);
        // The walker skips hidden fields, so one in the dialog's own section
        // counts as outside: it has no input to carry its error either.
        if (sectionKey === dialogSectionKey) {
          index.shownInDialog.add(fieldId);
        }
        if (!index.labels.has(fieldId)) {
          index.labels.set(fieldId, {
            sectionLabel: section.label ?? sectionKey,
            fieldLabel: component.fieldParams?.label,
          });
        }
      },
      sectionKey
    );
  });

  return index;
};

// react-hook-form nests errors by path segment; a node with a string
// `message` is the error of the field at that path.
const flattenErrors = (
  errors: unknown,
  path: string[] = []
): { fieldId: string; message: string }[] => {
  if (isPlainObject(errors) && typeof errors.message === 'string') {
    return [{ fieldId: path.join('.'), message: errors.message }];
  }
  if (!isPlainObject(errors) && !Array.isArray(errors)) return [];

  return Object.entries(errors).flatMap(([segment, nested]) =>
    flattenErrors(nested, [...path, segment])
  );
};

export const collectOutsideFieldErrors = (
  errors: unknown,
  { shownInDialog, labels }: SchemaFieldIndex
): OutsideFieldError[] =>
  flattenErrors(errors)
    .filter(({ fieldId }) => !shownInDialog.has(fieldId))
    .map((error) => ({ ...error, ...labels.get(error.fieldId) }));
