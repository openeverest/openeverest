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

import { Box, Button, Typography } from '@mui/material';
import { UpgradeHeaderProps } from './types';
import { Messages } from './messages';

const UpgradeHeader = ({
  upgradeAvailable,
  pendingUpgradeTasks,
  upgradeAllowed,
  onUpgrade,
  ...boxProps
}: UpgradeHeaderProps) => {
  if (!upgradeAvailable) {
    return null;
  }

  return (
    <Box
      {...boxProps}
      sx={[
        {
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        },
        ...(Array.isArray(boxProps.sx) ? boxProps.sx : [boxProps.sx]),
      ]}
    >
      <Typography variant="body1">
        A new version of the operators is available.
        {pendingUpgradeTasks
          ? 'Start upgrading by performing all the pending tasks in the Actions column.'
          : ''}
      </Typography>
      <Button
        size="medium"
        variant="contained"
        onClick={onUpgrade}
        disabled={pendingUpgradeTasks || !upgradeAllowed}
      >
        {Messages.upgradeOperators}
      </Button>
    </Box>
  );
};
export default UpgradeHeader;
