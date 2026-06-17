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

import { useContext, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { ScheduleFormDialog } from 'components/schedule-form-dialog/schedule-form-dialog';
import { ScheduleFormDialogContext } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog.context';
import { ScheduleModalContext } from '../backups.context';
import { useUpdateDbInstanceWithConflictRetry } from 'hooks/api/db-instances/useUpdateDbInstance';
import { useBackupClassesList } from 'hooks/api/backup-classes/useBackupClasses';
import { useBackupsList } from 'hooks/api/backups/useBackups';
import { useClusterName } from 'hooks/api/useClusterName';
import { ScheduleFormData } from 'components/schedule-form-dialog/schedule-form/schedule-form-schema';
import { getSchedulesPayload } from 'components/schedule-form-dialog/schedule-form/schedule-form.utils';
import { Instance } from 'shared-types/api.types';
import {
  flattenSchedules,
  applySchedulesToStorages,
  removeUnusedStorages,
} from '../backups.utils';

export const ScheduledBackupModal = () => {
  const {
    mode,
    setMode,
    selectedScheduleName,
    setSelectedScheduleName,
    openScheduleModal,
    setOpenScheduleModal,
    instance,
  } = useContext(ScheduleModalContext);

  const clusterName = useClusterName();
  const { instanceName = '' } = useParams();
  const { data: backupClasses = [] } = useBackupClassesList(clusterName);
  const classRef = instance.spec?.backup?.classRef?.name;
  const providerType = instance.spec?.provider;

  const availableBackupClasses = useMemo(
    () =>
      backupClasses.filter((bc) => {
        // Schedules always require ProviderManaged execution mode.
        if (bc.spec?.executionMode !== 'ProviderManaged') return false;
        const supported = bc.spec?.supportedProviders;
        if (!supported || supported.length === 0) return true;
        if (!providerType) return true;
        return supported.includes(providerType);
      }),
    [backupClasses, providerType]
  );

  const backupClass = useMemo(
    () =>
      availableBackupClasses.find((bc) => bc.metadata?.name === classRef) ??
      availableBackupClasses[0],
    [availableBackupClasses, classRef]
  );

  const { mutate: updateInstance, isPending } =
    useUpdateDbInstanceWithConflictRetry(instance, {
      onSuccess: () => setOpenScheduleModal(false),
    });

  const namespace = instance.metadata?.namespace ?? '';
  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    instanceName
  );
  const liveStorages = useMemo(
    () => removeUnusedStorages(instance.spec?.backup?.storages ?? [], backups),
    [instance, backups]
  );

  const disableClassSelection = !!classRef && liveStorages.length > 0;

  const schedules = useMemo(() => flattenSchedules(instance), [instance]);

  const handleSubmit = (data: ScheduleFormData) => {
    const updatedSchedules = getSchedulesPayload({
      formData: data,
      mode,
      schedules,
    });

    const updatedStorages = applySchedulesToStorages(
      instance,
      updatedSchedules
    );

    // If creating a schedule for a new storage not yet in the array, add it.
    const newStorageName =
      typeof data.storageLocation === 'string'
        ? data.storageLocation
        : data.storageLocation?.metadata?.name;
    const storageExists = updatedStorages.some(
      (s) => s.storageRef.name === newStorageName
    );
    if (!storageExists && newStorageName) {
      updatedStorages.push({
        name: newStorageName,
        storageRef: { name: newStorageName },
        schedules: updatedSchedules
          .filter((s) => s.storageName === newStorageName)
          .map((schedule) => ({
            name: schedule.name,
            cron: schedule.cron,
            enabled: schedule.enabled,
            retentionCopies: schedule.retentionCopies,
            ...(schedule.config && {
              config: schedule.config as Record<string, never>,
            }),
          })),
      });
    }

    const updatedInstance: Instance = {
      ...instance,
      spec: {
        ...instance.spec,
        backup: {
          ...instance.spec?.backup,
          classRef: classRef
            ? { name: classRef }
            : { name: data.backupClassName ?? '' },
          enabled: instance.spec?.backup?.enabled ?? true,
          storages: updatedStorages,
        },
      },
    };

    updateInstance(updatedInstance);
  };

  const handleClose = () => setOpenScheduleModal(false);

  if (!openScheduleModal) return null;

  return (
    <ScheduleFormDialogContext.Provider
      value={{
        mode,
        setMode,
        handleSubmit,
        handleClose,
        isPending,
        selectedScheduleName,
        setSelectedScheduleName,
        openScheduleModal,
        setOpenScheduleModal,
        externalContext: 'db-details-backups',
        dbInstanceInfo: {
          dbInstanceName: instance.metadata?.name,
          namespace,
          schedules,
          defaultSchedules: schedules,
          backupClass,
          availableBackupClasses,
          disableClassSelection,
          instanceStorageNames: liveStorages
            .map((s) => s.storageRef.name)
            .filter((name): name is string => !!name),
        },
      }}
    >
      <ScheduleFormDialog />
    </ScheduleFormDialogContext.Provider>
  );
};
