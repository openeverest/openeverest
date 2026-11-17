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
import { Box, Typography } from '@mui/material';
import { Messages } from '../backups.messages';

export const NoStoragesMessage = () => (
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
  </Box>
);
