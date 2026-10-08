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

import { useSecrets } from 'hooks/api/secrets';
import { useClusterName } from 'hooks/api/useClusterName';
import UIComponent from '../ui-component/ui-component';
import { useUiGeneratorContext } from '../ui-generator-context';
import { Component, FieldType } from '../ui-generator.types';
import { Messages } from './secret-field.messages';
import { SecretFieldProps } from './secret-field.types';

type SelectComponent = Extract<Component, { uiType: FieldType.Select }>;

export const SecretField = ({ item, name }: SecretFieldProps) => {
  const { providerObject, namespace } = useUiGeneratorContext();
  const cluster = useClusterName();
  const { definition, ...fieldParams } = item.fieldParams;
  const provider = providerObject?.metadata?.name;
  // Secrets are scoped to a provider and namespace, which a playground lacks.
  const hasContext = !!provider && !!namespace;

  const {
    data: secrets = [],
    isLoading,
    isError,
  } = useSecrets(
    cluster,
    namespace ?? '',
    { provider, definition },
    { enabled: hasContext }
  );

  const helperText = !hasContext
    ? Messages.noContext
    : isLoading
      ? Messages.loading
      : isError
        ? Messages.loadFailed
        : fieldParams.helperText;

  const selectItem: SelectComponent = {
    ...item,
    uiType: FieldType.Select,
    fieldParams: {
      ...fieldParams,
      options: secrets.flatMap(({ metadata }) =>
        metadata?.name ? [{ label: metadata.name, value: metadata.name }] : []
      ),
      disabled: !hasContext || isLoading || isError || fieldParams.disabled,
      helperText,
    },
  };

  return <UIComponent item={selectItem} name={name} />;
};
