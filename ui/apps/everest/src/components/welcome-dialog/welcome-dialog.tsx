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
        paper: { sx: { px: 4, pt: 4 } },
      }}
    >
      <DialogContent sx={{ display: 'flex', flexFlow: 'column' }}>
        <Stack
          sx={{
            alignItems: 'center',
          }}
        >
          <EverestMainIcon sx={{ fontSize: '100px', mb: 2 }} />
          <Typography variant="h2">{Messages.allSet}</Typography>
          <Typography
            variant="h6"
            sx={{
              textAlign: 'center',
            }}
          >
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
