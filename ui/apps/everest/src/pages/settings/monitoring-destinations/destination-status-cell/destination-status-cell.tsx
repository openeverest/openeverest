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

import { Tooltip, Typography } from '@mui/material';
import StatusField from 'components/status-field';
import { MonitoringDestinationStatus } from 'shared-types/monitoring-destinations.types';
import { DESTINATION_STATE_TO_STATUS } from '../monitoring-destinations.constants';
import { Messages } from '../monitoring-destinations.messages';

interface DestinationStatusCellProps {
  status: MonitoringDestinationStatus;
}

// The state, with the reason behind it on hover.
export const DestinationStatusCell = ({
  status: { state, reason, message, serverVersion },
}: DestinationStatusCellProps) => {
  const explanation = [reason, message].filter(Boolean).join(': ');
  const label = [
    Messages.state[state],
    serverVersion && Messages.serverVersion(serverVersion),
  ]
    .filter(Boolean)
    .join(' · ');

  return (
    <Tooltip title={explanation} placement="top">
      <span>
        <StatusField status={state} statusMap={DESTINATION_STATE_TO_STATUS}>
          <Typography variant="body2">{label}</Typography>
        </StatusField>
      </span>
    </Tooltip>
  );
};
