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

import { FormDialog } from 'components/form-dialog';
import {
  getBackupListQueryKey,
  useBackupsList,
  useCreateBackupOnDemand,
} from 'hooks/api/backups/useBackups.ts';
import {
  useBackupClassesList,
  useBackupClassUiSchema,
} from 'hooks/api/backup-classes/useBackupClasses.ts';
import { useContext, useMemo } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { CreateBackupPayload } from 'shared-types/backups.types.ts';
import { Instance } from 'shared-types/api.types.ts';
import { removeEmptyFieldValues } from 'components/ui-generator/utils/postprocess/postprocess-schema.ts';
import { OnDemandBackupFieldsWrapper } from './on-demand-backup-fields-wrapper.tsx';
import {
  BackupFormData,
  defaultValuesFc,
  schema,
} from './on-demand-backup-modal.types.ts';
import { ScheduleModalContext } from '../backups.context.ts';
import { CircularProgress, Typography } from '@mui/material';
import { useClusterName } from 'hooks/api/useClusterName.ts';
import { useUpdateDbInstanceWithConflictRetry } from 'hooks/api/db-instances/useUpdateDbInstance.ts';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages.ts';

export const OnDemandBackupModal = () => {
  const queryClient = useQueryClient();
  const { instanceName = '', namespace = '' } = useParams();
  const clusterName = useClusterName();

  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    instanceName
  );
  const backupNames = backups.map((item) => item.metadata?.name ?? '');
  const { mutate: createBackupOnDemand, isPending: creatingBackup } =
    useCreateBackupOnDemand(clusterName, namespace);

  const { openOnDemandModal, setOpenOnDemandModal, instance } =
    useContext(ScheduleModalContext);

  // Fetch backup storages at modal level to compute initial form defaults.
  const { isLoading: loadingStorages } =
    useBackupStoragesByNamespace(namespace);

  // Resolve BackupClass uiSchema for the currently selected class.
  // For schema building we use the instance's default class from the list.
  const instanceClassRef = instance.spec?.backup?.classRef?.name;
  const { data: backupClasses = [], isLoading: loadingClasses } =
    useBackupClassesList(clusterName);
  const defaultClass = useMemo(
    () => backupClasses.find((bc) => bc.metadata?.name === instanceClassRef),
    [backupClasses, instanceClassRef]
  );
  const { sections: backupSections } = useBackupClassUiSchema(defaultClass);

  const { mutate: updateInstance, isPending: updatingInstance } =
    useUpdateDbInstanceWithConflictRetry(instance);

  // Wait for essential data before rendering the form.
  const dataReady = !loadingStorages && !loadingClasses;

  // Compute defaultValues once data is ready. Storage auto-fills via
  // BackupStoragesInput useEffect since storages are already loaded.
  const values = useMemo(() => defaultValuesFc(), []);

  const createBackup = (data: BackupFormData) => {
    // UIGenerator fields use `path` from the uiSchema as their form field names,
    // so they appear at the top level of form data alongside static fields.
    // Extract them by removing the known static keys.
    const staticKeys = new Set(['name', 'backupClassName', 'storageName']);
    const dynamicFields = Object.fromEntries(
      Object.entries(data).filter(([key]) => !staticKeys.has(key))
    );
    const cleanedConfig =
      Object.keys(dynamicFields).length > 0
        ? removeEmptyFieldValues(dynamicFields)
        : undefined;

    createBackupOnDemand(
      {
        metadata: { name: data.name },
        spec: {
          instanceName: instanceName,
          backupClassName: data.backupClassName,
          storageName: data.storageName,
          ...(cleanedConfig &&
            Object.keys(cleanedConfig).length > 0 && {
              config: cleanedConfig,
            }),
        },
        // The API accepts a partial Backup object for creation; generated types
        // require the full shape, so we cast through unknown.
      } as unknown as CreateBackupPayload,
      {
        onSuccess() {
          queryClient.invalidateQueries({
            queryKey: getBackupListQueryKey(
              clusterName,
              namespace,
              instanceName
            ),
          });
          setOpenOnDemandModal(false);
        },
      }
    );
  };

  const handleSubmit = (data: BackupFormData) => {
    const existingStorages = instance.spec?.backup?.storages ?? [];
    const alreadyRegistered = existingStorages.some(
      (s) => s.storageRef.name === data.storageName
    );

    if (alreadyRegistered) {
      // Storage is already registered on the instance — just create the backup.
      createBackup(data);
      return;
    }

    // Auto-register the storage on the instance, then create the backup.
    const newStorage = {
      name: data.storageName,
      storageRef: { name: data.storageName },
      main: existingStorages.length === 0,
    };

    const updatedInstance: Instance = {
      ...instance,
      spec: {
        ...instance.spec,
        backup: {
          classRef: instance.spec?.backup?.classRef ?? {
            name: data.backupClassName,
          },
          enabled: true,
          storages: [...existingStorages, newStorage],
        },
      },
    };

    updateInstance(updatedInstance, {
      onSuccess() {
        createBackup(data);
      },
    });
  };

  return (
    <FormDialog
      isOpen={openOnDemandModal}
      closeModal={() => setOpenOnDemandModal(false)}
      headerMessage="Create on-demand backup"
      onSubmit={handleSubmit}
      submitting={creatingBackup || updatingInstance}
      submitMessage="Create"
      schema={schema(backupNames, backupSections)}
      defaultValues={values}
      size="XL"
      disableSubmit={!dataReady}
    >
      {dataReady ? (
        <>
          <Typography variant="body1">
            Select a backup class and storage to create an on-demand backup.
          </Typography>
          <OnDemandBackupFieldsWrapper />
        </>
      ) : (
        <CircularProgress sx={{ display: 'block', mx: 'auto', my: 4 }} />
      )}
    </FormDialog>
  );
};
