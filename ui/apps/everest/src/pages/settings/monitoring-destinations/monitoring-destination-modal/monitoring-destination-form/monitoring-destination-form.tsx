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
import { Alert, Button, MenuItem } from '@mui/material';
import { AutoCompleteInput, SelectInput, TextInput } from '@percona/ui-lib';
import { FormDialog } from 'components/form-dialog';
import {
  SecretAlreadyExistsError,
  useSaveMonitoringDestination,
} from 'hooks/api/monitoring-destinations';
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
} from '../monitoring-destination-modal.utils';
import { TlsFields } from '../tls-fields/tls-fields';

export const MonitoringDestinationForm = ({
  original,
  classes,
  namespaces,
  changedMeanwhile,
  onChangedMeanwhile,
  onClose,
}: MonitoringDestinationFormProps) => {
  const isEdit = !!original;
  const { mutate, isPending } = useSaveMonitoringDestination();
  const [conflictingSecret, setConflictingSecret] = useState<string>();

  const schema = useMemo(
    () => buildMonitoringDestinationSchema({ classes, original }),
    [classes, original]
  );
  const defaultValues = useMemo(
    () =>
      getFormDefaults({ original, namespace: namespaces[0] ?? '', classes }),
    [original, namespaces, classes]
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

  const save = (
    values: MonitoringDestinationFormValues,
    reuseSecretName?: string
  ) => {
    setConflictingSecret(undefined);
    mutate(
      buildSaveRequest({
        values,
        monitoringClass: classes.find(
          (c) => getMonitoringClassName(c) === values.className
        ),
        original,
        reuseSecretName,
      }),
      {
        onSuccess: onClose,
        onError: (error) => {
          if (error instanceof SecretAlreadyExistsError) {
            setConflictingSecret(error.secretName);
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
      defaultValues={defaultValues}
      onSubmit={(values) => save(values)}
      submitting={isPending}
      submitMessage={isEdit ? Messages.save : Messages.create}
      dataTestId="monitoring-destination"
      size="XL"
    >
      {({ handleSubmit }) => (
        <>
          {changedMeanwhile && (
            <Alert severity="warning">{Messages.changedMeanwhile}</Alert>
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
          <ClassFields classes={classes} original={original} />
          <TlsFields />
          {conflictingSecret && (
            <Alert
              severity="error"
              sx={{ mt: 3 }}
              action={
                <Button
                  color="inherit"
                  size="small"
                  onClick={handleSubmit((values) =>
                    save(values, conflictingSecret)
                  )}
                >
                  {Messages.reuseSecret}
                </Button>
              }
            >
              {Messages.secretExists(conflictingSecret)}
            </Alert>
          )}
        </>
      )}
    </FormDialog>
  );
};
