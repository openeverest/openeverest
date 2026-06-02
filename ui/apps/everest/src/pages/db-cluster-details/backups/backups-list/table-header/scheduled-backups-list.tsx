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
import { Box, IconButton, Paper, Stack, Typography } from '@mui/material';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import { getTimeSelectionPreviewMessage } from 'pages/database-form/database-preview/database.preview.messages';
import { getFormValuesFromCronExpression } from 'components/time-selection/time-selection.utils';
import { ScheduleModalContext } from '../../backups.context';
import { Messages } from './backups-list-table-header.messages';
import { useRBACPermissions } from 'hooks/rbac';
import { useUpdateDbInstanceWithConflictRetry } from 'hooks/api/db-instances/useUpdateDbInstance';
import { useBackupsList } from 'hooks/api/backups/useBackups';
import { useClusterName } from 'hooks/api/useClusterName';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { Instance } from 'shared-types/api.types';
import { flattenSchedules, removeUnusedStorages } from '../../backups.utils';

const ScheduledBackupsList = () => {
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);
  const [selectedSchedule, setSelectedSchedule] = useState('');
  const {
    instance,
    setMode: setScheduleModalMode,
    setSelectedScheduleName: setSelectedScheduleToModalContext,
    setOpenScheduleModal,
  } = useContext(ScheduleModalContext);

  const clusterName = useClusterName();
  const namespace = instance.metadata?.namespace ?? '';
  const instanceName = instance.metadata?.name ?? '';
  const { data: backups = [] } = useBackupsList(
    clusterName,
    namespace,
    instanceName
  );

  const { mutate: updateInstance, isPending: updatingInstance } =
    useUpdateDbInstanceWithConflictRetry(instance);

  const schedules = flattenSchedules(instance);

  // TODO: PITR delete logic — when PITR support is added, disable PITR if
  // the last schedule is deleted and there are no remaining backups.
  const willDisablePITR = false;

  const handleDelete = (scheduleName: string) => {
    setSelectedSchedule(scheduleName);
    setOpenDeleteDialog(true);
  };

  const handleCloseDeleteDialog = () => {
    setOpenDeleteDialog(false);
  };

  const handleConfirmDelete = (scheduleName: string) => {
    handleCloseDeleteDialog();
    // Remove the schedule from its storage entry.
    const storagesWithoutSchedule = (instance.spec?.backup?.storages ?? []).map(
      (storage) => ({
        ...storage,
        schedules: (storage.schedules ?? []).filter(
          (s) => s.name !== scheduleName
        ),
      })
    );

    // Remove storage entries that no longer have schedules or active backups.
    const updatedStorages = removeUnusedStorages(
      storagesWithoutSchedule,
      backups
    );

    const updatedInstance: Instance = {
      ...instance,
      spec: {
        ...instance.spec,
        backup: {
          ...instance.spec?.backup,
          classRef: instance.spec?.backup?.classRef ?? { name: '' },
          enabled: instance.spec?.backup?.enabled ?? true,
          storages: updatedStorages,
        },
      },
    };

    updateInstance(updatedInstance);
  };

  const handleEdit = (scheduleName: string) => {
    setScheduleModalMode(FormMode.Edit);
    setSelectedScheduleToModalContext(scheduleName);
    setOpenScheduleModal(true);
  };

  const { canUpdate: canUpdateInstance } = useRBACPermissions(
    'instances',
    `${instance.metadata?.namespace}/${instance.metadata?.name}`
  );

  return (
    <Stack
      useFlexGap
      spacing={1}
      width="100%"
      order={3}
      bgcolor={(theme) => theme.palette.surfaces?.elevation0}
      p={2}
      mt={2}
    >
      {schedules.map((item) => (
        <Paper
          key={`schedule-${item.name}`}
          sx={{
            py: 1,
            px: 2,
            borderRadius: 1,
            boxShadow: 'none',
          }}
          data-testid={`schedule-${item.name}`}
        >
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'row',
              alignItems: 'center',
            }}
          >
            <Box sx={{ width: '40%' }}>
              <Stack>
                <Typography variant="body1">{item.name}</Typography>
                <Typography
                  data-testid={`schedule-${item.cron}-text`}
                  variant="body2"
                >
                  {getTimeSelectionPreviewMessage(
                    getFormValuesFromCronExpression(item.cron)
                  )}
                </Typography>
              </Stack>
            </Box>
            <Box sx={{ width: '30%' }}>
              <Typography variant="body2">
                {`Retention copies: ${item.retentionCopies || 'infinite'}`}
              </Typography>
            </Box>
            <Box sx={{ width: '15%' }}>
              <Typography variant="body2">
                {`Storage: ${item.storageName}`}
              </Typography>
            </Box>
            <Box display="flex" ml="auto">
              {canUpdateInstance && (
                <>
                  <IconButton
                    color="primary"
                    onClick={() => handleEdit(item.name)}
                    data-testid="edit-schedule-button"
                  >
                    <EditOutlinedIcon />
                  </IconButton>
                  <IconButton
                    color="primary"
                    onClick={() => handleDelete(item.name)}
                    data-testid="delete-schedule-button"
                  >
                    <DeleteOutlineOutlinedIcon />
                  </IconButton>
                </>
              )}
            </Box>
          </Box>
        </Paper>
      ))}
      {openDeleteDialog && (
        <ConfirmDialog
          open={openDeleteDialog}
          selectedId={selectedSchedule}
          closeModal={handleCloseDeleteDialog}
          cancelMessage="Cancel"
          headerMessage={Messages.deleteModal.header}
          handleConfirm={handleConfirmDelete}
          disabledButtons={updatingInstance}
        >
          {Messages.deleteModal.content(selectedSchedule, willDisablePITR)}
        </ConfirmDialog>
      )}
    </Stack>
  );
};

export default ScheduledBackupsList;
