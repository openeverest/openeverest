import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  Stack,
  Typography,
} from '@mui/material';
import { EverestMainIcon } from '@percona/ui-lib';
import { useNavigate } from 'react-router-dom';
import { Messages } from './welcome-dialog.messages';

export const WelcomeDialog = ({
  open,
  closeDialog,
}: {
  open: boolean;
  closeDialog: () => void;
}) => {
  const navigate = useNavigate();

  const handleRedirectHome = () => {
    navigate('/');
    closeDialog();
  };

  return (
    <Dialog
      open={open}
      onClose={closeDialog}
      slotProps={{
        paper: { sx: { px: 4, pt: 4 } }
      }}
    >
      <DialogContent sx={{ display: 'flex', flexFlow: 'column' }}>
        <Stack sx={{
          alignItems: "center"
        }}>
          <EverestMainIcon sx={{ fontSize: '100px', mb: 2 }} />
          <Typography variant="h2">{Messages.allSet}</Typography>
          <Typography variant="h6" sx={{
            textAlign: "center"
          }}>
            {Messages.start}
          </Typography>
        </Stack>
      </DialogContent>
      {/* TODO: remove dialog actions when cards are uncommented */}
      <DialogActions sx={{ mt: 4 }}>
        <Button
          onClick={handleRedirectHome}
          variant="contained"
          size="large"
          data-testid="lets-go-button"
        >
          {Messages.letsGo}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
