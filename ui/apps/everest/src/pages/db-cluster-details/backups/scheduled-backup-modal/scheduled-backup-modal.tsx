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
import { ScheduleFormDialog } from 'components/schedule-form-dialog/schedule-form-dialog';
import { ScheduleFormDialogContext } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog.context';
import { ScheduleModalContext } from '../backups.context';
import { useUpdateDbInstanceWithConflictRetry } from 'hooks/api/db-instances/useUpdateDbInstance';
import { useBackupClassesList } from 'hooks/api/backup-classes/useBackupClasses';
import { useClusterName } from 'hooks/api/useClusterName';
import { ScheduleFormData } from 'components/schedule-form-dialog/schedule-form/schedule-form-schema';
import { getSchedulesPayload } from 'components/schedule-form-dialog/schedule-form/schedule-form.utils';
import { FlattenedSchedule } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';
import { Instance } from 'shared-types/api.types';

/** Flatten all schedules from every storage on the Instance, annotating with storageName. */
const flattenSchedules = (instance: Instance): FlattenedSchedule[] =>
  (instance.spec?.backup?.storages ?? []).flatMap((storage) =>
    (storage.schedules ?? []).map((schedule) => ({
      name: schedule.name,
      cron: schedule.cron,
      enabled: schedule.enabled,
      retentionCopies: schedule.retentionCopies,
      config: schedule.config as Record<string, unknown> | undefined,
      storageName: storage.storageRef.name ?? '',
    }))
  );

/** Rebuild Instance storages array from the flat schedules list. */
const buildStoragesFromSchedules = (
  instance: Instance,
  schedules: FlattenedSchedule[]
): NonNullable<
  NonNullable<NonNullable<Instance['spec']>['backup']>['storages']
> => {
  const existingStorages = instance.spec?.backup?.storages ?? [];
  return existingStorages.map((storage) => ({
    ...storage,
    schedules: schedules
      .filter((s) => s.storageName === (storage.storageRef.name ?? ''))
      .map((schedule) => ({
        name: schedule.name,
        cron: schedule.cron,
        enabled: schedule.enabled,
        retentionCopies: schedule.retentionCopies,
        ...(schedule.config
          ? { config: schedule.config as Record<string, never> }
          : {}),
      })),
  }));
};

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
  const { data: backupClasses = [] } = useBackupClassesList(clusterName);
  const classRef = instance.spec?.backup?.classRef?.name;
  const backupClass = useMemo(
    () => backupClasses.find((bc) => bc.metadata?.name === classRef),
    [backupClasses, classRef]
  );

  const { mutate: updateInstance, isPending } =
    useUpdateDbInstanceWithConflictRetry(instance, {
      onSuccess: () => setOpenScheduleModal(false),
    });

  const namespace = instance.metadata?.namespace ?? '';
  const schedules = useMemo(() => flattenSchedules(instance), [instance]);

  const handleSubmit = (data: ScheduleFormData) => {
    const updatedSchedules = getSchedulesPayload({
      formData: data,
      mode,
      schedules,
    });

    const updatedStorages = buildStoragesFromSchedules(
      instance,
      updatedSchedules
    );

    // If creating a schedule for a new storage not yet in the array, add it.
    const newStorageName =
      typeof data.storageLocation === 'string'
        ? data.storageLocation
        : data.storageLocation?.name;
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
          classRef: instance.spec?.backup?.classRef ?? { name: '' },
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
        },
      }}
    >
      <ScheduleFormDialog />
    </ScheduleFormDialogContext.Provider>
  );
};
