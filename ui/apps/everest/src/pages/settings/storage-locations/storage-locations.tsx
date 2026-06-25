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

import { Add } from '@mui/icons-material';
import { Box, Button } from '@mui/material';
import { Table } from '@percona/ui-lib';
import { useQueryClient } from '@tanstack/react-query';
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import {
  getBackupStoragesQueryKey,
  useBackupStorages,
  useCreateBackupStorage,
  useDeleteBackupStorage,
  useEditBackupStorage,
} from 'hooks/api/backup-storages/useBackupStorages';
import { type MRT_ColumnDef } from 'material-react-table';
import { ExpandedRowInfoLine } from './ExpandedRowInfoLine/ExpandedRowInfoLine';
import { useMemo, useState } from 'react';
import {
  BackupStorageCRD,
  BackupStorageFormValues,
  StorageType,
} from 'shared-types/backupStorages.types';
import { CreateEditModalStorage } from './createEditModal/create-edit-modal';
import { Messages } from './storage-locations.messages';
import {
  StorageLocationsFields,
  BackupStorageTableElement,
} from './storage-locations.types';
import {
  convertBackupStoragesPayloadToTableFormat,
  convertStoragesType,
} from './storage-locations.utils';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import { useClusterName } from 'hooks/api/useClusterName';
import TableActionsMenu from '../../../components/table-actions-menu';
import { StorageLocationsActionButtons } from './storage-locations-menu-actions';
import {
  optimisticCreateBy,
  optimisticDeleteBy,
  optimisticEditBy,
} from 'utils/generalOptimisticDataUpdate';

