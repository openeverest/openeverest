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

import { useMemo, useState } from 'react';
import { isAxiosError } from 'axios';
import { Alert, MenuItem } from '@mui/material';
import { AutoCompleteInput, SelectInput, TextInput } from '@percona/ui-lib';
import { FormDialog } from 'components/form-dialog';
import {
  SecretAlreadyExistsError,
  SecretChangedMeanwhileError,
  useSaveMonitoringDestination,
} from 'hooks/api/monitoring-destinations';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import {
  getMonitoringClassName,
  getMonitoringClassTitle,
  isMonitoringClassAccepted,
} from 'utils/monitoring-classes';
import { ClassFields } from '../class-fields/class-fields';
import { buildMonitoringDestinationSchema } from '../monitoring-destination-modal-schema';
import { DestinationFormFields } from '../monitoring-destination-modal.constants';
import { Messages } from '../monitoring-destination-modal.messages';
import {
  MonitoringDestinationFormProps,
  MonitoringDestinationFormValues,
} from '../monitoring-destination-modal.types';
import {
  buildSaveRequest,
  getFormDefaults,
  getSecretErrorMessage,
} from '../monitoring-destination-modal.utils';
import { TlsFields } from '../tls-fields/tls-fields';
import { RestartNotice } from '../restart-notice/restart-notice';

export const MonitoringDestinationForm = ({
  original,
  classes,
  namespaces,
  className,
  changedMeanwhile,
  onChangedMeanwhile,
  onClose,
}: MonitoringDestinationFormProps) => {
  const isEdit = !!original;
  const { mutate, isPending } = useSaveMonitoringDestination();
  const [secretError, setSecretError] = useState<
    SecretAlreadyExistsError | SecretChangedMeanwhileError
  >();
  const { canCreate, canRead, canUpdate, isPermissionsLoading } =
    useNamespacePermissionsForResource('secrets');
  const secretPermissions = useMemo(
    () => ({ canCreate, canRead, canUpdate }),
    [canCreate, canRead, canUpdate]
  );

  const schema = useMemo(
    () =>
      buildMonitoringDestinationSchema({
        classes,
        original,
        secretPermissions,
      }),
    [classes, original, secretPermissions]
  );
  const defaultValues = useMemo(
    () =>
      getFormDefaults({
        original,
        namespace: namespaces[0] ?? '',
        classes,
        className,
      }),
    [original, namespaces, classes, className]
  );
  // The class of an existing destination can't change, accepted or not.
  const selectableClasses = useMemo(
    () =>
      classes.filter((monitoringClass) =>
        original
          ? getMonitoringClassName(monitoringClass) ===
            original.spec.classRef.name
          : isMonitoringClassAccepted(monitoringClass)
      ),
    [classes, original]
  );

  const save = (values: MonitoringDestinationFormValues) => {
    setSecretError(undefined);
    mutate(
      buildSaveRequest({
        values,
        monitoringClass: classes.find(
          (c) => getMonitoringClassName(c) === values.className
        ),
        original,
        secretPermissions,
      }),
      {
        onSuccess: onClose,
        onError: (error) => {
          if (
            error instanceof SecretAlreadyExistsError ||
            error instanceof SecretChangedMeanwhileError
          ) {
            setSecretError(error);
          } else if (
            isEdit &&
            isAxiosError(error) &&
            error.response?.status === 409
          ) {
            onChangedMeanwhile();
          }
        },
      }
    );
  };

  return (
    <FormDialog<MonitoringDestinationFormValues>
      isOpen
      closeModal={onClose}
      headerMessage={isEdit ? Messages.editTitle : Messages.createTitle}
      schema={schema}
      // A destination that changed meanwhile refreshes what the user left untouched.
      values={defaultValues}
      resetOptions={{ keepDirtyValues: true }}
      onSubmit={save}
      submitting={isPending}
      submitMessage={isEdit ? Messages.save : Messages.create}
      dataTestId="monitoring-destination"
      size="XL"
    >
      <>
        {changedMeanwhile && (
          <Alert severity="warning">{Messages.changedMeanwhile}</Alert>
        )}
        {original && (
          <RestartNotice
            destination={original}
            monitoringClass={classes.find(
              (c) => getMonitoringClassName(c) === original.spec.classRef.name
            )}
          />
        )}
        <TextInput
          name={DestinationFormFields.name}
          label={Messages.name}
          isRequired
          textFieldProps={{ disabled: isEdit }}
        />
        <AutoCompleteInput
          name={DestinationFormFields.namespace}
          label={Messages.namespace}
          options={namespaces}
          disabled={isEdit}
          isRequired
        />
        <SelectInput
          name={DestinationFormFields.className}
          label={Messages.monitoringClass}
          selectFieldProps={{ disabled: isEdit }}
          isRequired
        >
          {selectableClasses.map((monitoringClass) => (
            <MenuItem
              key={getMonitoringClassName(monitoringClass)}
              value={getMonitoringClassName(monitoringClass)}
            >
              {getMonitoringClassTitle(monitoringClass)}
            </MenuItem>
          ))}
        </SelectInput>
        <ClassFields
          classes={classes}
          original={original}
          secretPermissions={secretPermissions}
          secretPermissionsLoading={isPermissionsLoading}
        />
        <TlsFields />
        {secretError && (
          <Alert severity="error" sx={{ mt: 3 }}>
            {getSecretErrorMessage(secretError)}
          </Alert>
        )}
      </>
    </FormDialog>
  );
};
