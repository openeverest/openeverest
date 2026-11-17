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

import { Box, Tooltip } from '@mui/material';
import { PendingIcon } from '@percona/ui-lib';
import StatusField from 'components/status-field';
import { DB_INSTANCE_STATUS_TO_BASE_STATUS } from 'pages/databases/DbClusterView.constants';
import { beautifyDbInstanceStatus } from 'pages/databases/DbClusterView.utils';
import type { Instance } from 'shared-types/api.types';
import {
  DB_INSTANCE_UNKNOWN_PHASE,
  DbInstancePhase,
  getPodsAlertConditions,
} from 'shared-types/instance.types';
import { Messages } from './instance-status.messages';

interface InstanceStatusProps {
  instance?: Instance;
  dataTestId?: string;
}

export const InstanceStatus = ({
  instance,
  dataTestId,
}: InstanceStatusProps) => {
  const phase: DbInstancePhase =
    instance?.status?.phase ?? DB_INSTANCE_UNKNOWN_PHASE;
  const hasPodsAlert = getPodsAlertConditions(instance).length > 0;

  return (
    <Tooltip
      title={hasPodsAlert ? Messages.podsNeedAttention : ''}
      placement="right"
      arrow
    >
      <Box component="span" sx={{ display: 'inline-flex' }}>
        <StatusField
          dataTestId={dataTestId}
          status={phase}
          statusMap={
            hasPodsAlert
              ? { ...DB_INSTANCE_STATUS_TO_BASE_STATUS, [phase]: 'warning' }
              : DB_INSTANCE_STATUS_TO_BASE_STATUS
          }
          defaultIcon={PendingIcon}
        >
          {beautifyDbInstanceStatus(phase)}
        </StatusField>
      </Box>
    </Tooltip>
  );
};
