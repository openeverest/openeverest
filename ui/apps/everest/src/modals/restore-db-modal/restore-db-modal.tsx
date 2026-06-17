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

import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { FormDialog } from 'components/form-dialog';
import { useBackupsList } from 'hooks/api/backups/useBackups';
import {
  getRestoresListQueryKey,
  useCreateRestoreFromBackup,
} from 'hooks/api/restores/useDbClusterRestore';
import { useClusterName } from 'hooks/api/useClusterName';
import { BackupStatus } from 'shared-types/backups.types';
import {
  BackupTypeValues,
  defaultValues,
  RestoreDbFormData,
  schema,
} from './restore-db-modal-schema';
import { Messages } from './restore-db-modal.messages';
import { ModalContent } from './modal-content';
import {
  RestorableBackupOption,
  RestoreDbModalProps,
} from './restore-db-modal.types';
import {
  getBackupName,
  getMetadataCreationTimestamp,
  getSafeTimeValue,
} from './restore-db-modal.utils';

const RestoreDbModal = ({
  isOpen,
  closeModal,
  instanceName,
  namespace,
  isNewClusterMode = false,
}: RestoreDbModalProps) => {
  const clusterName = useClusterName();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const { data: backups = [], isLoading } = useBackupsList(
    clusterName,
    namespace,
    instanceName,
    { enabled: !!instanceName }
  );

  // TODO: Re-enable PITR hooks when PITR restore flow is implemented.
  // const { data: pitrData } = useDbClusterPitr(instanceName, namespace, {
  //   queryKey: [instanceName, namespace, 'pitr', 'restore-modal'],
  //   gcTime: 0,
  // });

  const succeededBackups = useMemo<RestorableBackupOption[]>(() => {
    return backups
      .filter((backup) => backup.status?.state === BackupStatus.SUCCEEDED)
      .map((backup) => ({
        name: getBackupName(backup),
        startedAt:
          backup.status?.startedAt ?? getMetadataCreationTimestamp(backup),
      }))
      .filter((backup) => !!backup.name)
      .sort(
        (a, b) => getSafeTimeValue(b.startedAt) - getSafeTimeValue(a.startedAt)
      );
  }, [backups]);

  const { mutate: createRestore, isPending: restoringFromBackup } =
    useCreateRestoreFromBackup(clusterName, namespace);

  // TODO: Re-enable PITR mutation when PITR restore flow is implemented.
  // const {
  //   mutate: restoreBackupFromPointInTime,
  //   isPending: restoringFromPointInTime,
  // } = useDbClusterRestoreFromPointInTime(instanceName);

  // TODO: Re-enable PITR schema when PITR restore flow is implemented.
  // const pitrSchema = useMemo(
  //   () => schema(!!pitrData?.gaps, pitrData?.earliestDate, pitrData?.latestDate),
  //   [pitrData]
  // );

  const handleSubmit = ({ backupName, backupType }: RestoreDbFormData) => {
    if (backupType === BackupTypeValues.fromBackup) {
      if (!backupName) {
        return;
      }

      if (isNewClusterMode) {
        closeModal();
        navigate('/databases/new', {
          state: {
            selectedDbCluster: instanceName,
            backupName,
            namespace,
          },
        });
        return;
      }

      createRestore(
        { instanceName, backupName },
        {
          onSuccess: () => {
            queryClient.invalidateQueries({
              queryKey: getRestoresListQueryKey(
                clusterName,
                namespace,
                instanceName
              ),
            });
            closeModal();
          },
        }
      );
      return;
    }

    // TODO: Re-enable PITR submit branch when PITR restore flow is implemented.
    // restoreBackupFromPointInTime({
    //   backupName: pitrBackupName,
    //   namespace,
    //   pointInTimeDate,
    // });
  };

  return (
    <FormDialog
      size="XXXL"
      isOpen={isOpen}
      dataTestId="restore-modal"
      closeModal={closeModal}
      headerMessage={
        isNewClusterMode ? Messages.headerMessageCreate : Messages.headerMessage
      }
      // TODO: Re-enable PITR schema when PITR restore flow is implemented.
      // schema={pitrSchema}
      schema={schema()}
      // TODO: Re-enable PITR pending state when PITR restore flow is implemented.
      // submitting={restoringFromBackup || restoringFromPointInTime}
      submitting={restoringFromBackup}
      defaultValues={defaultValues}
      onSubmit={handleSubmit}
      submitMessage={isNewClusterMode ? Messages.create : Messages.restore}
    >
      <ModalContent
        isLoading={isLoading}
        header={isNewClusterMode ? Messages.subHeadCreate : Messages.subHead}
        succeededBackups={succeededBackups}
      />
    </FormDialog>
  );
};

export default RestoreDbModal;
