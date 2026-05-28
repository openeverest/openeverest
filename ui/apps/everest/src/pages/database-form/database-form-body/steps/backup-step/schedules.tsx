// everest
// Copyright (C) 2023 Percona LLC
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

import { Stack, Typography } from '@mui/material';
import EditableItem from 'components/editable-item/editable-item';
import { Messages } from './schedules.messages';
import { useEffect, useMemo, useState } from 'react';
import { DbWizardFormFields } from 'consts.ts';
import { useFormContext } from 'react-hook-form';
import {
  getSchedulesPayload,
  removeScheduleFromArray,
} from 'components/schedule-form-dialog/schedule-form/schedule-form.utils';
import { ScheduleContent } from './schedule-body';
import { ScheduleFormDialog } from 'components/schedule-form-dialog';
import { ScheduleFormDialogContext } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog.context';
import { ScheduleFormData } from 'components/schedule-form-dialog/schedule-form/schedule-form-schema';
import { ActionableLabeledContent } from '@percona/ui-lib';
import { useDatabasePageMode } from '../../../hooks/use-database-page-mode';
import {
  dbWizardToScheduleFormDialogMap,
  FlattenedSchedule,
} from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';
import { ScheduleWizardMode, WizardMode } from 'shared-types/wizard.types';
import { BackupStorageCRD } from 'shared-types/backupStorages.types';
import { useBackupClassesList } from 'hooks/api/backup-classes/useBackupClasses';
import { useClusterName } from 'hooks/api/useClusterName';

/** Form field path where wizard stores the flat schedules array. */
export const BACKUP_SCHEDULES_FIELD = 'backup.schedules';
/** Form field path for backup class reference name. */
export const BACKUP_CLASS_REF_FIELD = 'backup.classRef.name';

type Props = {
  backupStorages: BackupStorageCRD[];
};

export const Schedules = ({ backupStorages }: Props) => {
  const { watch, setValue } = useFormContext();
  const dbWizardMode = useDatabasePageMode();
  const clusterName = useClusterName();
  const { data: backupClasses = [] } = useBackupClassesList(clusterName);

  const [openScheduleModal, setOpenScheduleModal] = useState(false);
  const [mode, setMode] = useState<ScheduleWizardMode>(WizardMode.New);
  const [selectedScheduleName, setSelectedScheduleName] = useState<string>('');

  const k8sNamespace: string = watch(DbWizardFormFields.k8sNamespace);
  const dbName: string = watch(DbWizardFormFields.dbName);
  const formSchedules: FlattenedSchedule[] =
    watch(BACKUP_SCHEDULES_FIELD) ?? [];

  // In wizard mode, the first backup class is auto-selected.
  // User can change it inside the modal (BackupConfigFields).
  const selectedClassName: string | undefined = watch(BACKUP_CLASS_REF_FIELD);
  const backupClass = useMemo(
    () =>
      backupClasses.find((bc) => bc.metadata?.name === selectedClassName) ??
      backupClasses[0],
    [backupClasses, selectedClassName]
  );

  // Auto-set the backup class if not yet selected
  useEffect(() => {
    if (
      !selectedClassName &&
      backupClasses.length > 0 &&
      backupClasses[0]?.metadata?.name
    ) {
      setValue(BACKUP_CLASS_REF_FIELD, backupClasses[0].metadata.name);
    }
  }, [backupClasses, selectedClassName, setValue]);

  const createButtonDisabled = openScheduleModal || backupStorages.length === 0;

  const handleDelete = (name: string) => {
    setValue(
      BACKUP_SCHEDULES_FIELD,
      removeScheduleFromArray(name, formSchedules)
    );
  };

  const handleEdit = (name: string) => {
    setSelectedScheduleName(name);
    setMode(WizardMode.Edit);
    setOpenScheduleModal(true);
  };

  const handleCreate = () => {
    setMode(WizardMode.New);
    setOpenScheduleModal(true);
  };

  const handleSubmit = (data: ScheduleFormData) => {
    const updatedSchedulesArray = getSchedulesPayload({
      formData: data,
      mode,
      schedules: formSchedules,
    });
    setValue(BACKUP_SCHEDULES_FIELD, updatedSchedulesArray);
    setSelectedScheduleName('');
    setOpenScheduleModal(false);
  };

  const handleClose = () => {
    setOpenScheduleModal(false);
  };

  return (
    <>
      <ActionableLabeledContent
        label={Messages.label}
        actionButtonProps={{
          disabled: createButtonDisabled,
          dataTestId: 'create-schedule',
          buttonText: Messages.create,
          onClick: () => handleCreate(),
        }}
      >
        <Stack>
          {formSchedules.map((item: FlattenedSchedule) => (
            <EditableItem
              key={item.name}
              dataTestId={item.name}
              children={
                <ScheduleContent
                  schedule={item}
                  storageName={item.storageName}
                />
              }
              editButtonProps={{ onClick: () => handleEdit(item.name) }}
              deleteButtonProps={{ onClick: () => handleDelete(item.name) }}
            />
          ))}
          {formSchedules.length === 0 && (
            <EditableItem
              dataTestId="empty"
              children={
                <Typography variant="body1">{Messages.noSchedules}</Typography>
              }
            />
          )}
        </Stack>
      </ActionableLabeledContent>
      {openScheduleModal && (
        <ScheduleFormDialogContext.Provider
          value={{
            mode,
            handleSubmit,
            handleClose,
            isPending: false,
            setMode,
            selectedScheduleName,
            setSelectedScheduleName,
            openScheduleModal,
            setOpenScheduleModal,
            externalContext: dbWizardToScheduleFormDialogMap(dbWizardMode),
            dbInstanceInfo: {
              dbInstanceName: dbName,
              namespace: k8sNamespace,
              schedules: formSchedules,
              defaultSchedules: formSchedules,
              backupClass,
              availableBackupClasses: backupClasses,
              disableClassSelection: formSchedules.length > 0,
              instanceStorageNames: [],
            },
          }}
        >
          <ScheduleFormDialog />
        </ScheduleFormDialogContext.Provider>
      )}
    </>
  );
};
