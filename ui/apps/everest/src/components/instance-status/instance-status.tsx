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

import { Box, Tooltip, Typography } from '@mui/material';
import { PendingIcon } from '@percona/ui-lib';
import StatusField from 'components/status-field';
import { useNow } from 'hooks/utils/useNow';
import { InstanceStatusDisplay } from 'shared-types/instance.types';
import {
  formatElapsed,
  getInstanceStatusExplanation,
} from 'utils/instance-status';
import { TONE_TO_BASE_STATUS } from './instance-status.constants';
import { Messages } from './instance-status.messages';

interface InstanceStatusProps {
  status: InstanceStatusDisplay;
  // The details page says whether the database is up in the same line, and
  // spells out there what a list leaves to the tooltip.
  showAvailability?: boolean;
  dataTestId?: string;
}

// The status word with the icon of its tone. In a list, hovering explains it.
export const InstanceStatus = ({
  status,
  showAvailability = false,
  dataTestId,
}: InstanceStatusProps) => {
  const now = useNow();
  const { tone, label, sentence, since } = status;
  const message = getInstanceStatusExplanation(status);
  const explained = !showAvailability && (!!message || !!since);

  return (
    <Tooltip
      title={
        explained && (
          <>
            {message && <Box>{message}</Box>}
            {since && (
              <Box sx={{ opacity: 0.8 }}>
                {Messages.since(formatElapsed(since, now))}
              </Box>
            )}
          </>
        )
      }
      placement="right"
      arrow
    >
      <Box component="span" sx={{ display: 'inline-flex', minWidth: 0 }}>
        <StatusField
          dataTestId={dataTestId}
          status={tone}
          statusMap={TONE_TO_BASE_STATUS}
          defaultIcon={PendingIcon}
          stackProps={{ sx: { minWidth: 0 } }}
        >
          <Typography
            component="span"
            variant="inherit"
            sx={{ whiteSpace: 'nowrap' }}
          >
            {showAvailability ? sentence : label}
          </Typography>
        </StatusField>
      </Box>
    </Tooltip>
  );
};
