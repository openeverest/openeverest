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
import { Box, MenuItem } from '@mui/material';
import { MenuButton } from '@percona/ui-lib';
import { ScheduleModalContext } from '../../backups.context';
import { DbInstancePhaseStatus } from 'shared-types/instance.types';
import ScheduledBackupsList from './scheduled-backups-list';
import { StoragesList } from '../../storages-list';
import { ExpandableSectionToggle } from 'components/expandable-section-toggle';
import { BackupListTableHeaderProps } from './backups-list-table-header.types';
import { Messages } from './backups-list-table-header.messages';
import { useRBACPermissions } from 'hooks/rbac';
import { useActiveBackupClass } from 'hooks/api/backup-classes/useBackupClasses';
// TODO: v2 — uncomment when schedule limit checking is implemented
// import { useBackupClassesList } from 'hooks/api/backup-classes/useBackupClasses';
// import { useClusterName } from 'hooks/api/useClusterName';

const BackupListTableHeader = ({
  onNowClick,
  onScheduleClick,
}: BackupListTableHeaderProps) => {
  const [expandedSection, setExpandedSection] = useState<
    'schedules' | 'storages' | null
  >(null);
  const { instance } = useContext(ScheduleModalContext);
  // const clusterName = useClusterName();

  const allSchedules =
    instance.spec.backup?.storages?.flatMap((s) => s.schedules ?? []) ?? [];
  const schedulesNumber = allSchedules.length;
  const storagesNumber = instance.spec.backup?.storages?.length ?? 0;

  // PITR lives on backup storages; the panel is only relevant when the provider
  // supports it and at least one storage exists.
  const activeClass = useActiveBackupClass(instance);
  const showPitrPanel =
    (activeClass?.spec?.providerManaged?.supportsPITR ?? false) &&
    storagesNumber > 0;

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
    setExpandedSection((prev) => (prev === 'schedules' ? null : 'schedules'));
  };

  const handleShowStorages = () => {
    setExpandedSection((prev) => (prev === 'storages' ? null : 'storages'));
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
        {/* Toggles are right-aligned as a group (inline-flex so the row is not broken); filters stay left (order=0). */}
        {(schedulesNumber > 0 || showPitrPanel) && (
          <Box sx={{ display: 'inline-flex', ml: 'auto' }}>
            {schedulesNumber > 0 && (
              <ExpandableSectionToggle
                label={Messages.activeSchedules(schedulesNumber)}
                open={expandedSection === 'schedules'}
                onToggle={handleShowSchedules}
                dataTestId="scheduled-backups"
              />
            )}
            {showPitrPanel && (
              <ExpandableSectionToggle
                label={Messages.pitr}
                open={expandedSection === 'storages'}
                onToggle={handleShowStorages}
                dataTestId="storages-toggle"
              />
            )}
          </Box>
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
      {schedulesNumber > 0 && expandedSection === 'schedules' && (
        <ScheduledBackupsList />
      )}
      {showPitrPanel && expandedSection === 'storages' && <StoragesList />}
    </>
  );
};

export default BackupListTableHeader;
