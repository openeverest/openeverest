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

import { Box, Card, CardContent, CardHeader, Stack } from '@mui/material';
import { PendingIcon, Table } from '@percona/ui-lib';
import StatusField from 'components/status-field';
import { type MRT_ColumnDef } from 'material-react-table';
import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { beautifyDbInstanceStatus } from './DbClusterView.utils';
import { InstanceTableElement } from './dbClusterView.types';
import { CreateDbButton } from 'components/create-db-button';
import EmptyStateDatabases from 'components/empty-state-databases/empty-state-databases';
import EmptyStateNamespaces from 'components/empty-state-namespaces/empty-state-namespaces';
import { DB_INSTANCE_STATUS_TO_BASE_STATUS } from './DbClusterView.constants';
import { DbActions } from 'components/db-actions/db-actions';
import {
  DbInstancePhaseValues,
  DbInstancePhase,
} from 'shared-types/instance.types';
import { usePlugins } from 'contexts/plugins';
import type { GlobalDashboardWidgetExtension } from '@openeverest/plugin-sdk';
import PluginErrorBoundary from 'components/plugin-host/PluginErrorBoundary';
import LoadingPageSkeleton from 'components/loading-page-skeleton/LoadingPageSkeleton';
import { useDatabasesView } from './hooks/useDatabasesView';

export const DbClusterView = () => {
  const navigate = useNavigate();
  const {
    state,
    namespaces,
    providersNamesFilter,
    hasCreatePermission,
    canAddCluster,
  } = useDatabasesView();

  const columns = useMemo<MRT_ColumnDef<InstanceTableElement>[]>(
    () => [
      {
        accessorKey: 'phase',
        header: 'Status',
        filterVariant: 'multi-select',
        filterSelectOptions: DbInstancePhaseValues.map((status) => ({
          text: beautifyDbInstanceStatus(status),
          value: status,
        })),
        maxSize: 120,
        Cell: ({ cell }) => {
          const status = cell.getValue<DbInstancePhase>();

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
            sx={{
              justifyContent: 'center',
              alignItems: 'center',
              gap: 1,
            }}
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

  // Collect plugin globalDashboardWidget extensions.
  const { plugins } = usePlugins();
  const dashboardWidgets = useMemo(
    () =>
      plugins.flatMap((p) =>
        p.extensions
          .filter(
            (ext): ext is GlobalDashboardWidgetExtension =>
              ext.type === 'globalDashboardWidget'
          )
          .map((ext) => ({ pluginName: p.name, ext }))
      ),
    [plugins]
  );

  if (state.status === 'loading') {
    return <LoadingPageSkeleton />;
  }

  return (
    <Stack
      direction="column"
      sx={{
        alignItems: 'center',
      }}
    >
      {dashboardWidgets.length > 0 && (
        <Box
          sx={{
            width: '100%',
            display: 'grid',
            gridTemplateColumns: {
              xs: '1fr',
              md: 'repeat(2, 1fr)',
              xl: 'repeat(3, 1fr)',
            },
            gap: 2,
            mb: 2,
          }}
        >
          {dashboardWidgets.map((w) => {
            const WidgetComponent = w.ext.component;
            return (
              <Card
                key={`widget-${w.pluginName}-${w.ext.label}`}
                variant="outlined"
              >
                <CardHeader
                  title={w.ext.label}
                  slotProps={{
                    title: { variant: 'subtitle1' },
                  }}
                />
                <CardContent>
                  <PluginErrorBoundary pluginName={w.pluginName}>
                    <WidgetComponent namespaces={namespaces} />
                  </PluginErrorBoundary>
                </CardContent>
              </Card>
            );
          })}
        </Box>
      )}
      <Box sx={{ width: '100%' }}>
        {state.status === 'no-namespaces' ? (
          <EmptyStateNamespaces />
        ) : state.status === 'empty' ? (
          <EmptyStateDatabases hasCreatePermission={hasCreatePermission} />
        ) : (
          <Table
            getRowId={(row) => row.instanceName}
            tableName="dbClusterView"
            state={{ isLoading: false }}
            columns={columns}
            data={state.rows}
            enableRowActions
            renderRowActions={({ row }) => {
              return (
                <DbActions dbInstance={row.original.raw} showDetailsAction />
              );
            }}
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
              canAddCluster && (
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
        )}
      </Box>
    </Stack>
  );
};
