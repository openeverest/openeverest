// TODO: check main — review this file against main for any lost functionality
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
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import { RestoreDbModal } from 'modals';
import TableActionsMenu from 'components/table-actions-menu';
import { DATE_FORMAT } from 'consts';
import { format } from 'date-fns';
import {
  getBackupListQueryKey,
  useBackupsList,
  useDeleteBackup,
} from 'hooks/api/backups/useBackups.ts';
import { MRT_ColumnDef } from 'material-react-table';
import { useContext, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  Backup,
  BackupList,
  BackupStatus,
} from 'shared-types/backups.types.ts';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { ScheduleModalContext } from '../backups.context.ts';
import { BACKUP_STATUS_TO_BASE_STATUS } from './backups-list.constants';
import { Messages } from './backups-list.messages';
import BackupListTableHeader from './table-header';
import { getBackupActionButtons } from './backups-list-menu-actions';
import { useClusterName } from 'hooks/api/useClusterName.ts';
import { useQueryClient } from '@tanstack/react-query';
import { useUpdateDbInstanceWithConflictRetry } from 'hooks/api/db-instances/useUpdateDbInstance';
import { Instance } from 'shared-types/api.types';
import { removeUnusedStorages } from '../backups.utils';
import { useRBACPermissions } from 'hooks/rbac';

export const BackupsList = () => {
  const { instanceName = '', namespace = '' } = useParams();
  const clusterName = useClusterName();
  const queryClient = useQueryClient();
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);
  const [selectedBackup, setSelectedBackup] = useState('');
  const [openRestoreModal, setOpenRestoreModal] = useState(false);
  const [isNewClusterMode, setIsNewClusterMode] = useState(false);
  const [selectedRestoreBackup, setSelectedRestoreBackup] = useState('');

  const { instance, setOpenOnDemandModal, setOpenScheduleModal, setMode } =
    useContext(ScheduleModalContext);

  const { canDelete } = useRBACPermissions(
    'backups',
    `${namespace}/${instanceName}`
  );

  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    instanceName,
    {
      refetchInterval: 10 * 1000,
    }
  );

  const { mutate: deleteBackupMutate, isPending: deletingBackup } =
    useDeleteBackup(clusterName, namespace, instanceName);

  const { mutate: updateInstance } =
    useUpdateDbInstanceWithConflictRetry(instance);

  const handleDeleteBackup = (backupName: string) => {
    setSelectedBackup(backupName);
    setOpenDeleteDialog(true);
  };

  const handleRestoreBackup = (backupName: string) => {
    setSelectedRestoreBackup(backupName);
    setIsNewClusterMode(false);
    setOpenRestoreModal(true);
  };

  const handleRestoreToNewDbBackup = (backupName: string) => {
    setSelectedRestoreBackup(backupName);
    setIsNewClusterMode(true);
    setOpenRestoreModal(true);
  };

  const handleConfirmDelete = (backupName: string) => {
    setOpenDeleteDialog(false);
    deleteBackupMutate(
      { backupName },
      {
        onSuccess: () => {
          // Optimistically mark the backup as Deleting in the cache so the
          // actions menu disables immediately, before the next poll cycle.
          queryClient.setQueryData<BackupList>(
            getBackupListQueryKey(clusterName, namespace, instanceName),
            (prev) =>
              prev
                ? {
                    ...prev,
                    items: prev.items?.map((b) =>
                      b.metadata?.name === backupName
                        ? {
                            ...b,
                            status: {
                              ...b.status,
                              state: BackupStatus.DELETING,
                            },
                          }
                        : b
                    ),
                  }
                : prev
          );
          queryClient.invalidateQueries({
            queryKey: getBackupListQueryKey(
              clusterName,
              namespace,
              instanceName
            ),
          });

          // Clean up instance storages that are no longer referenced.
          const remainingBackups = backups.filter(
            (b) => b.metadata?.name !== backupName
          );
          const existingStorages = instance.spec?.backup?.storages ?? [];
          const remainingStorages = removeUnusedStorages(
            existingStorages,
            remainingBackups
          );
          if (remainingStorages.length < existingStorages.length) {
            const updatedInstance: Instance = {
              ...instance,
              spec: {
                ...instance.spec,
                backup:
                  remainingStorages.length > 0
                    ? {
                        classRef: instance.spec?.backup?.classRef ?? {
                          name: '',
                        },
                        enabled: instance.spec?.backup?.enabled ?? true,
                        storages: remainingStorages,
                      }
                    : undefined,
              },
            };
            updateInstance(updatedInstance);
          }
        },
      }
    );
  };

  const columns = useMemo<MRT_ColumnDef<Backup>[]>(
    () => [
      {
        accessorFn: (row) => row.status?.state ?? '',
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
        accessorFn: (row) => row.status?.size ?? '',
        id: 'size',
        header: 'Size',
        enableColumnFilter: false,
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

  const handleScheduleBackup = () => {
    setMode(FormMode.New);
    setOpenScheduleModal(true);
  };

  return (
    <>
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
            onScheduleClick={handleScheduleBackup}
          />
        )}
        enableRowActions
        renderRowActions={({ row }) => (
          <TableActionsMenu
            menuItems={getBackupActionButtons(
              row,
              handleDeleteBackup,
              handleRestoreBackup,
              handleRestoreToNewDbBackup,
              canDelete,
              deletingBackup &&
                selectedBackup === (row.original.metadata?.name ?? '')
            )}
          />
        )}
      />
      {openDeleteDialog && (
        <ConfirmDialog
          open={openDeleteDialog}
          selectedId={selectedBackup}
          cancelMessage="Cancel"
          closeModal={() => setOpenDeleteDialog(false)}
          headerMessage={Messages.deleteDialog.header}
          handleConfirm={handleConfirmDelete}
          disabledButtons={deletingBackup}
        >
          {Messages.deleteDialog.content(selectedBackup)}
        </ConfirmDialog>
      )}
      {openRestoreModal && (
        <RestoreDbModal
          isOpen={openRestoreModal}
          closeModal={() => setOpenRestoreModal(false)}
          instanceName={instanceName}
          namespace={namespace}
          isNewClusterMode={isNewClusterMode}
          preselectedBackupName={selectedRestoreBackup}
        />
      )}
    </>
  );
};
