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

import { FormGroup, Box, Skeleton } from '@mui/material';
import { keepPreviousData } from '@tanstack/react-query';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages';
import { useFormContext } from 'react-hook-form';
import { DbWizardFormFields } from 'consts.ts';
import BackupsActionableAlert from 'components/actionable-alert/backups-actionable-alert';
import { StepHeader } from '../../steps-old/step-header/step-header.tsx';
import { Messages } from './backup-step.messages.ts';
import { Schedules, BACKUP_CLASS_REF_FIELD } from './schedules.tsx';
import { PitrBlock } from './pitr';
import { StepProps } from '../../../database-form.types';

export const BackupStep = (
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _props: StepProps
) => {
  const { watch } = useFormContext();

  const selectedNamespace: string = watch(DbWizardFormFields.k8sNamespace);
  const backupClassName: string | undefined = watch(BACKUP_CLASS_REF_FIELD);

  const { data: backupStorages = [], isLoading } = useBackupStoragesByNamespace(
    selectedNamespace,
    backupClassName ? { backupClass: backupClassName } : undefined,
    { refetchInterval: 5 * 1000, placeholderData: keepPreviousData }
  );

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
        <FormGroup sx={{ mt: 3 }}>
          <Schedules backupStorages={backupStorages} />
          <PitrBlock />
        </FormGroup>
      ) : (
        <BackupsActionableAlert namespace={selectedNamespace} />
      )}
    </Box>
  );
};
