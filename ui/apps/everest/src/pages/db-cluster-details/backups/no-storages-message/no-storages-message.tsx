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

import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import { Box, Button, Typography } from '@mui/material';
import { useParams } from 'react-router-dom';
import { useAddBackupStorage } from 'hooks/api/backup-storages/useAddBackupStorage';
import { CreateEditModalStorage } from 'pages/settings/storage-locations/createEditModal/create-edit-modal';
import { Messages } from '../backups.messages';

export const NoStoragesMessage = () => {
  const { namespace = '' } = useParams();
  // Also reports a credentials name that is taken on the form.
  const { canAdd, isOpen, open, modalProps } = useAddBackupStorage(namespace);

  return (
    <Box
      sx={{
        display: 'flex',
        py: 6,
        px: 0,
        flexDirection: 'column',
        alignItems: 'center',
        gap: 1,
        alignSelf: 'stretch',
      }}
    >
      <Box sx={{ fontSize: '100px', lineHeight: 0 }}>
        <WarningAmberIcon fontSize="inherit" />
      </Box>
      <Typography variant="body1">{Messages.noStoragesMessage}</Typography>
      <Button
        sx={{ my: 4 }}
        variant="contained"
        onClick={open}
        disabled={!canAdd}
      >
        {Messages.addStorage}
      </Button>
      {isOpen && <CreateEditModalStorage {...modalProps} />}
    </Box>
  );
};
