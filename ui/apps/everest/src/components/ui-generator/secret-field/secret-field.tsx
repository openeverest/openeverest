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

import { useState } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import { ActionableLabeledContent } from '@percona/ui-lib';
import { useSecrets } from 'hooks/api/secrets';
import { useClusterName } from 'hooks/api/useClusterName';
import { useRBACPermissions } from 'hooks/rbac';
import UIComponent from '../ui-component/ui-component';
import { useUiGeneratorContext } from '../ui-generator-context';
import { Component, FieldType, FormMode } from '../ui-generator.types';
import { SecretCreateModal } from './secret-create-modal';
import {
  getSecretDefinitionSection,
  getSecretHelperText,
  getSecretOptions,
} from './secret-field.utils';
import { SecretFieldProps } from './secret-field.types';

type SelectComponent = Extract<Component, { uiType: FieldType.Select }>;

export const SecretField = ({ item, name }: SecretFieldProps) => {
  const { providerObject, namespace, formMode, generator } =
    useUiGeneratorContext();
  const cluster = useClusterName();
  const { setValue } = useFormContext();
  const value: unknown = useWatch({ name });
  const [isCreating, setIsCreating] = useState(false);
  const { definition, createLabel, label, ...fieldParams } = item.fieldParams;
  const provider = providerObject?.metadata?.name;
  // Secrets are scoped to a provider and namespace, which a playground lacks.
  const hasContext = !!provider && !!namespace;
  const isEditable = !fieldParams.readOnly && !fieldParams.disabled;
  const createForm = getSecretDefinitionSection(providerObject, definition);
  const { canCreate } = useRBACPermissions('secrets', `${namespace}/*`);
  const canAdd =
    hasContext &&
    isEditable &&
    !!createForm &&
    canCreate &&
    formMode !== FormMode.Edit;

  const {
    data: secrets = [],
    isLoading,
    isError,
    isSuccess,
  } = useSecrets(
    cluster,
    namespace ?? '',
    { provider, definition },
    { enabled: hasContext && isEditable }
  );
  const names = secrets.flatMap(({ metadata }) =>
    metadata?.name ? [metadata.name] : []
  );

  const selectItem: SelectComponent = {
    ...item,
    uiType: FieldType.Select,
    fieldParams: {
      ...fieldParams,
      options: getSecretOptions(
        names,
        typeof value === 'string' ? value : '',
        isSuccess
      ),
      disabled: !hasContext || isLoading || isError || fieldParams.disabled,
      helperText: getSecretHelperText({
        hasContext,
        isEditable,
        isLoading,
        isError,
        isEmpty: names.length === 0,
        canAdd,
        helperText: fieldParams.helperText,
      }),
    },
  };

  const handleCreated = (secretName: string) => {
    setValue(name, secretName, { shouldDirty: true, shouldValidate: true });
    setIsCreating(false);
  };

  return (
    <ActionableLabeledContent
      label={label}
      // The label row labels the select, so the select sits right under it.
      horizontalStackSx={{ marginBottom: 1 }}
      verticalStackSx={{ '.MuiFormControl-root': { mt: 0 } }}
      actionButtonProps={
        canAdd
          ? { buttonText: createLabel, onClick: () => setIsCreating(true) }
          : undefined
      }
    >
      <UIComponent item={selectItem} name={name} />
      {isCreating && createForm && generator && providerObject && namespace && (
        <SecretCreateModal
          definition={definition}
          section={createForm}
          providerObject={providerObject}
          namespace={namespace}
          generator={generator}
          onClose={() => setIsCreating(false)}
          onCreated={handleCreated}
        />
      )}
    </ActionableLabeledContent>
  );
};
