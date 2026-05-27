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

import { useContext, useState } from 'react';
// import { useMemo } from 'react';
import { Box, Button, MenuItem } from '@mui/material';
// import { Tooltip } from '@mui/material';
import KeyboardArrowDownOutlinedIcon from '@mui/icons-material/KeyboardArrowDownOutlined';
import KeyboardArrowUpOutlined from '@mui/icons-material/KeyboardArrowUpOutlined';
import { MenuButton } from '@percona/ui-lib';
import { ScheduleModalContext } from '../../backups.context';
import { DbInstancePhaseStatus } from 'shared-types/instance.types';
import ScheduledBackupsList from './scheduled-backups-list';
import { BackupListTableHeaderProps } from './backups-list-table-header.types';
import { Messages } from './backups-list-table-header.messages';
import { useRBACPermissions } from 'hooks/rbac';
// TODO: v2 — uncomment when schedule limit checking is implemented
// import { useBackupClassesList } from 'hooks/api/backup-classes/useBackupClasses';
// import { useClusterName } from 'hooks/api/useClusterName';

const BackupListTableHeader = ({
  onNowClick,
  onScheduleClick,
}: BackupListTableHeaderProps) => {
  const [showSchedules, setShowSchedules] = useState(false);
  const { instance } = useContext(ScheduleModalContext);
  // const clusterName = useClusterName();

  const allSchedules =
    instance.spec.backup?.storages?.flatMap((s) => s.schedules ?? []) ?? [];
  const schedulesNumber = allSchedules.length;

  const restoring = instance.status?.phase === DbInstancePhaseStatus.Restoring;

  // TODO: v2 — schedule limit checking, uncomment when ready
  // const { data: backupClasses = [] } = useBackupClassesList(clusterName);
  // const classRef = instance.spec?.backup?.classRef?.name;
  // const activeClass = useMemo(
  //   () => backupClasses.find((bc) => bc.metadata?.name === classRef),
  //   [backupClasses, classRef]
  // );
  // const maxStorages = activeClass?.spec?.providerManaged?.limits?.maxStorages;
  // const scheduleLimitExceeded =
  //   maxStorages != null &&
  //   (instance.spec?.backup?.storages?.length ?? 0) >= maxStorages;
  // const disableScheduleBackups = noStoragesAvailable || scheduleLimitExceeded;

  const handleNowClick = (handleClose: () => void) => {
    onNowClick();
    handleClose();
  };

  const handleScheduleClick = (handleClose: () => void) => {
    onScheduleClick();
    handleClose();
  };

  const handleShowSchedules = () => {
    setShowSchedules((prev) => !prev);
  };

  // TODO: RBAC resource names for v2 are not finalized yet.
  // Using 'backups' as the resource name based on current v2 convention.
  const { canCreate } = useRBACPermissions(
    'backups',
    `${instance.metadata?.namespace}/${instance.metadata?.name}`
  );
  // TODO: v2 — RBAC for instances resource name TBD
  // const { canUpdate: canUpdateInstance } = useRBACPermissions(
  //   'instances',
  //   `${instance.metadata?.namespace}/${instance.metadata?.name}`
  // );

  return (
    <>
      <Box
        sx={(theme) => ({
          [theme.breakpoints.down('md')]: {
            width: '100%',
            order: 1,
          },
        })}
      >
        {/* Order is necessary to keep filters on the left side (i.e. filters have order=0) */}
        {schedulesNumber > 0 && (
          <Button
            size="small"
            data-testid="scheduled-backups"
            sx={{
              ml: 'auto',
              mr: 2,
              position: 'relative',
              ...(showSchedules && {
                '&::after': {
                  content: '""',
                  position: 'absolute',
                  bottom: '-29px',
                  width: '0px',
                  height: '0px',
                  borderStyle: 'solid',
                  borderWidth: '0 14.5px 29px 14.5px',
                  borderColor: (theme) =>
                    `transparent transparent ${theme.palette.surfaces?.elevation0} transparent`,
                  transform: 'rotate(0deg)',
                },
              }),
            }}
            onClick={handleShowSchedules}
            endIcon={
              showSchedules ? (
                <KeyboardArrowUpOutlined />
              ) : (
                <KeyboardArrowDownOutlinedIcon />
              )
            }
          >
            {Messages.activeSchedules(schedulesNumber)}
          </Button>
        )}
        {canCreate && (
          <MenuButton
            matchAnchorWidth
            buttonProps={{
              disabled: restoring,
            }}
            buttonText="Create backup"
            children={(handleClose) => [
              <MenuItem
                key="now"
                data-testid="now-menu-item"
                onClick={() => handleNowClick(handleClose)}
              >
                {Messages.now}
              </MenuItem>,
              <MenuItem
                key="schedule"
                data-testid="schedule-menu-item"
                // TODO: v2 RBAC - disable/hide Schedule when instances:update
                // permission check is available and wired in UI.
                onClick={() => handleScheduleClick(handleClose)}
              >
                {Messages.schedule}
              </MenuItem>,
            ]}
          />
        )}
      </Box>
      {schedulesNumber > 0 && showSchedules && <ScheduledBackupsList />}
    </>
  );
};

export default BackupListTableHeader;
