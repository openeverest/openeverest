import { Box, Button } from '@mui/material';
import { BackupListTableHeaderProps } from './backups-list-table-header.types';
import { Messages } from './backups-list-table-header.messages';

const BackupListTableHeader = ({
  onNowClick,
}: BackupListTableHeaderProps) => {
  return (
    <Box
      sx={(theme) => ({
        [theme.breakpoints.down('md')]: {
          width: '100%',
          order: 1,
        },
      })}
    >
      <Button
        variant="contained"
        size="small"
        data-testid="backup-now-button"
        onClick={onNowClick}
      >
        {Messages.createBackup}
      </Button>
    </Box>
  );
};

export default BackupListTableHeader;
