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

import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import { Box, Button, Typography } from '@mui/material';
import { useParams } from 'react-router-dom';
import {
  BACKUP_STORAGES_QUERY_KEY,
  useCreateBackupStorage,
} from 'hooks/api/backup-storages/useBackupStorages';
import { CreateEditModalStorage } from 'pages/settings/storage-locations/createEditModal/create-edit-modal';
import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { BackupStorageFormValues } from 'shared-types/backupStorages.types';
import { Messages } from '../backups.messages';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import { useClusterName } from 'hooks/api/useClusterName';

export const NoStoragesMessage = () => {
  const queryClient = useQueryClient();
  const clusterName = useClusterName();
  const { namespace = '' } = useParams();
  const [openCreateEditModal, setOpenCreateEditModal] = useState(false);
  const { mutate: createBackupStorage, isPending: creatingBackupStorage } =
    useCreateBackupStorage();

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

  const handleCloseModal = () => {
    setOpenCreateEditModal(false);
  };

  const { canCreate } = useNamespacePermissionsForResource('backup-storages');
  return (
    <Box
      sx={{
        display: 'flex',
        py: 6,
        px: 0,
        flexDirection: 'column',
        alignItems: 'center',
        gap: 1,
        alignSelf: 'stretch',
      }}
    >
      <Box sx={{ fontSize: '100px', lineHeight: 0 }}>
        <WarningAmberIcon fontSize="inherit" />
      </Box>
      <Typography variant="body1">{Messages.noStoragesMessage}</Typography>
      <Button
        sx={{ my: 4 }}
        variant="contained"
        onClick={() => setOpenCreateEditModal(true)}
        disabled={canCreate.length <= 0}
      >
        {Messages.addStorage}
      </Button>
      {openCreateEditModal && (
        <CreateEditModalStorage
          open={openCreateEditModal}
          handleCloseModal={handleCloseModal}
          handleSubmitModal={handleSubmit}
          isLoading={creatingBackupStorage}
          prefillNamespace={namespace}
        />
      )}
    </Box>
  );
};
