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

import { useParams } from 'react-router-dom';
import { capitalize } from '@mui/material';
import { MRT_ColumnDef } from 'material-react-table';
import { format } from 'date-fns';
import { Table } from '@percona/ui-lib';
import { DATE_FORMAT } from 'consts';
import StatusField from 'components/status-field/status-field';
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import { Restore, PXC_STATUS, PSMDB_STATUS, PG_STATUS } from 'shared-types/restores.types';
import { Messages } from './restores.messages';
import {
  getRestoreListQueryKey,
  useDbClusterRestores,
  useDeleteRestore,
} from 'hooks/api/restores/useDbClusterRestore';
import { useMemo, useState } from 'react';
import { RESTORE_STATUS_TO_BASE_STATUS } from './restores.constants';
import { useQueryClient } from '@tanstack/react-query';
import TableActionsMenu from 'components/table-actions-menu';
import { RestoreActionButtons } from './restores-menu-actions';
import { useClusterName } from 'hooks/api/useClusterName';

function getTypeCellValue(restore: Restore) {
  if (restore.spec.dataSource.pitr) return 'PITR';
  return 'Full';
}

const Restores = () => {
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);
  const [selectedRestore, setSelectedRestore] = useState('');
  const { instanceName = '', namespace = '' } = useParams();
  const clusterName = useClusterName();
  const queryClient = useQueryClient();

  const { data: restores = [], isLoading: loadingRestores } =
    useDbClusterRestores(clusterName, namespace, instanceName, {
      enabled: !!instanceName && !!namespace,
    });

  const { mutate: deleteRestoreMutate, isPending: deletingRestore } =
    useDeleteRestore(clusterName, namespace, instanceName);

  const columns = useMemo<MRT_ColumnDef<Restore>[]>(() => {
    return [
      {
        header: 'Status',
        accessorFn: (row) => row.status?.state ?? 'unknown',
        id: 'state',
        Cell: ({ cell }) => {
          const status = cell.getValue<string>() as PXC_STATUS | PSMDB_STATUS | PG_STATUS;
          return (
            <StatusField
              status={status}
              statusMap={RESTORE_STATUS_TO_BASE_STATUS}
            >
              {capitalize(status)}
            </StatusField>
          );
        },
      },
      {
        header: 'Name',
        accessorFn: (row) => (row.metadata as { name?: string })?.name ?? '',
        id: 'name',
      },
      {
        header: 'Started',
        accessorFn: (row) => row.status?.startedAt ?? '',
        id: 'startedAt',
        Cell: ({ cell }) =>
          cell.getValue<string>()
            ? format(new Date(cell.getValue<string>()), DATE_FORMAT)
            : '-----',
      },
      {
        header: 'Finished',
        accessorFn: (row) => row.status?.completedAt ?? '',
        id: 'completedAt',
        Cell: ({ cell }) =>
          cell.getValue<string>()
            ? format(new Date(cell.getValue<string>()), DATE_FORMAT)
            : '-----',
      },
      {
        header: 'Type',
        id: 'type',
        accessorFn: (row) => getTypeCellValue(row),
      },
      {
        header: 'Backup Source',
        accessorFn: (row) => row.spec.dataSource.backupName ?? '',
        id: 'backupSource',
      },
    ];
  }, []);

  const handleDeleteRestore = (restoreName: string) => {
    setSelectedRestore(restoreName);
    setOpenDeleteDialog(true);
  };

  const handleConfirmDelete = (restoreName: string) => {
    deleteRestoreMutate(restoreName, {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: getRestoreListQueryKey(
            clusterName,
            namespace,
            instanceName
          ),
        });
        setOpenDeleteDialog(false);
      },
    });
  };

  return (
    <>
      <Table
        getRowId={(row) =>
          (row.metadata as { name?: string })?.name ?? ''
        }
        state={{ isLoading: loadingRestores }}
        tableName={`${instanceName}-restore`}
        columns={columns}
        data={restores}
        initialState={{
          sorting: [
            {
              id: 'startedAt',
              desc: true,
            },
          ],
        }}
        noDataMessage="No restores"
        enableRowActions
        renderRowActions={({ row }) => (
          <TableActionsMenu
            menuItems={RestoreActionButtons(
              row,
              handleDeleteRestore,
              namespace,
              instanceName
            )}
          />
        )}
      />
      {openDeleteDialog && (
        <ConfirmDialog
          open={openDeleteDialog}
          selectedId={selectedRestore}
          cancelMessage="Cancel"
          closeModal={() => setOpenDeleteDialog(false)}
          headerMessage={Messages.deleteDialog.header}
          handleConfirm={handleConfirmDelete}
          disabledButtons={deletingRestore}
        >
          {Messages.deleteDialog.content(selectedRestore)}
        </ConfirmDialog>
      )}
    </>
  );
};

export default Restores;
