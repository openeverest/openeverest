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

import { Table } from '@percona/ui-lib';
import StatusField from 'components/status-field';
import { DATE_FORMAT } from 'consts';
import { format } from 'date-fns';
import { useBackupsList } from 'hooks/api/backups/useBackups.ts';
import { MRT_ColumnDef } from 'material-react-table';
import { useContext, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { Backup, BackupStatus } from 'shared-types/backups.types.ts';
import { ScheduleModalContext } from '../backups.context.ts';
import { BACKUP_STATUS_TO_BASE_STATUS } from './backups-list.constants';
import { Messages } from './backups-list.messages';
import BackupListTableHeader from './table-header';
import { useClusterName } from 'hooks/api/useClusterName.ts';

export const BackupsList = () => {
  const { instanceName = '', namespace = '' } = useParams();
  const clusterName = useClusterName();

  const {
    instance,
    setOpenOnDemandModal,
  } = useContext(ScheduleModalContext);

  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    instanceName,
    {
      refetchInterval: 10 * 1000,
    }
  );

  const columns = useMemo<MRT_ColumnDef<Backup>[]>(
    () => [
      {
        accessorFn: (row) => row.status?.state ?? BackupStatus.UNKNOWN,
        id: 'state',
        header: 'Status',
        filterVariant: 'multi-select',
        Cell: ({ cell }) => (
          <StatusField
            status={cell.getValue<string>()}
            statusMap={BACKUP_STATUS_TO_BASE_STATUS}
          >
            {cell.getValue<string>()}
          </StatusField>
        ),
      },
      {
        accessorFn: (row) => row.metadata?.name ?? '',
        id: 'name',
        header: 'Name',
      },
      {
        accessorFn: (row) => row.spec?.storageName ?? '',
        id: 'storageName',
        header: 'Storage',
      },
      {
        accessorFn: (row) => row.spec?.backupClassName ?? '',
        id: 'backupClassName',
        header: 'Backup class',
      },
      {
        accessorFn: (row) => row.status?.startedAt ?? '',
        id: 'startedAt',
        header: 'Started',
        enableColumnFilter: false,
        sortingFn: 'datetime',
        Cell: ({ cell }) =>
          cell.getValue<string>()
            ? format(cell.getValue<string>(), DATE_FORMAT)
            : '',
      },
      {
        accessorFn: (row) => row.status?.completedAt ?? '',
        id: 'completedAt',
        header: 'Finished',
        enableColumnFilter: false,
        sortingFn: 'datetime',
        Cell: ({ cell }) =>
          cell.getValue<string>()
            ? format(cell.getValue<string>(), DATE_FORMAT)
            : '',
      },
    ],
    []
  );

  if (!instance) {
    return null;
  }

  const handleManualBackup = () => {
    setOpenOnDemandModal(true);
  };

  return (
    <Table
      getRowId={(row) => row.metadata?.name ?? ''}
      tableName="backupList"
      noDataMessage={Messages.noData}
      data={backups}
      columns={columns}
      initialState={{
        sorting: [
          {
            id: 'startedAt',
            desc: true,
          },
        ],
      }}
      renderTopToolbarCustomActions={() => (
        <BackupListTableHeader
          onNowClick={handleManualBackup}
          onScheduleClick={() => {}}
          noStoragesAvailable={false}
          currentBackups={backups}
        />
      )}
    />
  );
};
