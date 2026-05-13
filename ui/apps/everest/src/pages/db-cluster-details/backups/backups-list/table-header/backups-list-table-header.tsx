import { useContext, useState } from 'react';
import { Box, Button, MenuItem, Tooltip } from '@mui/material';
import KeyboardArrowDownOutlinedIcon from '@mui/icons-material/KeyboardArrowDownOutlined';
import KeyboardArrowUpOutlined from '@mui/icons-material/KeyboardArrowUpOutlined';
import { MenuButton } from '@percona/ui-lib';
import { ScheduleModalContext } from '../../backups.context';
import { DbInstancePhaseStatus } from 'shared-types/instance.types';
import ScheduledBackupsList from './scheduled-backups-list';
import { BackupListTableHeaderProps } from './backups-list-table-header.types';
import { Messages } from './backups-list-table-header.messages';
import { useRBACPermissions } from 'hooks/rbac';

const BackupListTableHeader = ({
  onNowClick,
  onScheduleClick,
  noStoragesAvailable,
  currentBackups,
}: BackupListTableHeaderProps) => {
  const [showSchedules, setShowSchedules] = useState(false);
  const { instance } = useContext(ScheduleModalContext);

  // In v2, schedules are nested under storages
  const allSchedules =
    instance.spec.backup?.storages?.flatMap((s) => s.schedules ?? []) ?? [];
  const schedulesNumber = allSchedules.length;

  const restoring =
    instance.status?.phase === DbInstancePhaseStatus.Restoring;

  // TODO: In v2, Instance.spec.provider is a string (e.g. "psmdb", "pxc", "pg").
  // PG schedule limit logic needs to be adapted once provider values are finalized.
  // const pgLimitExceeded =
  //   instance.spec.provider === 'pg' &&
  //   allSchedules.length >= 3;
  const pgLimitExceeded = false;
  const disableScheduleBackups = noStoragesAvailable || pgLimitExceeded;

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

  // TODO: RBAC for instances — resource name TBD
  const { canUpdate: canUpdateInstance } = useRBACPermissions(
    'instances',
    `${instance.metadata?.namespace}/${instance.metadata?.name}`
  );

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
            buttonProps={{
              disabled: restoring,
            }}
            buttonText="Create backup"
          >
            {(handleClose) => [
              <MenuItem
                key="now"
                data-testid="now-menu-item"
                onClick={() => handleNowClick(handleClose)}
              >
                {Messages.now}
              </MenuItem>,
              canUpdateInstance && (
                <Box key="schedule">
                  {disableScheduleBackups ? (
                    <Tooltip
                      title={
                        pgLimitExceeded
                          ? Messages.exceededScheduleBackupsNumber
                          : Messages.noStoragesAvailable
                      }
                      placement="right"
                      arrow
                    >
                      <div>
                        <MenuItem data-testid="schedule-menu-item" disabled>
                          {Messages.schedule}
                        </MenuItem>
                      </div>
                    </Tooltip>
                  ) : (
                    <MenuItem
                      onClick={() => handleScheduleClick(handleClose)}
                      data-testid="schedule-menu-item"
                    >
                      {Messages.schedule}
                    </MenuItem>
                  )}
                </Box>
              ),
            ]}
          </MenuButton>
        )}
      </Box>
      {schedulesNumber > 0 && showSchedules && (
        <ScheduledBackupsList currentBackups={currentBackups} />
      )}
    </>
  );
};

export default BackupListTableHeader;
