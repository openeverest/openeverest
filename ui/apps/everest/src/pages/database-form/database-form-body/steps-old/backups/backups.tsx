// everest
// Copyright (C) 2023 Percona LLC
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

import { FormGroup, Box, Skeleton } from '@mui/material';
import { DbType } from '@percona/types';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages';
import { useDbBackups } from 'hooks/api/backups/useBackups.ts';
import { useFormContext } from 'react-hook-form';
import { getAvailableBackupStoragesForBackups } from 'utils/backups.ts';
import { DbWizardFormFields, PG_SLOTS_LIMIT } from 'consts.ts';
import BackupsActionableAlert from 'components/actionable-alert/backups-actionable-alert';
import { StepHeader } from '../step-header/step-header.tsx';
import { Messages } from './backups.messages.ts';
import Schedules from './schedules/index.ts';
import PITR from './pitr/index.ts';

export const Backups = () => {
  const { watch } = useFormContext();

  const [selectedNamespace, schedules, dbType, dbName] = watch([
    DbWizardFormFields.k8sNamespace,
    DbWizardFormFields.schedules,
    DbWizardFormFields.dbType,
    DbWizardFormFields.dbName,
  ]);
  const { data: backupStorages = [], isLoading } =
    useBackupStoragesByNamespace(selectedNamespace);
  const { data: backups = [] } = useDbBackups(dbName, selectedNamespace, {
    enabled: dbType === DbType.Postresql,
  });
  const { storagesToShow, uniqueStoragesInUse } =
    getAvailableBackupStoragesForBackups(
      backups,
      schedules,
      backupStorages,
      dbType,
      dbType === DbType.Postresql
    );
  const scheduleCreationDisabled =
    dbType === DbType.Postresql && storagesToShow.length === 0;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column' }}>
      <StepHeader
        pageTitle={Messages.backups}
        pageDescription={Messages.captionBackups}
      />
      {isLoading ? (
        <>
          <Skeleton height="200px" />
          <Skeleton />
          <Skeleton />
        </>
      ) : backupStorages.length > 0 ? (
        <>
          {scheduleCreationDisabled &&
            uniqueStoragesInUse.length < PG_SLOTS_LIMIT && (
              <BackupsActionableAlert namespace={selectedNamespace} />
            )}
          <FormGroup sx={{ mt: 3 }}>
            <Schedules
              storagesToShow={storagesToShow}
              disableCreateButton={scheduleCreationDisabled}
            />
            <PITR />
          </FormGroup>
        </>
      ) : (
        <BackupsActionableAlert namespace={selectedNamespace} />
      )}
    </Box>
  );
};
