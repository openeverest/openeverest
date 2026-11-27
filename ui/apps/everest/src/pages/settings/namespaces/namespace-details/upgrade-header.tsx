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
  Alert,
  Box,
  Button,
  CircularProgress,
  LinearProgress,
  Typography,
} from '@mui/material';
import { formatDistanceToNow } from 'date-fns';
import { UpgradeHeaderProps } from './types';
import { Messages } from './messages';

const UpgradeHeader = ({
  upgradeAvailable,
  pendingUpgradeTasks,
  upgradeAllowed,
  onUpgrade,
  upgradePhase,
  ...boxProps
}: UpgradeHeaderProps) => {
  if (upgradePhase.phase === 'operator-upgrading') {
    const elapsed = formatDistanceToNow(new Date(upgradePhase.startedAt), {
      addSuffix: true,
    });
    return (
      <Box display="flex" alignItems="center" gap={2} {...boxProps}>
        <CircularProgress size={20} />
        <Typography variant="body1">
          Upgrading operator to v{upgradePhase.targetVersion}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Started {elapsed}
        </Typography>
      </Box>
    );
  }

  if (upgradePhase.phase === 'actions-required') {
    const { resolvedCount, totalCount } = upgradePhase;
    const progress =
      resolvedCount + totalCount > 0
        ? (resolvedCount / (resolvedCount + totalCount)) * 100
        : 0;
    return (
      <Box {...boxProps}>
        <Box
          display="flex"
          justifyContent="space-between"
          alignItems="center"
          mb={1}
        >
          <Typography variant="body1">
            Operator upgraded — {totalCount} cluster
            {totalCount !== 1 ? 's' : ''} require attention.
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {resolvedCount} of {resolvedCount + totalCount} resolved
          </Typography>
        </Box>
        <LinearProgress variant="determinate" value={progress} />
      </Box>
    );
  }

  if (upgradePhase.phase === 'error') {
    const { stuckClusters } = upgradePhase;
    return (
      <Box {...boxProps}>
        <Alert severity="error">
          Upgrade incomplete: <strong>{stuckClusters.join(', ')}</strong>{' '}
          {stuckClusters.length === 1 ? 'is' : 'are'} not ready. Check the
          database{stuckClusters.length !== 1 ? 's' : ''} and retry
          individually.
        </Alert>
      </Box>
    );
  }

  if (!upgradeAvailable) {
    return null;
  }

  return (
    <Box
      display="flex"
      justifyContent="space-between"
      alignItems="center"
      {...boxProps}
    >
      <Typography variant="body1">
        A new version of the operators is available.
        {pendingUpgradeTasks
          ? ' Start upgrading by performing all the pending tasks in the Actions column.'
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
