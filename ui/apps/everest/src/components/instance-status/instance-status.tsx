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
import { InstanceStatusSource } from 'shared-types/instance.types';
import {
  getInstanceStatusDisplay,
  getInstanceStatusText,
} from 'utils/instance-status';
import { TONE_TO_BASE_STATUS } from './instance-status.constants';

interface InstanceStatusProps {
  instance?: InstanceStatusSource;
  dataTestId?: string;
}

export const InstanceStatus = ({
  instance,
  dataTestId,
}: InstanceStatusProps) => {
  const status = getInstanceStatusDisplay(instance);

  return (
    <Tooltip title={status.message} placement="right" arrow>
      <Box component="span" sx={{ display: 'inline-flex' }}>
        <StatusField
          dataTestId={dataTestId}
          status={status.tone}
          statusMap={TONE_TO_BASE_STATUS}
          defaultIcon={PendingIcon}
        >
          {getInstanceStatusText(instance, status)}
        </StatusField>
      </Box>
    </Tooltip>
  );
};
