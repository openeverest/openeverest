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
import { Button, Chip } from '@mui/material';
import { Table } from '@percona/ui-lib';
import { useQueryClient } from '@tanstack/react-query';
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import {
  MONITORING_CONFIGS_QUERY_KEY,
  useCreateMonitoringConfig,
  useDeleteMonitoringConfig,
  useMonitoringConfigsListMultiNs,
  useUpdateMonitoringConfig,
} from 'hooks/api/monitoring/useMonitoringConfigsList';
import { MRT_ColumnDef } from 'material-react-table';
import { useMemo, useState } from 'react';
import { MonitoringConfig } from 'shared-types/api.types';
import { CreateEditEndpointModal } from './createEditModal/create-edit-modal';
import { EndpointFormType } from './createEditModal/create-edit-modal.types';
import { Messages } from './monitoring-endpoints.messages';
import { convertMonitoringConfigsToTableFormat } from './monitoring-endpoints.utils';
import { MonitoringConfigTableElement } from './monitoring-endpoints.types';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import TableActionsMenu from '../../../components/table-actions-menu';
import { MonitoringActionButtons } from './monitoring-endpoint-menu-actions';

export const MonitoringEndpoints = () => {
  const [openCreateEditModal, setOpenCreateEditModal] = useState(false);
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);
  const [selectedConfig, setSelectedConfig] = useState<{
    config: MonitoringConfig;
    namespace: string;
  }>();
  const { configs: monitoringConfigs, isNamespacesLoading } =
    useMonitoringConfigsListMultiNs();

  const tableData = useMemo(
    () => convertMonitoringConfigsToTableFormat(monitoringConfigs),
    [monitoringConfigs]
  );

  const monitoringConfigsLoading =
    isNamespacesLoading ||
    (monitoringConfigs.length > 0 &&
      !monitoringConfigs.some((result) => result.queryResult.isSuccess));

  const { mutate: createMonitoringConfig, isPending: creatingConfig } =
    useCreateMonitoringConfig();
  const { mutate: deleteMonitoringConfig, isPending: removingConfig } =
    useDeleteMonitoringConfig();
  const { mutate: updateMonitoringConfig, isPending: updatingConfig } =
    useUpdateMonitoringConfig();
  const queryClient = useQueryClient();
  const columns = useMemo<MRT_ColumnDef<MonitoringConfigTableElement>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
      },
      {
        accessorKey: 'url',
        header: 'Endpoint',
      },
      {
        accessorKey: 'namespace',
        header: Messages.namespaces,
      },
      // {
      //   accessorKey: 'pmmServerVersion',
      //   header: 'PMM Version',
      // },
      {
        accessorKey: 'inUse',
        header: 'In Use',
        Cell: ({ cell }) => (
          <Chip
            label={cell.getValue<boolean>() ? 'Yes' : 'No'}
            color={cell.getValue<boolean>() ? 'success' : 'default'}
            size="small"
            variant="outlined"
          />
        ),
      },
    ],
    []
  );

  const handleOpenCreateModal = () => {
    setOpenCreateEditModal(true);
  };

  const handleOpenEditModal = (element: MonitoringConfigTableElement) => {
    setSelectedConfig({ config: element.raw, namespace: element.namespace });
    setOpenCreateEditModal(true);
  };

  const handleCloseModal = () => {
    setSelectedConfig(undefined);
    setOpenCreateEditModal(false);
  };

  const handleCloseDeleteDialog = () => {
    setSelectedConfig(undefined);
    setOpenDeleteDialog(false);
  };

  const handleDeleteConfig = (element: MonitoringConfigTableElement) => {
    setSelectedConfig({ config: element.raw, namespace: element.namespace });
    setOpenDeleteDialog(true);
  };

  const handleSubmitModal = (isEditMode: boolean, data: EndpointFormType) => {
    const { name, url, namespace, verifyTLS, user, password } = data;
    const pmm: Record<string, string> = {};
    if (user) pmm.user = user;
    if (password) pmm.password = password;

    if (isEditMode) {
      updateMonitoringConfig(
        {
          namespace,
          name,
          payload: {
            url,
            type: 'pmm',
            verifyTLS,
            pmm,
          },
        },
        {
          onSuccess: () => {
            queryClient.invalidateQueries({
              queryKey: [MONITORING_CONFIGS_QUERY_KEY],
            });
            handleCloseModal();
          },
        }
      );
    } else {
      createMonitoringConfig(
        {
          namespace,
          payload: {
            name,
            url,
            type: 'pmm',
            verifyTLS,
            pmm,
          },
        },
        {
          onSuccess: () => {
            queryClient.invalidateQueries({
              queryKey: [MONITORING_CONFIGS_QUERY_KEY],
            });
            handleCloseModal();
          },
        }
      );
    }
  };

  const handleConfirmDelete = (configName: string, namespace: string) => {
    deleteMonitoringConfig(
      { name: configName, namespace },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({
            queryKey: [MONITORING_CONFIGS_QUERY_KEY],
          });
          handleCloseDeleteDialog();
        },
      }
    );
  };

  const { canCreate } =
    useNamespacePermissionsForResource('monitoring-configs');

  return (
    <>
      <Table
        getRowId={(row) => `${row.namespace}/${row.name}`}
        tableName="monitoringEndpoints"
        hideExpandAllIcon
        data={tableData}
        columns={columns}
        state={{
          isLoading: monitoringConfigsLoading,
        }}
        enableRowActions
        noDataMessage="No monitoring endpoint added"
        renderTopToolbarCustomActions={() =>
          canCreate.length > 0 && (
            <Button
              size="small"
              startIcon={<Add />}
              data-testid="add-monitoring-endpoint"
              variant="outlined"
              onClick={handleOpenCreateModal}
              sx={{ display: 'flex' }}
            >
              {Messages.add}
            </Button>
          )
        }
        renderRowActions={({ row }) => {
          const menuItems = MonitoringActionButtons(
            row,
            handleDeleteConfig,
            handleOpenEditModal
          );
          return <TableActionsMenu menuItems={menuItems} />;
        }}
      />
      {openCreateEditModal && (
        <CreateEditEndpointModal
          open={openCreateEditModal}
          handleClose={handleCloseModal}
          handleSubmit={handleSubmitModal}
          isLoading={creatingConfig || updatingConfig}
          selectedEndpoint={selectedConfig?.config}
          selectedNamespace={selectedConfig?.namespace}
        />
      )}
      {openDeleteDialog && (
        <ConfirmDialog
          open={openDeleteDialog}
          cancelMessage="Cancel"
          selectedId={selectedConfig?.config.metadata?.name || ''}
          selectedNamespace={selectedConfig?.namespace || ''}
          closeModal={handleCloseDeleteDialog}
          headerMessage={Messages.deleteDialogHeader}
          handleConfirmNamespace={handleConfirmDelete}
          disabledButtons={removingConfig}
        >
          {Messages.deleteConfirmation(
            selectedConfig?.config.metadata?.name || ''
          )}
        </ConfirmDialog>
      )}
    </>
  );
};
