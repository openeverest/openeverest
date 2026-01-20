import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  Typography,
} from '@mui/material';
import { DialogTitle } from '@percona/ui-lib';
import { Messages } from './upgrade-reload-everest-dialog.messages';
import KnowMoreButton from 'components/know-more-button/know-more-button';

interface UpgradeEverestReloadDialogProps {
  isOpen: boolean;
  closeModal: () => void;
  version: string;
}
const UpgradeEverestReloadDialog = ({
  isOpen,
  closeModal,
  version,
}: UpgradeEverestReloadDialogProps) => {
  return (
    <Dialog
      open={isOpen}
      onClose={closeModal}
      data-testid="everest-reload-dialog"
      sx={{ '.MuiPaper-root': { minWidth: '640px' } }}
    >
      <DialogTitle onClose={closeModal}>{Messages.headerMessage}</DialogTitle>
      <DialogContent sx={{ whiteSpace: 'pre-wrap' }}>
        <Typography variant="body1">
          {Messages.successfullyUpgraded(version)}
          <KnowMoreButton href="https://openeverest.io/documentation/current/release-notes/release_notes_index.html" />
        </Typography>
        <Typography variant="body1">{Messages.clickToReload}</Typography>
      </DialogContent>
      <DialogActions>
        <Button
          variant="contained"
          onClick={() => {
            window.location.reload();
          }}
          data-testid="everest-reload-dialog"
        >
          {Messages.cancelMessage}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default UpgradeEverestReloadDialog;
