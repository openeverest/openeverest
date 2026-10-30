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
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import TableActionsMenu from 'components/table-actions-menu';
import {
  useFetchMonitoringBindings,
  useMonitoringBindingsForNamespaces,
} from 'hooks/api/monitoring-bindings';
import { useMonitoringClasses } from 'hooks/api/monitoring-classes';
import {
  useDeleteMonitoringDestination,
  useMonitoringDestinationsForNamespaces,
} from 'hooks/api/monitoring-destinations';
import {
  useNamespacePermissionsForResource,
  useRBACPermissions,
} from 'hooks/rbac';
import { MonitoringClass } from 'shared-types/monitoring-destinations.types';
import { getSecretDeletedWithDestination } from 'utils/monitoring-destinations';
import { DestinationStatusCell } from './destination-status-cell/destination-status-cell';
import { UsedByCell } from './used-by-cell/used-by-cell';
import { getMonitoringDestinationMenuItems } from './monitoring-destination-menu-actions';
import { MonitoringDestinationModal } from './monitoring-destination-modal/monitoring-destination-modal';
import { MonitoringDestinationTarget } from './monitoring-destination-modal/monitoring-destination-modal.types';
import { IN_USE_NAMES_SHOWN } from './monitoring-destinations.constants';
import { Messages } from './monitoring-destinations.messages';
import {
  MonitoringDestinationPermissions,
  MonitoringDestinationRow,
} from './monitoring-destinations.types';
import {
  toMonitoringDestinationRows,
  toWritableNamespaces,
} from './monitoring-destinations.utils';

const NO_CLASSES: MonitoringClass[] = [];

type ModalState = { target?: MonitoringDestinationTarget } | undefined;

interface DeleteState {
  row: MonitoringDestinationRow;
  usedBy: string[];
}

export const MonitoringDestinations = () => {
  const destinationPermissions = useNamespacePermissionsForResource(
    'monitoring-destinations'
  );
  const secretPermissions = useNamespacePermissionsForResource('secrets');
  const bindingPermissions = useNamespacePermissionsForResource(
    'monitoring-bindings'
  );
  const { canRead: canReadClasses } = useRBACPermissions('monitoring-classes');
  const { data: classes = NO_CLASSES } = useMonitoringClasses();

  const readNamespaces = destinationPermissions.canRead;
  const destinationResults =
    useMonitoringDestinationsForNamespaces(readNamespaces);
  const bindingResults = useMonitoringBindingsForNamespaces(
    bindingPermissions.canRead
  );

  const rows = toMonitoringDestinationRows(
    readNamespaces.map((namespace, index) => ({
      namespace,
      destinations: destinationResults[index]?.data ?? [],
      bindings:
        bindingResults[bindingPermissions.canRead.indexOf(namespace)]?.data ??
        [],
    })),
    classes
  );
  const isLoading =
    destinationPermissions.isPermissionsLoading ||
    destinationResults.some((result) => result.isLoading);
  const hasAddress = rows.some((row) => !!row.address);

  const permissions: MonitoringDestinationPermissions = {
    createNamespaces: toWritableNamespaces({
      destinationNamespaces: destinationPermissions.canCreate,
      secretCreateNamespaces: secretPermissions.canCreate,
      secretReadNamespaces: secretPermissions.canRead,
      canReadClasses,
    }),
    updateNamespaces: toWritableNamespaces({
      destinationNamespaces: destinationPermissions.canUpdate,
      secretCreateNamespaces: secretPermissions.canCreate,
      secretReadNamespaces: secretPermissions.canRead,
      canReadClasses,
    }),
    deleteNamespaces: destinationPermissions.canDelete,
  };

  const [modal, setModal] = useState<ModalState>();
  const [deleteState, setDeleteState] = useState<DeleteState>();
  const fetchBindings = useFetchMonitoringBindings();
  const { mutate: deleteDestination, isPending: isDeleting } =
    useDeleteMonitoringDestination();

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

  // Asks the server who uses the destination right before deleting it.
  const handleDelete = (row: MonitoringDestinationRow) =>
    fetchBindings(row.namespace, row.name)
      .then((bindings) =>
        setDeleteState({
          row,
          usedBy: bindings.map((binding) => binding.spec.instanceRef.name),
        })
      )
      .catch(() => undefined);

  const namespaceDestinationsOf = (namespace: string) =>
    rows.filter((row) => row.namespace === namespace).map((row) => row.raw);

  const handleConfirmDelete = (_name: string, namespace: string) => {
    if (!deleteState) {
      return;
    }
    deleteDestination(
      {
        namespace,
        destination: deleteState.row.raw,
        namespaceDestinations: namespaceDestinationsOf(namespace),
      },
      { onSuccess: () => setDeleteState(undefined) }
    );
  };

  const secretDeletedToo =
    deleteState &&
    getSecretDeletedWithDestination(
      deleteState.row.raw,
      namespaceDestinationsOf(deleteState.row.namespace)
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
              onDelete: handleDelete,
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
      {deleteState &&
        (deleteState.usedBy.length > 0 ? (
          <ConfirmDialog
            open
            selectedId={deleteState.row.name}
            closeModal={() => setDeleteState(undefined)}
            headerMessage={Messages.inUseHeader}
            cancelMessage={Messages.close}
            submitMessage=""
          >
            {Messages.inUse(
              deleteState.row.name,
              deleteState.usedBy,
              IN_USE_NAMES_SHOWN
            )}
          </ConfirmDialog>
        ) : (
          <ConfirmDialog
            open
            selectedId={deleteState.row.name}
            selectedNamespace={deleteState.row.namespace}
            closeModal={() => setDeleteState(undefined)}
            headerMessage={Messages.deleteHeader}
            cancelMessage={Messages.cancel}
            handleConfirmNamespace={handleConfirmDelete}
            disabledButtons={isDeleting}
          >
            {Messages.deleteConfirmation(deleteState.row.name)}
            {secretDeletedToo &&
              ` ${Messages.secretDeletedToo(secretDeletedToo)}`}
          </ConfirmDialog>
        ))}
    </>
  );
};
