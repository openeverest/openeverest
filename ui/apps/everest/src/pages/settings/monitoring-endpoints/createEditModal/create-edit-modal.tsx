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

import { AutoCompleteInput, TextInput } from '@percona/ui-lib';
import { FormDialog } from 'components/form-dialog';
import TlsAlert from 'components/tls-alert';
import TlsCheckbox from 'components/tls-checkbox';
import { useMemo } from 'react';
import { Messages } from '../monitoring-endpoints.messages';
import {
  CreateEditEndpointModalProps,
  EndpointFormFields,
  EndpointFormType,
  endpointDefaultValues,
  getEndpointSchema,
} from './create-edit-modal.types';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import { HiddenInput } from 'components/hidden-input';

export const CreateEditEndpointModal = ({
  open,
  handleClose,
  isLoading = false,
  handleSubmit,
  selectedEndpoint,
  selectedNamespace,
}: CreateEditEndpointModalProps) => {
  const isEditMode = !!selectedEndpoint;

  const { canCreate } =
    useNamespacePermissionsForResource('monitoring-configs');
  const endpointSchema = getEndpointSchema(isEditMode);

  const defaultValues = useMemo(
    () =>
      selectedEndpoint
        ? {
            ...endpointDefaultValues,
            name: selectedEndpoint.metadata?.name ?? '',
            namespace: selectedNamespace ?? '',
            url: selectedEndpoint.spec.url,
            verifyTLS: selectedEndpoint.spec.verifyTLS ?? true,
          }
        : endpointDefaultValues,
    [selectedEndpoint, selectedNamespace]
  );

  const onSubmit = (data: EndpointFormType) => {
    handleSubmit(isEditMode, data);
  };

  return (
    <FormDialog
      size="XL"
      isOpen={open}
      closeModal={handleClose}
      submitting={isLoading}
      onSubmit={onSubmit}
      defaultValues={defaultValues}
      headerMessage={Messages.addEditDialogHeader(isEditMode)}
      schema={endpointSchema}
      submitMessage={Messages.addEditDialogSubmitButton(isEditMode)}
    >
      {({ watch }) => (
        <>
          <TextInput
            name={EndpointFormFields.name}
            label={Messages.fieldLabels.name}
            isRequired
            textFieldProps={{
              disabled: isEditMode,
              placeholder: Messages.fieldPlaceholders.name,
            }}
          />
          <AutoCompleteInput
            name={EndpointFormFields.namespace}
            label={Messages.fieldLabels.namespace}
            options={canCreate}
            disabled={isEditMode}
            isRequired
            textFieldProps={{
              placeholder: Messages.fieldPlaceholders.namespaces,
            }}
          />
          <TextInput
            name={EndpointFormFields.url}
            label={Messages.fieldLabels.endpoint}
            isRequired
            textFieldProps={{
              placeholder: Messages.fieldPlaceholders.endpoint,
              onBlur: (event) => event.target.value.replace(/\/+$/, ''),
            }}
          />
          <TextInput
            name={EndpointFormFields.user}
            label={Messages.fieldLabels.user}
            isRequired={!isEditMode}
            {...(isEditMode && {
              controllerProps: {
                rules: {
                  deps: [EndpointFormFields.password],
                },
              },
            })}
            textFieldProps={{
              placeholder: Messages.fieldPlaceholders.user,
            }}
          />
          <HiddenInput
            name="password"
            isRequired={!isEditMode}
            textFieldProps={{
              label: Messages.fieldLabels.password,
              placeholder: Messages.fieldPlaceholders.password,
            }}
            {...(isEditMode && {
              controllerProps: {
                rules: {
                  deps: [EndpointFormFields.user],
                },
              },
            })}
          />
          <TlsCheckbox formControlLabelProps={{ sx: { mt: 2 } }} />
          {!watch(EndpointFormFields.verifyTLS) && <TlsAlert sx={{ mt: 2 }} />}
        </>
      )}
    </FormDialog>
  );
};
