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

import { useMemo, useState } from 'react';
import { Box, Button, Dialog, DialogContent, IconButton, Menu, MenuItem } from '@mui/material';
import {
  DeleteOutline as DeleteOutlineIcon,
  KeyboardReturn as KeyboardReturnIcon,
  // Add as AddIcon, // TODO: re-enable when create-new-db-from-backup is restored
} from '@mui/icons-material';
import ExtensionIcon from '@mui/icons-material/Extension';
import MoreHorizIcon from '@mui/icons-material/MoreHoriz';
import { DbActionsProps } from './db-actions.types';
import { useRBACPermissions } from 'hooks/rbac';
import { Messages } from './db-actions.messages';
import { ArrowDropDownIcon } from '@mui/x-date-pickers/icons';
import DbActionsModals from './db-actions-modals';
import { useDbInstanceActions } from 'hooks/api/db-instance';
import { usePlugins } from 'contexts/plugins';
import type { ClusterActionExtension } from '@openeverest/plugin-sdk';
import PluginErrorBoundary from 'components/plugin-host/PluginErrorBoundary';
import { useBackupsList } from 'hooks/api/backups/useBackups';
import { useClusterName } from 'hooks/api/useClusterName';
import { BackupStatus } from 'shared-types/backups.types';

export const DbActions = ({
  // showDetailsAction = false,
  showStatusActions = false,
  dbInstance,
}: DbActionsProps) => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [isNewClusterMode, setIsNewClusterMode] = useState(false);
  const [activePluginAction, setActivePluginAction] = useState<{
    pluginName: string;
    ext: ClusterActionExtension;
  } | null>(null);
  const {
    openRestoreDialog,
    handleCloseRestoreDialog,
    // handleDbRestart,
    handleDeleteDbInstance,
    // isPaused,
    openDeleteDialog,
    handleConfirmDelete,
    handleCloseDeleteDialog,
    openDetailsDialog,
    // handleOpenDbDetailsDialog,
    handleCloseDetailsDialog,
    // handleDbSuspendOrResumed,
    handleRestoreDbCluster,
    deleteMutation,
  } = useDbInstanceActions(dbInstance);
  const open = Boolean(anchorEl);
  const dbInstanceName = dbInstance.metadata?.name;
  const namespace = dbInstance.metadata?.namespace ?? '';
  const clusterName = useClusterName();

  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    dbInstanceName ?? ''
  );
  const hasBackups = backups.length > 0;
  const hasReadyBackup = backups.some(
    (b) => b.status?.state === BackupStatus.SUCCEEDED
  );
  // const redirectURL = `/databases/${namespace}/${dbInstanceName}/overview`;

  // const navigate = useNavigate();
  // TODO needs a final enum
  // const actionsBlocked = shouldDbActionsBeBlocked(dbInstance.status?.phase as DbInstanceStatus || '');
  const actionsBlocked = dbInstance?.status?.phase === 'Terminating';
  // const hasSchedules = !!(
  //   dbInstance.spec.backup && (dbInstance.spec.backup.schedules || []).length > 0
  // );
  // const monitoringEnabled = !!(
  //   dbInstance.spec.monitoring && dbInstance.spec.monitoring?.monitoringConfigName
  // );
  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    setAnchorEl(event.currentTarget);
  };
  const closeMenu = () => {
    setAnchorEl(null);
  };

  const { /*canUpdate*/ canDelete } = useRBACPermissions(
    'database-clusters',
    `${dbInstance.metadata?.namespace}/${dbInstance.metadata?.name}`
  );

  // const { canCreate: canCreateClusters } = useRBACPermissions(
  //   'database-clusters',
  //   `${dbInstance.metadata?.namespace}/*`
  // );

  // const { canCreate: canCreateRestore } = useRBACPermissions(
  //   'database-cluster-restores',
  //   `${namespace}/*`
  // );

  // const { canRead: canReadCredentials } = useRBACPermissions(
  //   'database-cluster-credentials',
  //   `${namespace}/${dbInstanceName}`
  // );

  // const { canCreate: canCreateBackups } = useRBACPermissions(
  //   'database-cluster-backups',
  //   `${namespace}/${dbInstanceName}`
  // );

  // const { canRead: canReadMonitoring } = useRBACPermissions(
  //   'monitoring-instances',
  //   `${namespace}/${dbInstanceName}`
  // );

  // const canRestore = canCreateRestore && canReadCredentials;
  // TODO RBAC
  // const noActionAvailable = !canUpdate && !canDelete && !canRestore;
  const noActionAvailable = false;

  // Collect plugin clusterAction extensions.
  const { plugins } = usePlugins();
  const pluginActions = useMemo(
    () =>
      plugins.flatMap((p) =>
        p.extensions
          .filter((ext): ext is ClusterActionExtension => ext.type === 'clusterAction')
          .map((ext) => ({ pluginName: p.name, ext })),
      ),
    [plugins],
  );
  // let canCreateClusterFromBackup = canRestore && canCreateClusters;

  // if (hasSchedules) {
  //   canCreateClusterFromBackup = canCreateClusterFromBackup && canCreateBackups;
  // }

  // if (monitoringEnabled) {
  //   canCreateClusterFromBackup =
  //     canCreateClusterFromBackup && canReadMonitoring;
  // }

  const sx = {
    display: 'flex',
    gap: 1,
    alignItems: 'center',
    px: 2,
    py: '10px',
  };

  if (noActionAvailable) {
    return null;
  }

  return (
    <>
      <Box>
        {showStatusActions ? (
          <Button
            id="actions-button"
            data-testid="actions-button"
            aria-controls={open ? 'actions-button-menu' : undefined}
            aria-haspopup="true"
            aria-expanded={open ? 'true' : undefined}
            onClick={handleClick}
            variant="text"
            size="large"
            endIcon={<ArrowDropDownIcon />}
          >
            Actions
          </Button>
        ) : (
          <IconButton
            data-testid="actions-menu-button"
            aria-haspopup="true"
            aria-controls={open ? 'basic-menu' : undefined}
            aria-expanded={open ? 'true' : undefined}
            onClick={handleClick}
          >
            <MoreHorizIcon />
          </IconButton>
        )}

        <Menu
          id="actions-button"
          anchorEl={anchorEl}
          open={open}
          onClose={closeMenu}
          onClick={closeMenu}
          MenuListProps={{
            'aria-labelledby': 'row-actions-button',
          }}
        >
          {/*showDetailsAction && (
            <MenuItem
              data-testid={`${dbInstanceName}-details`}
              key={0}
              onClick={() => {
                navigate(redirectURL);
              }}
              sx={sx}
            >
              <VisibilityOutlinedIcon /> {Messages.menuItems.dbDetails}
            </MenuItem>
          )*/}
          {/*canUpdate && (
            <MenuItem
              disabled={actionsBlocked}
              key={2}
              onClick={() => {
                handleDbRestart(dbInstance);
              }}
              sx={sx}
            >
              <RestartAltIcon /> {Messages.menuItems.restart}
            </MenuItem>
          )*/}
          {/* TODO: Temporarily hidden — create new DB from backup deferred by team */}
          {/* <MenuItem
            data-testid={`${dbInstanceName}-create-new-db-from-backup`}
            disabled={actionsBlocked}
            key={1}
            onClick={() => {
              setIsNewClusterMode(true);
              handleRestoreDbCluster();
            }}
            sx={sx}
          >
            <AddIcon /> {Messages.menuItems.createNewDbFromBackup}
          </MenuItem> */}
          {/*TODO RBAC */}
          {hasBackups && (
            <MenuItem
              data-testid={`${dbInstanceName}-restore`}
              disabled={actionsBlocked || !hasReadyBackup}
              key={3}
              onClick={() => {
                setIsNewClusterMode(false);
                handleRestoreDbCluster();
              }}
              sx={sx}
            >
              <KeyboardReturnIcon /> {Messages.menuItems.restoreFromBackup}
            </MenuItem>
          )}
          {/*
          {showStatusActions && dbInstance?.status?.details && (
            <MenuItem
              key={6}
              sx={sx}
              onClick={() => {
                handleOpenDbDetailsDialog();
              }}
            >
              <VisibilityOutlinedIcon /> {Messages.menuItems.dbStatusDetails}
            </MenuItem>
          )} */}
          {/* {canUpdate && (
            <MenuItem
              disabled={actionsBlocked}
              key={4}
              onClick={() => {
                handleDbSuspendOrResumed(dbInstance);
              }}
              sx={sx}
            >
              <PauseCircleOutline />{' '}
              {isPaused(dbInstance)
                ? Messages.menuItems.resume
                : Messages.menuItems.suspend}
            </MenuItem>
          )} */}
          {canDelete && (
            <MenuItem
              disabled={actionsBlocked}
              data-testid={`${dbInstanceName}-delete`}
              key={5}
              onClick={() => {
                handleDeleteDbInstance();
              }}
              sx={sx}
            >
              <DeleteOutlineIcon /> {Messages.menuItems.delete}
            </MenuItem>
          )}
          {pluginActions.map((pa) => (
            <MenuItem
              key={`plugin-${pa.pluginName}-${pa.ext.label}`}
              data-testid={`${dbInstanceName}-plugin-${pa.pluginName}`}
              onClick={() => setActivePluginAction(pa)}
              sx={sx}
            >
              <ExtensionIcon /> {pa.ext.label}
            </MenuItem>
          ))}
        </Menu>
      </Box>
      <DbActionsModals
        dbInstance={dbInstance}
        isNewClusterMode={isNewClusterMode}
        openRestoreDialog={openRestoreDialog}
        handleCloseRestoreDialog={handleCloseRestoreDialog}
        openDeleteDialog={openDeleteDialog}
        handleCloseDeleteDialog={handleCloseDeleteDialog}
        handleConfirmDelete={handleConfirmDelete}
        openDetailsDialog={openDetailsDialog}
        handleCloseDetailsDialog={handleCloseDetailsDialog}
        deleteMutation={deleteMutation}
      />
      {activePluginAction && (() => {
        const ActionComponent = activePluginAction.ext.component;
        return (
          <Dialog
            open
            onClose={() => setActivePluginAction(null)}
            maxWidth="md"
            fullWidth
          >
            <DialogContent>
              <PluginErrorBoundary pluginName={activePluginAction.pluginName}>
                <ActionComponent
                  cluster={dbInstance}
                  namespace={dbInstance.metadata?.namespace ?? ''}
                  onClose={() => setActivePluginAction(null)}
                />
              </PluginErrorBoundary>
            </DialogContent>
          </Dialog>
        );
      })()}
    </>
  );
};

export default DbActions;
