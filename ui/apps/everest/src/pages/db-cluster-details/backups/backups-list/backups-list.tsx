// everest
// Copyright (C) 2023 Percona LLC
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
import { useQueryClient } from '@tanstack/react-query';
import StatusField from 'components/status-field';
import { DATE_FORMAT } from 'consts';
import { format } from 'date-fns';
import {
  BACKUPS_QUERY_KEY,
  useDbBackups,
  useDbClusterPitr,
  useDeleteBackup,
  useUpdateDbClusterWithConflictRetry,
} from 'hooks';
import { MRT_ColumnDef } from 'material-react-table';
import { Alert, Typography } from '@mui/material';
import { RestoreDbModal } from 'modals/index.ts';
import { useContext, useMemo, useState } from 'react';
import {
  Backup,
  BackupStatus,
  GetBackupsPayload,
} from 'shared-types/backups.types';
import { ScheduleModalContext } from '../backups.context.ts';
import { BACKUP_STATUS_TO_BASE_STATUS } from './backups-list.constants';
import { Messages } from './backups-list.messages';
import { Messages as DbDetailsMessages } from '../../db-cluster-details.messages';
import BackupListTableHeader from './table-header';
import { CustomConfirmDialog } from 'components/custom-confirm-dialog/custom-confirm-dialog.tsx';
import { DbEngineType } from '@percona/types';
import { getAvailableBackupStoragesForBackups } from 'utils/backups.ts';
import { dbEngineToDbType } from '@percona/utils';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages.ts';
import TableActionsMenu from 'components/table-actions-menu';
import { BackupActionButtons } from './backups-list-menu-actions';
import { shouldDbActionsBeBlocked } from 'utils/db.tsx';
import { WizardMode } from 'shared-types/wizard.types.ts';

