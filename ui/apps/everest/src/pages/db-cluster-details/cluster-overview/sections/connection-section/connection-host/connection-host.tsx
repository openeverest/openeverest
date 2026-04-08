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

import { Box, Typography } from '@mui/material';
import { CopyToClipboardButton } from '@percona/ui-lib';

export interface ConnectionHostProps {
  host: string;
}

const ConnectionHost = ({ host }: ConnectionHostProps) => (
  <Box sx={{ display: 'flex', gap: 1 }}>
    <Typography variant="body2">{host}</Typography>
    <CopyToClipboardButton
      buttonProps={{
        color: 'primary',
        size: 'small',
        sx: { mt: -0.5 },
      }}
      textToCopy={host}
    />
  </Box>
);

export default ConnectionHost;
