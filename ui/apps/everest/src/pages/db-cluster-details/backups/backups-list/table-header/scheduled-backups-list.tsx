import { useContext, useEffect, useState } from 'react';
import { Box, IconButton, Paper, Stack, Typography } from '@mui/material';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import { ManageableSchedules } from 'shared-types/dbCluster.types';
import { ConfirmDialog } from 'components/confirm-dialog/confirm-dialog';
import { ScheduleModalContext } from '../../backups.context';
import { getTimeSelectionPreviewMessage } from 'pages/database-form/database-preview/database.preview.messages';
import { getFormValuesFromCronExpression } from 'components/time-selection/time-selection.utils';
import { Messages } from './backups-list-table-header.messages';
import { useRBACPermissions } from 'hooks/rbac';
import {
  deleteScheduleFromDbCluster,
  transformSchedulesIntoManageableSchedules,
} from 'utils/db';
import { useUpdateDbClusterWithConflictRetry } from 'hooks';
import { WizardMode } from 'shared-types/wizard.types';
import { DbEngineType } from 'shared-types/dbEngines.types';
import { Backup, BackupStatus } from 'shared-types/backups.types';

type Props = {
  currentBackups: Backup[];
};

const ScheduledBackupsList = ({ currentBackups }: Props) => {
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);
  const [selectedSchedule, setSelectedSchedule] = useState<string>('');
  const {
    dbCluster,
    setMode: setScheduleModalMode,
    setSelectedScheduleName: setSelectedScheduleToModalContext,
    setOpenScheduleModal,
  } = useContext(ScheduleModalContext);
  const { mutate: updateCluster, isPending: updatingCluster } =
    useUpdateDbClusterWithConflictRetry(dbCluster, {
      onSuccess: () => handleCloseDeleteDialog(),
    });
  const [schedules, setSchedules] = useState<ManageableSchedules[]>([]);
  const pitrEnabled = !!dbCluster.spec?.backup?.pitr?.enabled;
  const emptyBackups =
    currentBackups.length === 0 ||
    currentBackups.every((b) => b.state === BackupStatus.DELETING);
  const willDisablePITR = emptyBackups && schedules.length === 1 && pitrEnabled;
  const handleDelete = (scheduleName: string) => {
    setSelectedSchedule(scheduleName);
    setOpenDeleteDialog(true);
  };

  const handleCloseDeleteDialog = () => {
    setOpenDeleteDialog(false);
  };

  const handleConfirmDelete = (scheduleName: string) => {
    const newClusterData = deleteScheduleFromDbCluster(
      scheduleName,
      dbCluster,
      willDisablePITR
    );

    if (dbCluster.spec.engine.type === DbEngineType.POSTGRESQL) {
      if (
        currentBackups.length === 0 &&
        !newClusterData.spec.backup?.schedules?.length &&
        newClusterData.spec.backup?.pitr
      ) {
        newClusterData.spec.backup = {
          ...newClusterData.spec.backup,
          pitr: {
            ...newClusterData.spec.backup?.pitr,
            enabled: false,
            backupStorageName: '',
          },
        };
      }
    }
    updateCluster(newClusterData);
  };

  const handleEdit = (scheduleName: string) => {
    setScheduleModalMode(WizardMode.Edit);
    setSelectedScheduleToModalContext(scheduleName);
    setOpenScheduleModal(true);
  };

  const { canUpdate: canUpdateDb } = useRBACPermissions(
    'database-clusters',
    `${dbCluster.metadata.namespace}/${dbCluster.metadata.name}`
  );

  const { canCreate: canCreateBackups } = useRBACPermissions(
    'database-cluster-backups',
    `${dbCluster.metadata.namespace}/${dbCluster.metadata.name}`
  );

  useEffect(() => {
    transformSchedulesIntoManageableSchedules(
      dbCluster.spec.backup?.schedules || [],
      dbCluster.metadata.namespace,
      canCreateBackups,
      canUpdateDb
    ).then((newSchedules) => {
      setSchedules(newSchedules);
    });
  }, [canCreateBackups, canUpdateDb, dbCluster]);

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
          key={`schedule-${item?.name}`}
          sx={{
            py: 1,
            px: 2,
            borderRadius: 1,
            boxShadow: 'none',
          }}
          data-testid={`schedule-${item?.name}`}
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
                  data-testid={`schedule-${item?.schedule}-text`}
                  variant="body2"
                >
                  {getTimeSelectionPreviewMessage(
                    getFormValuesFromCronExpression(item.schedule)
                  )}
                </Typography>
              </Stack>
            </Box>
            <Box sx={{ width: '30%' }}>
              <Typography variant="body2">
                {`Retention copies: 
                ${item?.retentionCopies || 'infinite'}`}
              </Typography>
            </Box>
            <Box sx={{ width: '15%' }}>
              {' '}
              <Typography variant="body2">
                {' '}
                {`Storage: ${item.backupStorageName}`}
              </Typography>
            </Box>
            <Box display="flex">
              {item.canBeManaged && (
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
          disabledButtons={updatingCluster}
        >
          {Messages.deleteModal.content(selectedSchedule, willDisablePITR)}
        </ConfirmDialog>
      )}
    </Stack>
  );
};

export default ScheduledBackupsList;
