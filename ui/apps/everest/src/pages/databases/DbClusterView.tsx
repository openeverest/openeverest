// everest
// Copyright (C) 2023 Percona LLC
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

import { Box, Stack } from '@mui/material';
import { PendingIcon, Table } from '@percona/ui-lib';
import StatusField from 'components/status-field';
import { useNamespaces } from 'hooks/api/namespaces/useNamespaces';
import { useProviders } from 'hooks/api/providers/useProviders';
import { type MRT_ColumnDef } from 'material-react-table';
import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useInstancesForNamespaces } from 'hooks/api/db-instances/useDbInstanceList';
import {
  beautifyDbInstanceStatus,
  convertDbInstancesPayloadToTableFormat,
} from './DbClusterView.utils';
import { InstanceTableElement } from './dbClusterView.types';
import CreateDbButton from 'components/create-db-button/create-db-button';
import EmptyStateDatabases from 'components/empty-state-databases/empty-state-databases';
import EmptyStateNamespaces from 'components/empty-state-namespaces/empty-state-namespaces';
import { DB_INSTANCE_STATUS_TO_BASE_STATUS } from './DbClusterView.constants';
import { DbInstanceStatus } from 'shared-types/instance.types';

export const DbClusterView = () => {
  const { data: namespaces = [], isLoading: loadingNamespaces } = useNamespaces(
    {
      refetchInterval: 10 * 1000,
    }
  );

  const navigate = useNavigate();
  const { data: providers = [] } = useProviders();
  const hasAvailableProviders = providers.length > 0;
  const providersNamesFilter = providers.reduce<
    { text: string; value: string }[]
  >((acc, p) => {
    const name = p.metadata?.name;
    if (name) acc.push({ text: name, value: name });
    return acc;
  }, []);

  // TODO RBAC
  // const { canCreate } = useNamespacePermissionsForResource('database-instances');

  // const canAddCluster = canCreate.length > 0 && hasAvailableProviders;
  const canAddCluster = hasAvailableProviders;

  // TODO uncomment when providerImporters will be ready
  // const { data: availableEnginesForImport } = useDataImporters();

  const instancesResults = useInstancesForNamespaces(
    namespaces.map((ns) => ({
      namespace: ns,
    }))
  );
  const instancesLoading = instancesResults.some(
    (result) => result.queryResult.isLoading
  );

  const tableData = useMemo(
    () => convertDbInstancesPayloadToTableFormat(instancesResults),
    [instancesResults]
  );

  const columns = useMemo<MRT_ColumnDef<InstanceTableElement>[]>(
    () => [
      {
        accessorKey: 'phase',
        header: 'Status',
        filterVariant: 'multi-select',
        filterSelectOptions: Object.values(DbInstanceStatus).map((status) => ({
          text: beautifyDbInstanceStatus(status),
          value: status,
        })),
        maxSize: 120,
        Cell: ({ cell }) => {
          const status = cell.getValue<DbInstanceStatus>();

          return (
            <StatusField
              dataTestId={cell.row.original?.instanceName}
              status={status}
              statusMap={DB_INSTANCE_STATUS_TO_BASE_STATUS}
              defaultIcon={PendingIcon}
            >
              {beautifyDbInstanceStatus(
                status /*
              cell.row.original?.raw.status?.conditions || []
                 */
              )}
            </StatusField>
          );
        },
      },
      {
        accessorKey: 'instanceName',
        header: 'Database name',
      },
      {
        accessorFn: ({ provider }) => provider,
        filterVariant: 'multi-select',
        filterSelectOptions: providersNamesFilter,
        header: 'Provider',
        id: 'provider',
        Cell: ({ row }) => (
          <Stack
            direction="row"
            justifyContent="center"
            alignItems="center"
            gap={1}
          >
            {row.original?.provider} {/* {row.original?.dbVersion} */}
          </Stack>
        ),
      },
      // {
      //   id: 'lastBackup',
      //   header: 'Last backup',
      //   Cell: ({ row }) => (
      //     <LastBackup
      //       dbName={row.original?.databaseName}
      //       namespace={row.original?.namespace}
      //     />
      //   ),
      // },
      // {
      //   accessorKey: 'nodes',
      //   id: 'nodes',
      //   header: 'Nº nodes',
      // },
      {
        accessorKey: 'namespace',
        id: 'namespace',
        header: 'Namespace',
      },
      // {
      //   accessorKey: 'monitoringConfigName',
      //   header: 'Monitoring instance name',
      //   minSize: 250,
      // },
      // {
      //   accessorKey: 'backupsEnabled',
      //   header: 'Backups',
      //   filterVariant: 'checkbox',
      //   accessorFn: (row) => (row.backupsEnabled ? 'true' : 'false'),
      //   Cell: ({ cell }) =>
      //     cell.getValue() === 'true' ? 'Enabled' : 'Disabled',
      // },
      // {
      //   accessorKey: 'kubernetesCluster',
      //   header: 'Kubernetes Cluster',
      // },
    ],
    []
  );
  return (
    <Stack direction="column" alignItems="center">
      <Box sx={{ width: '100%' }}>
        <Table
          getRowId={(row) => row.instanceName}
          tableName="dbClusterView"
          emptyState={
            namespaces.length > 0 ? (
              <EmptyStateDatabases
                showCreationButton={canAddCluster}
                hasCreatePermission={canAddCluster}
              />
            ) : (
              <EmptyStateNamespaces />
            )
          }
          state={{ isLoading: instancesLoading || loadingNamespaces }}
          columns={columns}
          data={tableData}
          // enableRowActions
          // renderRowActions={({ row }) => {
          //   return <DbActions dbCluster={row.original.raw} showDetailsAction />;
          // }}
          muiTableBodyRowProps={({ row, isDetailPanel }) => ({
            onClick: (e) => {
              if (
                !isDetailPanel &&
                e.currentTarget.contains(e.target as Node)
              ) {
                navigate(
                  `/databases/${row.original.namespace}/${row.original.instanceName}/overview`
                );
              }
            },
            sx: {
              ...(!isDetailPanel && {
                cursor: 'pointer', // you might want to change the cursor too when adding an onClick
              }),
            },
          })}
          enableRowHoverAction
          rowHoverAction={(row) =>
            navigate(
              `/databases/${row.original.namespace}/${row.original.instanceName}/overview`
            )
          }
          renderTopToolbarCustomActions={() =>
            canAddCluster &&
            tableData.length > 0 && (
              <Box display="flex" mb={1}>
                {/*TODO uncomment when providerImporters will be ready */}
                {/* {(availableEnginesForImport?.items || []).length > 0 && (
                  <CreateDbButton createFromImport />
                )} */}
                <CreateDbButton />
              </Box>
            )
          }
          hideExpandAllIcon
        />
      </Box>
    </Stack>
  );
};
