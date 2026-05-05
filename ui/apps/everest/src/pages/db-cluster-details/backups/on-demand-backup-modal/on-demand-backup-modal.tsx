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
import { useContext, useMemo } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { CreateBackupPayload } from 'shared-types/backups.types.ts';
import { OnDemandBackupFieldsWrapper } from './on-demand-backup-fields-wrapper.tsx';
import {
  BackupFormData,
  defaultValuesFc,
  schema,
} from './on-demand-backup-modal.types.ts';
import { ScheduleModalContext } from '../backups.context.ts';
import { Typography } from '@mui/material';
import { useClusterName } from 'hooks/api/useClusterName.ts';

export const OnDemandBackupModal = () => {
  const queryClient = useQueryClient();
  const { instanceName = '', namespace = '' } = useParams();
  const clusterName = useClusterName();

  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    instanceName
  );
  const backupNames = backups.map(
    (item) => item.metadata?.name ?? ''
  );
  const { mutate: createBackupOnDemand, isPending: creatingBackup } =
    useCreateBackupOnDemand(clusterName, namespace);

  const { openOnDemandModal, setOpenOnDemandModal } =
    useContext(ScheduleModalContext);

  const handleSubmit = (data: BackupFormData) => {
    createBackupOnDemand(
      {
        metadata: { name: data.name },
        spec: {
          instanceName: instanceName,
          backupClassName: data.backupClassName,
          storageName: data.storageName,
        },
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

  const values = useMemo(() => defaultValuesFc(), []);

  return (
    <FormDialog
      isOpen={openOnDemandModal}
      closeModal={() => setOpenOnDemandModal(false)}
      headerMessage="Create on-demand backup"
      onSubmit={handleSubmit}
      submitting={creatingBackup}
      submitMessage="Create"
      schema={schema(backupNames)}
      values={values}
      size="XL"
    >
      <Typography variant="body1">
        Select a backup class and storage to create an on-demand backup.
      </Typography>
      <OnDemandBackupFieldsWrapper />
    </FormDialog>
  );
};