export const BackupsList = () => {
  const queryClient = useQueryClient();
  const [openRestoreDbModal, setOpenRestoreDbModal] = useState(false);
  const [isNewClusterMode, setIsNewClusterMode] = useState(false);
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);
  const [selectedBackup, setSelectedBackup] = useState('');
  const {
    dbCluster,
    setMode: setScheduleModalMode,
    setOpenScheduleModal,
    setOpenOnDemandModal,
  } = useContext(ScheduleModalContext);

  const { mutate: deleteBackup, isPending: deletingBackup } = useDeleteBackup(
    dbCluster?.metadata.namespace
  );
  const { data: backups = [] } = useDbBackups(
    dbCluster.metadata.name,
    dbCluster.metadata.namespace,
    {
      refetchInterval: 10 * 1000,
    }
  );
  const { mutate: updateCluster } = useUpdateDbClusterWithConflictRetry(
    dbCluster,
    {
      onSuccess: () => handleCloseDeleteDialog(),
    }
  );
  const { data: pitrData } = useDbClusterPitr(
    dbCluster.metadata.name!,
    dbCluster.metadata.namespace,
    {
      enabled: !!dbCluster.metadata.name && !!dbCluster.metadata.namespace,
    }
  );
  const { data: backupStorages = [] } = useBackupStoragesByNamespace(
    dbCluster?.metadata.namespace
  );
  const dbType = dbCluster.spec?.engine.type;
  const pitrEnabled = !!dbCluster.spec?.backup?.pitr?.enabled;
  const willDisablePITR =
    (dbCluster.spec?.backup?.schedules || []).length === 0 &&
    pitrEnabled &&
    backups.length === 1;

  const { storagesToShow, uniqueStoragesInUse } =
    getAvailableBackupStoragesForBackups(
      backups,
      dbCluster.spec?.backup?.schedules || [],
      backupStorages,
      dbEngineToDbType(dbType),
      dbType === DbEngineType.POSTGRESQL
    );
  const noStoragesAvailable =
    dbType === DbEngineType.POSTGRESQL && storagesToShow.length === 0;

  const columns = useMemo<MRT_ColumnDef<Backup>[]>(
    () => [
      {
        accessorKey: 'state',
        header: 'Status',
        filterVariant: 'multi-select',
        filterSelectOptions: Object.values(BackupStatus),
        Cell: ({ cell }) => (
          <StatusField
            status={cell.getValue<BackupStatus>()}
            statusMap={BACKUP_STATUS_TO_BASE_STATUS}
          >
            {/* @ts-ignore */}
            {cell.getValue()}
          </StatusField>
        ),
      },
      {
        accessorKey: 'name',
        header: 'Name',
      },
      {
        accessorKey: 'backupStorageName',
        header: 'Storage',
      },
      {
        accessorKey: 'created',
        header: 'Started',
        enableColumnFilter: false,
        sortingFn: 'datetime',
        Cell: ({ cell }) =>
          cell.getValue<string>()
            ? format(cell.getValue<string>(), DATE_FORMAT)
            : '',
      },
      {
        accessorKey: 'completed',
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

  if (!dbCluster) {
    return null;
  }

  const handleDeleteBackup = (backupName: string) => {
    setSelectedBackup(backupName);
    setOpenDeleteDialog(true);
  };

  const handleManualBackup = () => {
    setOpenOnDemandModal(true);
  };

  const handleScheduledBackup = () => {
    setScheduleModalMode(WizardMode.New);
    setOpenScheduleModal(true);
  };

  const handleCloseDeleteDialog = () => {
    setOpenDeleteDialog(false);
  };

  const handleConfirmDelete = (
    backupName: string,
    cleanupBackupStorage: boolean
  ) => {
    const newBackupNr = backups.length - 1;
    deleteBackup(
      { backupName: backupName, cleanupBackupStorage: cleanupBackupStorage },
      {
        onSuccess: () => {
          if (willDisablePITR) {
            updateCluster({
              ...dbCluster,
              spec: {
                ...dbCluster.spec,
                backup: {
                  ...dbCluster.spec.backup,
                  pitr: {
                    enabled: false,
                    backupStorageName:
                      dbCluster.spec.backup?.pitr?.backupStorageName || '',
                  },
                },
              },
            });
          }
          queryClient.setQueryData(
            [
              BACKUPS_QUERY_KEY,
              dbCluster.metadata.namespace,
              dbCluster.metadata.name,
            ],
            (oldData: GetBackupsPayload) => ({
              items: oldData.items.map((backup) =>
                backup.metadata.name === backupName
                  ? {
                      ...backup,
                      status: {
                        ...backup.status,
                        state: BackupStatus.DELETING,
                      },
                    }
                  : backup
              ),
            })
          );

          if (
            dbCluster.spec.engine.type === DbEngineType.POSTGRESQL &&
            newBackupNr === 0 &&
            !dbCluster.spec.backup?.schedules?.length
          ) {
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
                    enabled: false,
                  },
                },
              },
            });
          }

          handleCloseDeleteDialog();
        },
      }
    );
  };

  const handleRestoreBackup = (backupName: string) => {
    setSelectedBackup(backupName);
    setIsNewClusterMode(false);
    setOpenRestoreDbModal(true);
  };

  const handleRestoreToNewDbBackup = (backupName: string) => {
    setSelectedBackup(backupName);
    setOpenRestoreDbModal(true);
    setIsNewClusterMode(true);
  };

  return (
    <>
      {pitrData?.gaps && (
        <Alert severity="error">{DbDetailsMessages.pitrError}</Alert>
      )}
      {dbType === DbEngineType.POSTGRESQL && (
        <Typography variant="body2" mt={2} px={1}>
          {Messages.pgMaximum(uniqueStoragesInUse.length)}
        </Typography>
      )}
      <Table
        getRowId={(row) => row.name}
        tableName="backupList"
        noDataMessage={Messages.noData}
        data={backups}
        columns={columns}
        initialState={{
          sorting: [
            {
              id: 'created',
              desc: true,
            },
          ],
        }}
        renderTopToolbarCustomActions={() => (
          <BackupListTableHeader
            onNowClick={handleManualBackup}
            onScheduleClick={handleScheduledBackup}
            noStoragesAvailable={noStoragesAvailable}
            currentBackups={backups}
          />
        )}
        enableRowActions
        renderRowActions={({ row }) => {
          const menuItems = BackupActionButtons(
            row,
            shouldDbActionsBeBlocked(dbCluster.status?.status),
            handleDeleteBackup,
            handleRestoreBackup,
            handleRestoreToNewDbBackup,
            dbCluster
          );
          return <TableActionsMenu menuItems={menuItems} />;
        }}
      />
      {openDeleteDialog && (
        <CustomConfirmDialog
          isOpen={openDeleteDialog}
          selectedId={selectedBackup}
          closeModal={handleCloseDeleteDialog}
          headerMessage={Messages.deleteDialog.header}
          handleConfirm={() =>
            handleConfirmDelete(
              selectedBackup,
              dbCluster.spec.engine.type !== DbEngineType.POSTGRESQL
            )
          }
          submitting={deletingBackup}
          confirmationInput={false}
          dialogContent={Messages.deleteDialog.content(
            selectedBackup,
            dbCluster.spec.engine.type,
            willDisablePITR
          )}
          alertMessage={Messages.deleteDialog.alertMessage}
          submitMessage={Messages.deleteDialog.confirmButton}
        />
      )}
      {openRestoreDbModal && dbCluster && (
        <RestoreDbModal
          dbCluster={dbCluster}
          namespace={dbCluster.metadata.namespace}
          isNewClusterMode={isNewClusterMode}
          isOpen={openRestoreDbModal}
          closeModal={() => setOpenRestoreDbModal(false)}
          backupName={selectedBackup}
        />
      )}
    </>
  );
};