export const StorageLocations = () => {
  const queryClient = useQueryClient();
  const clusterName = useClusterName();
  const { canCreate } = useNamespacePermissionsForResource('backup-storages');
  const backupStorages = useBackupStorages();

  const backupStoragesLoading = backupStorages.some(
    (result) => result.isLoading
  );

  const tableData = useMemo(
    () => convertBackupStoragesPayloadToTableFormat(backupStorages),
    [backupStorages]
  );

  const { mutate: createBackupStorage, isPending: creatingBackupStorage } =
    useCreateBackupStorage();
  const { mutate: editBackupStorage, isPending: editingBackupStorage } =
    useEditBackupStorage();
  const { mutate: deleteBackupStorage, isPending: deletingBackupStorage } =
    useDeleteBackupStorage();

  const [openCreateEditModal, setOpenCreateEditModal] = useState(false);
  const [selectedStorageName, setSelectedStorageName] = useState<string>('');
  const [selectedStorageNamespace, setSelectedStorageNamespace] =
    useState<string>('');
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);
  const [selectedStorageLocation, setSelectedStorageLocation] =
    useState<BackupStorageCRD>();

  const columns = useMemo<MRT_ColumnDef<BackupStorageTableElement>[]>(
    () => [
      {
        accessorKey: StorageLocationsFields.name,
        header: Messages.name,
      },
      {
        accessorKey: StorageLocationsFields.type,
        header: Messages.type,
        Cell: ({ cell }) => convertStoragesType(cell.getValue<StorageType>()),
      },
      {
        accessorKey: StorageLocationsFields.bucketName,
        header: Messages.bucketName,
      },
      {
        accessorKey: StorageLocationsFields.namespace,
        header: Messages.namespace,
      },
      {
        accessorKey: StorageLocationsFields.url,
        header: Messages.url,
        enableHiding: false,
      },
    ],
    []
  );

  const handleOpenCreateModal = () => {
    setSelectedStorageLocation(undefined);
    setOpenCreateEditModal(true);
  };

  const handleOpenEditModal = (storageLocation: BackupStorageTableElement) => {
    setSelectedStorageLocation(storageLocation.raw);
    setOpenCreateEditModal(true);
  };

  const handleCloseModal = () => {
    setSelectedStorageLocation(undefined);
    setOpenCreateEditModal(false);
  };

  const handleEditBackupStorage = (data: BackupStorageFormValues) => {
    const queryKey = getBackupStoragesQueryKey(clusterName, data.namespace);
    editBackupStorage(data, {
      onSuccess: (updatedLocation) => {
        const updatedStorage = updatedLocation as BackupStorageCRD;
        optimisticEditBy<BackupStorageCRD>(
          queryClient,
          queryKey,
          updatedStorage,
          (item) =>
            item.metadata?.name === updatedStorage.metadata?.name &&
            item.metadata?.namespace === updatedStorage.metadata?.namespace
        );
        handleCloseModal();
      },
    });
  };

  const handleCreateBackup = (data: BackupStorageFormValues) => {
    const queryKey = getBackupStoragesQueryKey(clusterName, data.namespace);
    createBackupStorage(data, {
      onSuccess: (newLocation) => {
        const createdStorage = newLocation as BackupStorageCRD;
        optimisticCreateBy<BackupStorageCRD>(
          queryClient,
          queryKey,
          createdStorage,
          (item) =>
            item.metadata?.name === createdStorage.metadata?.name &&
            item.metadata?.namespace === createdStorage.metadata?.namespace
        );
        handleCloseModal();
      },
    });
  };

  const handleSubmit = (isEdit: boolean, data: BackupStorageFormValues) => {
    if (isEdit) {
      handleEditBackupStorage(data);
    } else {
      handleCreateBackup(data);
    }
  };

  const handleDeleteBackup = (backupStorageName: string, namespace: string) => {
    setSelectedStorageName(backupStorageName);
    setSelectedStorageNamespace(namespace);
    setOpenDeleteDialog(true);
  };

  const handleCloseDeleteDialog = () => {
    setOpenDeleteDialog(false);
  };

  const handleConfirmDelete = (
    backupStorageName: string,
    namespace: string
  ) => {
    const queryKey = getBackupStoragesQueryKey(clusterName, namespace);
    deleteBackupStorage(
      {
        backupStorageId: backupStorageName,
        namespace,
      },
      {
        onSuccess: () => {
          optimisticDeleteBy<BackupStorageCRD>(
            queryClient,
            queryKey,
            (item) =>
              item.metadata?.name === backupStorageName &&
              item.metadata?.namespace === namespace
          );
          handleCloseDeleteDialog();
        },
      }
    );
  };

  return (
    <>
      <Table
        getRowId={(row) => row.name}
        tableName="storageLocations"
        noDataMessage={Messages.noData}
        hideExpandAllIcon
        state={{
          columnVisibility: {
            url: false,
            accessKey: false,
            secretKey: false,
          },
          isLoading: backupStoragesLoading,
        }}
        columns={columns}
        data={tableData}
        renderTopToolbarCustomActions={() =>
          canCreate.length > 0 && (
            <Button
              size="small"
              startIcon={<Add />}
              data-testid="add-backup-storage"
              variant="outlined"
              onClick={handleOpenCreateModal}
              sx={{ display: 'flex' }}
            >
              {Messages.addStorageLocationButton}
            </Button>
          )
        }
        enableRowActions
        renderRowActions={({ row }) => {
          const menuItems = StorageLocationsActionButtons(
            row,
            handleOpenEditModal,
            handleDeleteBackup
          );
          return <TableActionsMenu menuItems={menuItems} />;
        }}
        renderDetailPanel={({ row }) => (
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'start',
              alignItems: 'start',
              gap: '50px',
            }}
          >
            <Box>
              <ExpandedRowInfoLine
                label={Messages.url}
                value={row.original.url}
              />
            </Box>
            {/* TODO: uncomment when endpoint is ready
            <Box>
              <ExpandedRowInfoLine label="Access key" value={row.original.accessKey} />
              <ExpandedRowInfoLine label="Secret key" value={row.original.secretKey} />
            </Box>  */}
          </Box>
        )}
      />
      {openCreateEditModal && (
        <CreateEditModalStorage
          open={openCreateEditModal}
          handleCloseModal={handleCloseModal}
          handleSubmitModal={handleSubmit}
          selectedStorageLocation={selectedStorageLocation}
          isLoading={creatingBackupStorage || editingBackupStorage}
        />
      )}
      {openDeleteDialog && (
        <ConfirmDialog
          open={openDeleteDialog}
          cancelMessage="Cancel"
          selectedId={selectedStorageName}
          selectedNamespace={selectedStorageNamespace}
          closeModal={handleCloseDeleteDialog}
          headerMessage={Messages.deleteDialog.header}
          handleConfirmNamespace={handleConfirmDelete}
          disabledButtons={deletingBackupStorage}
        >
          {Messages.deleteDialog.content(selectedStorageName)}
        </ConfirmDialog>
      )}
    </>
  );
};
