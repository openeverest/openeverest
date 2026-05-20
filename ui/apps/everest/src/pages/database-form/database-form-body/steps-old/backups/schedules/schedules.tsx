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

// @ts-nocheck
// TODO remove this file after release of v2

import { Stack, Typography } from '@mui/material';
import EditableItem from 'components/editable-item/editable-item';
import { Messages } from './schedules.messages';
import { useEffect, useState } from 'react';
import { DbWizardFormFields } from 'consts.ts';
import { useFormContext } from 'react-hook-form';
import { ManageableSchedules } from 'shared-types/dbCluster.types';
import {
  getSchedulesPayload,
  removeScheduleFromArray,
} from 'components/schedule-form-dialog/schedule-form/schedule-form.utils';
import ScheduleContent from './schedule-body';
import { ScheduleFormDialog } from 'components/schedule-form-dialog';
import { ScheduleFormDialogContext } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog.context';
import { ScheduleFormData } from 'components/schedule-form-dialog/schedule-form/schedule-form-schema';
import { dbTypeToDbEngine } from '@percona/utils';
import { DbType } from '@percona/types';
import { ActionableLabeledContent } from '@percona/ui-lib';
import { useDatabasePageMode } from '../../../../hooks/use-database-page-mode';
import { dbWizardToScheduleFormDialogMap } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';
import { useDatabasePageDefaultValues } from '../../../../hooks/use-database-form-default-values';
import { PG_SLOTS_LIMIT } from 'consts';
import { useRBACPermissions } from 'hooks/rbac';
import { transformSchedulesIntoManageableSchedules } from 'utils/db';
import { ScheduleWizardMode, WizardMode } from 'shared-types/wizard.types';
import { BackupStorage } from 'shared-types/backupStorages.types';

type Props = {
  storagesToShow: BackupStorage[];
  disableCreateButton?: boolean;
};

const Schedules = ({ storagesToShow, disableCreateButton = false }: Props) => {
  const { watch, setValue } = useFormContext();
  const dbWizardMode = useDatabasePageMode();
  const {
    defaultValues: { schedules: defaultDbSchedules },
  } = useDatabasePageDefaultValues(dbWizardMode);
  const [openScheduleModal, setOpenScheduleModal] = useState(false);
  const [mode, setMode] = useState<ScheduleWizardMode>(WizardMode.New);
  const [schedules, setSchedules] = useState<ManageableSchedules[]>([]);
  const [selectedScheduleName, setSelectedScheduleName] = useState<string>('');

  const [dbType, k8sNamespace, formSchedules, dbName] = watch([
    DbWizardFormFields.dbType,
    DbWizardFormFields.k8sNamespace,
    DbWizardFormFields.schedules,
    DbWizardFormFields.dbName,
  ]);

  const { canCreate: canCreateBackups, canRead: canReadBackups } =
    useRBACPermissions('database-cluster-backups', `${k8sNamespace}/${dbName}`);
  const { canUpdate: canUpdateCluster } = useRBACPermissions(
    'database-clusters',
    `${k8sNamespace}/${dbName}`
  );

  const [activeStorage, setActiveStorage] = useState<string | undefined>(
    undefined
  );
  const createButtonDisabled =
    disableCreateButton ||
    openScheduleModal ||
    (dbType === DbType.Postresql && schedules?.length >= PG_SLOTS_LIMIT);

  useEffect(() => {
    if (schedules?.length > 0 && dbType === DbType.Mongo) {
      setActiveStorage(schedules[0]?.backupStorageName);
    } else {
      setActiveStorage(undefined);
    }
  }, [schedules, dbType]);

  useEffect(() => {
    const baseSchedules: ManageableSchedules[] = canReadBackups
      ? formSchedules
      : [];

    transformSchedulesIntoManageableSchedules(
      baseSchedules,
      k8sNamespace,
      canCreateBackups,
      mode === WizardMode.New ? true : canUpdateCluster
    ).then((newSchedules) => {
      setSchedules(newSchedules);
    });
  }, [
    canCreateBackups,
    canReadBackups,
    canUpdateCluster,
    formSchedules,
    k8sNamespace,
    mode,
  ]);

  const handleDelete = (name: string) => {
    setValue(
      DbWizardFormFields.schedules,
      removeScheduleFromArray(name, schedules)
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
      schedules,
    });
    setValue(DbWizardFormFields.schedules, updatedSchedulesArray);
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
        actionButtonProps={
          storagesToShow.length && canCreateBackups
            ? {
                disabled: createButtonDisabled,
                dataTestId: 'create-schedule',
                buttonText: 'Create backup schedule',
                onClick: () => handleCreate(),
              }
            : undefined
        }
      >
        <Stack>
          {dbType === DbType.Mongo && (
            <Typography variant="caption">{Messages.mongoDb}</Typography>
          )}
          {dbType === DbType.Postresql && (
            <Typography variant="caption">{Messages.pg}</Typography>
          )}
          {schedules.map((item: ManageableSchedules) => (
            <EditableItem
              key={item.name}
              dataTestId={item.name}
              children={
                <ScheduleContent
                  schedule={item}
                  storageName={item.backupStorageName}
                />
              }
              editButtonProps={
                item.canBeManaged
                  ? {
                      onClick: () => handleEdit(item.name),
                    }
                  : undefined
              }
              deleteButtonProps={
                item.canBeManaged
                  ? {
                      onClick: () => handleDelete(item.name),
                    }
                  : undefined
              }
            />
          ))}
          {schedules.length === 0 && (
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
              schedules,
              defaultSchedules: defaultDbSchedules,
              namespace: k8sNamespace,
              dbEngine: dbTypeToDbEngine(dbType),
              activeStorage,
              dbInstanceName: dbName,
            },
          }}
        >
          <ScheduleFormDialog />
        </ScheduleFormDialogContext.Provider>
      )}
    </>
  );
};

export default Schedules;
