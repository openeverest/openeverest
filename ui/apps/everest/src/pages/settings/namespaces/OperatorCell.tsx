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

// TODO: restore when v2 API for db-engines/upgrade-plan is available
// @ts-nocheck

import {
  Button,
  CircularProgress,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import { useRBACPermissions } from 'hooks/rbac';
import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';

interface NamespaceInstance {
  name: string;
  upgradeAvailable: boolean;
  isUpgrading: boolean;
  operators: string[];
  pendingActions: { pendingTask: string }[];
  operatorsDescription: string;
}

export const OperatorCell = ({
  description,
  namespaceInstance: {
    name,
    operators,
    upgradeAvailable,
    isUpgrading,
    pendingActions,
  },
}: {
  description: string;
  namespaceInstance: NamespaceInstance;
}) => {
  const operatorsToCheck = useMemo(
    () => operators.map((operator) => `${name}/${operator}`),
    [name, operators]
  );
  const { canRead } = useRBACPermissions('database-engines', operatorsToCheck);
  const navigate = useNavigate();
  const somePendingTask =
    pendingActions.filter(
      (a) => a.pendingTask === 'restart' || a.pendingTask === 'upgradeEngine'
    ).length > 0;

  const showUpgradeButton =
    (upgradeAvailable || somePendingTask) && canRead && !isUpgrading;

  return (
    <Stack direction="row" alignItems="center" width="100%">
      <Typography variant="body1">{description}</Typography>
      {isUpgrading && canRead && (
        <Tooltip title="Operator upgrade in progress">
          <Stack
            direction="row"
            alignItems="center"
            gap={1}
            sx={{ ml: 'auto' }}
          >
            <CircularProgress size={16} />
            <Typography variant="body2" color="text.secondary">
              Upgrading
            </Typography>
          </Stack>
        </Tooltip>
      )}
      {showUpgradeButton && (
        <Button
          onClick={() => navigate(`/settings/namespaces/${name}`)}
          sx={{ ml: 'auto' }}
        >
          Upgrade
        </Button>
      )}
    </Stack>
  );
};
