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

import { useState } from 'react';
import ActionableAlert from '..';
import { Messages } from './BackupsActionableAlert.messages';
import { BackupsActionableAlertProps } from './BackupsActionableAlert.types';
import { CreateEditModalStorage } from 'pages/settings/storage-locations/createEditModal/create-edit-modal';
import {
  BACKUP_STORAGES_QUERY_KEY,
  useCreateBackupStorage,
} from 'hooks/api/backup-storages/useBackupStorages';
import { useQueryClient } from '@tanstack/react-query';
import { BackupStorageFormValues } from 'shared-types/backupStorages.types';
import { useRBACPermissions } from 'hooks/rbac';
import { useClusterName } from 'hooks/api/useClusterName';

const BackupsActionableAlert = ({ namespace }: BackupsActionableAlertProps) => {
  const [openCreateEditModal, setOpenCreateEditModal] = useState(false);
  const { mutate: createBackupStorage, isPending: creatingBackupStorage } =
    useCreateBackupStorage();
  const queryClient = useQueryClient();
  const clusterName = useClusterName();
  const { canCreate } = useRBACPermissions('backup-storages', `${namespace}/*`);
  const handleCloseModal = () => {
    setOpenCreateEditModal(false);
  };

  const handleSubmit = (_: boolean, data: BackupStorageFormValues) => {
    createBackupStorage(data, {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: [BACKUP_STORAGES_QUERY_KEY, clusterName, data.namespace],
        });
        handleCloseModal();
      },
    });
  };

  return (
    <>
      <ActionableAlert
        message={Messages.noStoragesMessage}
        buttonMessage={Messages.addStorage}
        data-testid="no-storage-message"
        onClick={() => setOpenCreateEditModal(true)}
        {...(!canCreate && { action: undefined })}
      />
      {openCreateEditModal && (
        <CreateEditModalStorage
          open={openCreateEditModal}
          handleCloseModal={handleCloseModal}
          handleSubmitModal={handleSubmit}
          isLoading={creatingBackupStorage}
          prefillNamespace={namespace}
        />
      )}
    </>
  );
};

export default BackupsActionableAlert;
