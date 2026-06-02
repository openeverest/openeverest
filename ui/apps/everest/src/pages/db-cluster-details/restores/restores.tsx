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
// TODO: Re-enable PITR alert when PITR is restored.
// import { Alert } from '@mui/material';
import { MRT_ColumnDef } from 'material-react-table';
import { format } from 'date-fns';
import { Table } from '@percona/ui-lib';
import { DATE_FORMAT } from 'consts';
import StatusField from 'components/status-field/status-field';
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
// TODO: Re-enable PITR and import jobs when data-import-jobs is migrated to v2 API.
// import { useDbClusterPitr } from 'hooks/api/backups/useBackups';
import {
  PG_STATUS,
  PSMDB_STATUS,
  PXC_STATUS,
  Restore,
} from 'shared-types/restores.types';
import { Messages } from './restores.messages';
// TODO: Re-enable PITR alert when PITR is restored.
// import { Messages as DbDetailsMessages } from '../db-cluster-details.messages';
import {
  getRestoresListQueryKey,
  useInstanceRestores,
  useDeleteRestore,
} from 'hooks/api/restores/useDbClusterRestore';
import { useMemo, useState } from 'react';
import { RESTORE_STATUS_TO_BASE_STATUS } from './restores.constants';
import { useQueryClient } from '@tanstack/react-query';
import TableActionsMenu from 'components/table-actions-menu';
import { RestoreActionButtons } from './restores-menu-actions';
// TODO: Re-enable when data-import-jobs endpoint is migrated to v2 API.
// import { useDbClusterImportJobs } from 'hooks';
// import {
//   DataImportJob,
//   DataImportJobs,
// } from 'shared-types/dataImporters.types';
import { useClusterName } from 'hooks/api/useClusterName';

// TODO: Re-enable when data-import-jobs endpoint is migrated to v2 API.
// const getImportJobsData = (imports?: DataImportJobs): Restore[] => {
//   if (!imports?.items || !imports.items.length) return [];
//   return imports.items.map((importItem: DataImportJob) => ({
//     backupSource: importItem.spec.dataImporterName,
//     endTime: importItem.status?.completedAt || '',
//     name: importItem.metadata.name,
//     startTime: importItem.status?.startedAt || '',
//     state: importItem.status?.state || '',
//     type: 'import',
//   }));
// };

function getTypeCellValue(type: string) {
  if (type === 'import') return 'Import';
  if (type === 'pitr') return 'PITR';
  return 'Full';
}

const Restores = () => {
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);
  const [selectedRestore, setSelectedRestore] = useState('');
  const { instanceName = '', namespace = '' } = useParams();
  // TODO: Re-enable when v1 dbClusterName param is restored.
  // const { dbClusterName, namespace = '' } = useParams();
  const clusterName = useClusterName();
  const queryClient = useQueryClient();
  // TODO: Re-enable PITR data when PITR is restored.
  // const { data: pitrData } = useDbClusterPitr(dbClusterName!, namespace, {
  //   enabled: !!dbClusterName && !!namespace,
  // });
  const { data: restores = [], isLoading: loadingRestores } =
    useInstanceRestores(clusterName, namespace, instanceName, {
      enabled: !!instanceName && !!namespace,
    });
  // TODO: Re-enable v1 restores hook when PITR/import logic is restored.
  // const { data: restores = [], isLoading: loadingRestores } =
  //   useDbClusterRestores(namespace, dbClusterName!, {
  //     enabled: !!dbClusterName && !!namespace,
  //   });

  // TODO: Re-enable when data-import-jobs endpoint is migrated to v2 API.
  // const { data: imports } = useDbClusterImportJobs(namespace, instanceName);

  // const tableData = [...restores, ...getImportJobsData(imports)];
  const tableData = restores;

  const { mutate: deleteRestore, isPending: deletingRestore } =
    useDeleteRestore(clusterName, namespace);
  // TODO: Re-enable v1 delete restore when PITR/import logic is restored.
  // const { mutate: deleteRestore, isPending: deletingRestore } =
  //   useDeleteRestore(namespace);

  const columns = useMemo<MRT_ColumnDef<Restore>[]>(() => {
    return [
      {
        header: 'Status',
        accessorKey: 'state',
        Cell: ({ cell }) => (
          <StatusField
            status={cell.getValue<PXC_STATUS | PSMDB_STATUS | PG_STATUS>()}
            statusMap={RESTORE_STATUS_TO_BASE_STATUS}
          >
            {capitalize(cell.getValue<PXC_STATUS | PSMDB_STATUS | PG_STATUS>())}
          </StatusField>
        ),
      },
      {
        header: 'Name',
        accessorKey: 'name',
      },
      {
        header: 'Started',
        accessorKey: 'startTime',
        Cell: ({ cell }) =>
          cell.getValue<Date>()
            ? format(new Date(cell.getValue<Date>()), DATE_FORMAT)
            : '-----',
      },
      {
        header: 'Finished',
        accessorKey: 'endTime',
        Cell: ({ cell }) =>
          cell.getValue<Date>()
            ? format(new Date(cell.getValue<Date>()), DATE_FORMAT)
            : '-----',
      },
      {
        header: 'Type',
        accessorKey: 'type',
        Cell: ({ cell }) => getTypeCellValue(cell.getValue<string>() || ''),
      },
      {
        header: 'Backup Source',
        accessorKey: 'backupSource',
        Cell: ({ cell }) =>
          cell.row.original.type === 'import'
            ? 'External'
            : cell.getValue<string>(),
      },
    ];
  }, []);

  const handleDeleteBackup = (restoreName: string) => {
    setSelectedRestore(restoreName);
    setOpenDeleteDialog(true);
  };

  const handleConfirmDelete = (restoreName: string) => {
    deleteRestore(restoreName, {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: getRestoresListQueryKey(
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
      {/* TODO: Re-enable PITR alert when PITR is restored.
      {pitrData?.gaps && (
        <Alert severity="error">{DbDetailsMessages.pitrError}</Alert>
      )}
      */}
      <Table
        getRowId={(row) => row.name}
        state={{ isLoading: loadingRestores }}
        tableName={`${instanceName}-restore`}
        // TODO: Re-enable v1 dbClusterName when PITR/import logic is restored.
        // tableName={`${dbClusterName}-restore`}
        columns={columns}
        data={tableData}
        initialState={{
          sorting: [
            {
              id: 'startTime',
              desc: false,
            },
            { id: 'endTime', desc: false },
          ],
        }}
        noDataMessage="No restores"
        enableRowActions
        renderRowActions={({ row }) => {
          const menuItems = RestoreActionButtons(
            row,
            handleDeleteBackup,
            namespace,
            instanceName
          );
          // TODO: Re-enable 'import' type check when data-import-jobs is migrated to v2 API.
          // return row.original.type !== 'import' ? (
          //   <TableActionsMenu menuItems={menuItems} />
          // ) : null;
          return <TableActionsMenu menuItems={menuItems} />;
        }}
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
