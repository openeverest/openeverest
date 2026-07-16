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
