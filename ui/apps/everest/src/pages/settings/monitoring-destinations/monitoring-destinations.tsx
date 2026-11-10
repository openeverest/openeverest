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
import TableActionsMenu from 'components/table-actions-menu';
import { DeleteDestinationDialog } from './delete-destination-dialog/delete-destination-dialog';
import { DestinationStatusCell } from './destination-status-cell/destination-status-cell';
import { useDestinationDeletion } from './hooks/use-destination-deletion';
import { useMonitoringDestinationPermissions } from './hooks/use-monitoring-destination-permissions';
import { useMonitoringDestinationRows } from './hooks/use-monitoring-destination-rows';
import { getMonitoringDestinationMenuItems } from './monitoring-destination-menu-actions';
import { MonitoringDestinationModal } from './monitoring-destination-modal/monitoring-destination-modal';
import { MonitoringDestinationTarget } from './monitoring-destination-modal/monitoring-destination-modal.types';
import { Messages } from './monitoring-destinations.messages';
import { MonitoringDestinationRow } from './monitoring-destinations.types';
import { UsedByCell } from './used-by-cell/used-by-cell';

type ModalState = { target?: MonitoringDestinationTarget } | undefined;

export const MonitoringDestinations = () => {
  const { rows, classes, isLoading } = useMonitoringDestinationRows();
  const permissions = useMonitoringDestinationPermissions();
  const { deletion, isDeleting, request, confirm, cancel } =
    useDestinationDeletion(rows);
  const [modal, setModal] = useState<ModalState>();
  const hasAddress = rows.some((row) => !!row.address);

  const columns = useMemo<MRT_ColumnDef<MonitoringDestinationRow>[]>(
    () => [
      { accessorKey: 'name', header: Messages.columns.name },
      { accessorKey: 'namespace', header: Messages.columns.namespace },
      { accessorKey: 'type', header: Messages.columns.type },
      // Only classes whose destination form has a URL field have an address
      // (e.g. PMM); Prometheus and Coroot destinations have none.
      ...(hasAddress
        ? [
            {
              accessorKey: 'address',
              header: Messages.columns.address,
              Cell: ({
                row,
              }: {
                row: { original: MonitoringDestinationRow };
              }) => row.original.address ?? Messages.none,
            } satisfies MRT_ColumnDef<MonitoringDestinationRow>,
          ]
        : []),
      {
        id: 'status',
        header: Messages.columns.status,
        accessorFn: (row) => row.status.state,
        Cell: ({ row }) => (
          <DestinationStatusCell status={row.original.status} />
        ),
      },
      {
        id: 'usedBy',
        header: Messages.columns.usedBy,
        accessorFn: (row) => row.usedBy.length,
        Cell: ({ row }) => (
          <UsedByCell
            destination={row.original.name}
            instances={row.original.usedBy}
          />
        ),
      },
    ],
    [hasAddress]
  );

  return (
    <>
      <Table
        getRowId={(row) => `${row.namespace}/${row.name}`}
        tableName="monitoringDestinations"
        hideExpandAllIcon
        data={rows}
        columns={columns}
        state={{ isLoading }}
        noDataMessage={Messages.noData}
        enableRowActions
        renderTopToolbarCustomActions={() =>
          permissions.createNamespaces.length > 0 && (
            <Button
              size="small"
              startIcon={<Add />}
              data-testid="add-monitoring-destination"
              variant="outlined"
              onClick={() => setModal({})}
            >
              {Messages.add}
            </Button>
          )
        }
        renderRowActions={({ row }) => (
          <TableActionsMenu
            menuItems={getMonitoringDestinationMenuItems({
              row: row.original,
              permissions,
              onEdit: ({ namespace, name }) =>
                setModal({ target: { namespace, name } }),
              onDelete: request,
            })}
          />
        )}
      />
      {modal && (
        <MonitoringDestinationModal
          target={modal.target}
          classes={classes}
          namespaces={permissions.createNamespaces}
          onClose={() => setModal(undefined)}
        />
      )}
      {deletion && (
        <DeleteDestinationDialog
          deletion={deletion}
          isDeleting={isDeleting}
          onConfirm={confirm}
          onClose={cancel}
        />
      )}
    </>
  );
};
