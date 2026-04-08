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
  BACKUPS_QUERY_KEY,
  useCreateBackupOnDemand,
  useDbBackups,
} from 'hooks/api/backups/useBackups';
import { useContext, useMemo } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import {
  GetBackupsPayload,
  SingleBackupPayload,
} from 'shared-types/backups.types';
// import { Messages } from '../../db-cluster-details.messages.ts';
import { OnDemandBackupFieldsWrapper } from './on-demand-backup-fields-wrapper.tsx';
import {
  BackupFormData,
  defaultValuesFc,
  schema,
} from './on-demand-backup-modal.types.ts';
import { ScheduleModalContext } from '../backups.context.ts';
import { Typography } from '@mui/material';
import { DbCluster } from 'shared-types/dbCluster.types.ts';
import { DbEngineType } from 'shared-types/dbEngines.types.ts';
import { useUpdateDbClusterWithConflictRetry } from 'hooks';

export const OnDemandBackupModal = ({
  dbCluster,
}: {
  dbCluster: DbCluster;
}) => {
  const queryClient = useQueryClient();
  const { dbClusterName, namespace = '' } = useParams();

  const { data: backups = [] } = useDbBackups(dbClusterName!, namespace);
  const backupNames = backups.map((item) => item.name);
  const { mutate: createBackupOnDemand, isPending: creatingBackup } =
    useCreateBackupOnDemand(dbClusterName!, namespace);
  const { mutate: updateCluster } =
    useUpdateDbClusterWithConflictRetry(dbCluster);

  const { openOnDemandModal, setOpenOnDemandModal } =
    useContext(ScheduleModalContext);

  const handleSubmit = (data: BackupFormData) => {
    createBackupOnDemand(data, {
      onSuccess(newBackup: SingleBackupPayload) {
        queryClient.setQueryData<GetBackupsPayload | undefined>(
          [BACKUPS_QUERY_KEY, namespace, dbClusterName],
          (oldData) => {
            if (!oldData) {
              return undefined;
            }

            return {
              items: [newBackup, ...oldData.items],
            };
          }
        );

        if (dbCluster.spec.engine.type === DbEngineType.POSTGRESQL) {
          updateCluster({
            ...dbCluster,
            spec: {
              ...dbCluster.spec,
              backup: {
                ...dbCluster.spec.backup,
                pitr: {
                  ...(dbCluster.spec.backup?.pitr || {}),
                  backupStorageName:
                    dbCluster.spec.backup?.pitr?.backupStorageName || '',
                  enabled: true,
                },
              },
            },
          });
        }
        setOpenOnDemandModal(false);
      },
    });
  };

  const values = useMemo(() => defaultValuesFc(), []);

  return (
    <FormDialog
      isOpen={openOnDemandModal}
      closeModal={() => setOpenOnDemandModal(false)}
      headerMessage={/*Messages.onDemandBackupModal.headerMessage*/ ''}
      onSubmit={handleSubmit}
      submitting={creatingBackup}
      submitMessage={/*Messages.onDemandBackupModal.submitMessage*/ ''}
      schema={schema(backupNames)}
      values={values}
      size="XL"
    >
      <Typography variant="body1">
        {/*Messages.onDemandBackupModal.subHead*/}
      </Typography>
      <OnDemandBackupFieldsWrapper />
    </FormDialog>
  );
};
