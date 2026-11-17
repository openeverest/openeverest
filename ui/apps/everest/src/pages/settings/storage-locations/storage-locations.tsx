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

import { useMemo, useState } from 'react';
import { Add } from '@mui/icons-material';
import { Button } from '@mui/material';
import { Table } from '@percona/ui-lib';
import { type MRT_ColumnDef } from 'material-react-table';
import {
  BackupStorageModal,
  BackupStorageTarget,
  useBackupStoragePermissions,
} from 'components/backup-storage-modal';
import { useDeleteBackupStorage } from 'hooks/api/backup-storages';
import { DeleteStorageDialog } from './delete-storage-dialog/delete-storage-dialog';
import { useStorageLocationRows } from './hooks/use-storage-location-rows';
import { StorageDetails } from './storage-details/storage-details';
import { StorageRowActions } from './storage-row-actions/storage-row-actions';
import { StorageStatusCell } from './storage-status-cell/storage-status-cell';
import { Messages } from './storage-locations.messages';
import { BackupStorageRow } from './storage-locations.types';

type ModalState = { target?: BackupStorageTarget } | undefined;

export const StorageLocations = () => {
  const { rows, isLoading } = useStorageLocationRows();
  const { createNamespaces } = useBackupStoragePermissions();
  const { mutate: deleteStorage, isPending: isDeleting } =
    useDeleteBackupStorage();
  const [deletion, setDeletion] = useState<BackupStorageRow>();
  const [modal, setModal] = useState<ModalState>();

  const columns = useMemo<MRT_ColumnDef<BackupStorageRow>[]>(
    () => [
      { accessorKey: 'name', header: Messages.columns.name },
      { accessorKey: 'namespace', header: Messages.columns.namespace },
      { accessorKey: 'type', header: Messages.columns.type },
      { accessorKey: 'location', header: Messages.columns.location },
      {
        id: 'status',
        header: Messages.columns.status,
        accessorFn: (row) => row.status.state,
        Cell: ({ row }) => <StorageStatusCell status={row.original.status} />,
      },
    ],
    []
  );

  return (
    <>
      <Table
        getRowId={(row) => `${row.namespace}/${row.name}`}
        tableName="storageLocations"
        hideExpandAllIcon
        noDataMessage={Messages.noData}
        data={rows}
        columns={columns}
        state={{ isLoading }}
        enableRowActions
        renderTopToolbarCustomActions={() =>
          createNamespaces.length > 0 && (
            <Button
              size="small"
              startIcon={<Add />}
              data-testid="add-backup-storage"
              variant="outlined"
              onClick={() => setModal({})}
            >
              {Messages.add}
            </Button>
          )
        }
        renderRowActions={({ row }) => (
          <StorageRowActions
            row={row.original}
            onEdit={({ namespace, name }) =>
              setModal({ target: { namespace, name } })
            }
            onDelete={setDeletion}
          />
        )}
        renderDetailPanel={({ row }) => (
          <StorageDetails storage={row.original.raw} />
        )}
      />
      {modal && (
        <BackupStorageModal
          target={modal.target}
          onClose={() => setModal(undefined)}
        />
      )}
      {deletion && (
        <DeleteStorageDialog
          row={deletion}
          isDeleting={isDeleting}
          onClose={() => setDeletion(undefined)}
          onConfirm={() =>
            deleteStorage(
              { namespace: deletion.namespace, name: deletion.name },
              { onSuccess: () => setDeletion(undefined) }
            )
          }
        />
      )}
    </>
  );
};
