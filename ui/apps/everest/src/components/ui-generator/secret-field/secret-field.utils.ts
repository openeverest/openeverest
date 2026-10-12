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

import { Provider } from 'shared-types/api.types';
import {
  FieldType,
  isSection,
  Section,
  SecretFieldParams,
} from '../ui-generator.types';
import { getComponentSourcePath } from '../utils/preprocess/normalized-component';
import { walkLeafComponents } from '../utils/schema-walker/schema-walker';
import { Messages } from './secret-field.messages';

// The create form of a provider's secret definition, if the provider ships one.
export const getSecretDefinitionSection = (
  provider: Provider | undefined,
  definition: string
): Section | undefined => {
  const uiSchema = provider?.spec?.secrets?.[definition]?.uiSchema;
  return isSection(uiSchema) ? uiSchema : undefined;
};

export const isSecretEditable = ({
  readOnly,
  disabled,
}: Pick<SecretFieldParams, 'readOnly' | 'disabled'>) => !readOnly && !disabled;

// Form paths of the secret fields whose value is picked from a list.
export const getEditableSecretFields = (sections: Record<string, Section>) => {
  const fields: { path: string; definition: string }[] = [];
  Object.values(sections).forEach(({ components }) =>
    walkLeafComponents(components, ({ component }) => {
      if (component.uiType !== FieldType.Secret) return;
      const path = getComponentSourcePath(component);
      if (path && isSecretEditable(component.fieldParams)) {
        fields.push({ path, definition: component.fieldParams.definition });
      }
    })
  );
  return fields;
};

// A value that isn't a listed secret (e.g. set by a preset) stays selectable, marked once the list confirms it.
export const getSecretOptions = (
  names: string[],
  value: string,
  isListed: boolean
) => [
  ...names.map((name) => ({ label: name, value: name })),
  ...(value && !names.includes(value)
    ? [{ label: isListed ? Messages.unmanaged(value) : value, value }]
    : []),
];

export const getSecretHelperText = ({
  hasContext,
  isEditable,
  isLoading,
  isError,
  isEmpty,
  canAdd,
  helperText,
}: {
  hasContext: boolean;
  isEditable: boolean;
  isLoading: boolean;
  isError: boolean;
  isEmpty: boolean;
  canAdd: boolean;
  helperText?: string;
}) => {
  if (!hasContext) return Messages.noContext;
  // A read-only field shows its value without listing secrets.
  if (!isEditable) return helperText;
  if (isLoading) return Messages.loading;
  if (isError) return Messages.loadFailed;
  if (isEmpty) return canAdd ? Messages.emptyAddOne : Messages.empty;
  return helperText;
};
