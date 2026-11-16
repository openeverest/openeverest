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
import { Alert } from '@mui/material';
import { AutoCompleteInput, TextInput } from '@percona/ui-lib';
import {
  getCredentialsSecretErrorMessage,
  getSecretUseProblem,
  useCanUseSecret,
  useSecretNamespacePermissions,
} from 'components/class-authentication';
import { ClassPicker } from 'components/class-picker';
import { FormDialog } from 'components/form-dialog';
import { TlsConfigFields } from 'components/tls-config-fields';
import { useSaveBackupStorage } from 'hooks/api/backup-storages';
import {
  SecretAlreadyExistsError,
  SecretChangedMeanwhileError,
} from 'hooks/api/secrets';
import {
  getBackupStorageClassName,
  toBackupStorageClassContract,
} from 'utils/backup-storage-classes';
import { buildBackupStorageSchema } from '../backup-storage-modal-schema';
import { StorageFormFields } from '../backup-storage-modal.constants';
import { Messages } from '../backup-storage-modal.messages';
import {
  BackupStorageFormProps,
  BackupStorageFormValues,
} from '../backup-storage-modal.types';
import {
  buildSaveRequest,
  getFormDefaults,
} from '../backup-storage-modal.utils';
import { ClassFields } from '../class-fields/class-fields';

export const BackupStorageForm = ({
  original,
  classes,
  namespaces,
  namespace,
  changedMeanwhile,
  onChangedMeanwhile,
  onClose,
}: BackupStorageFormProps) => {
  const isEdit = !!original;
  // Without its class the form can't tell which parameters it owns, and a
  // save would drop them.
  const classUnavailable =
    !!original &&
    !classes.some(
      (storageClass) =>
        getBackupStorageClassName(storageClass) ===
        original.spec.backupStorageClassRef.name
    );
  const { mutate, isPending } = useSaveBackupStorage();
  // Why the last save was refused before or while writing the Secret.
  const [saveProblem, setSaveProblem] = useState<string>();
  const {
    permissions: secretPermissions,
    isPermissionsLoading: secretPermissionsLoading,
  } = useSecretNamespacePermissions();
  const canUseSecret = useCanUseSecret();

  const schema = useMemo(
    () =>
      buildBackupStorageSchema({
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
        namespace: namespace ?? namespaces[0] ?? '',
        classes,
      }),
    [original, namespace, namespaces, classes]
  );
  // The class of an existing storage can't change.
  const selectableClasses = useMemo(
    () =>
      classes
        .filter(
          (storageClass) =>
            !original ||
            getBackupStorageClassName(storageClass) ===
              original.spec.backupStorageClassRef.name
        )
        .map(toBackupStorageClassContract),
    [classes, original]
  );

  const save = async (values: BackupStorageFormValues) => {
    const request = buildSaveRequest({
      values,
      storageClass: classes.find(
        (c) => getBackupStorageClassName(c) === values.className
      ),
      original,
      secretPermissions,
    });
    const useProblem = await getSecretUseProblem({
      spec: request.storage.spec,
      savedSpec: original?.spec,
      namespace: values.namespace,
      canUseSecret: (name) => canUseSecret(values.namespace, name),
    });
    setSaveProblem(useProblem);
    if (useProblem) {
      return;
    }
    mutate(request, {
      onSuccess: onClose,
      onError: (error) => {
        if (
          error instanceof SecretAlreadyExistsError ||
          error instanceof SecretChangedMeanwhileError
        ) {
          setSaveProblem(
            getCredentialsSecretErrorMessage(
              error,
              Messages.authenticationNouns
            )
          );
        } else if (
          isEdit &&
          isAxiosError(error) &&
          error.response?.status === 409
        ) {
          onChangedMeanwhile();
        }
      },
    });
  };

  return (
    <FormDialog<BackupStorageFormValues>
      isOpen
      closeModal={onClose}
      headerMessage={isEdit ? Messages.editTitle : Messages.createTitle}
      schema={schema}
      // A storage that changed meanwhile refreshes what the user left untouched.
      values={defaultValues}
      resetOptions={{ keepDirtyValues: true }}
      onSubmit={save}
      submitting={isPending}
      disableSubmit={classUnavailable}
      submitMessage={isEdit ? Messages.save : Messages.create}
      dataTestId="backup-storage"
      size="XL"
    >
      <>
        {original && classUnavailable && (
          <Alert severity="error">
            {Messages.classUnavailable(
              original.spec.backupStorageClassRef.name
            )}
          </Alert>
        )}
        {changedMeanwhile && (
          <Alert severity="warning">{Messages.changedMeanwhile}</Alert>
        )}
        <TextInput
          name={StorageFormFields.name}
          label={Messages.name}
          isRequired
          textFieldProps={{ disabled: isEdit }}
        />
        <AutoCompleteInput
          name={StorageFormFields.namespace}
          label={Messages.namespace}
          options={namespaces}
          disabled={isEdit}
          isRequired
        />
        <ClassPicker
          name={StorageFormFields.className}
          label={Messages.storageClass}
          classes={selectableClasses}
          disabled={isEdit}
        />
        <ClassFields
          classes={classes}
          original={original}
          secretPermissions={secretPermissions}
          secretPermissionsLoading={secretPermissionsLoading}
        />
        <TlsConfigFields />
        {saveProblem && (
          <Alert severity="error" sx={{ mt: 3 }}>
            {saveProblem}
          </Alert>
        )}
      </>
    </FormDialog>
  );
};
