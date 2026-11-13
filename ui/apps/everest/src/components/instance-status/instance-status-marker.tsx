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

import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { Box, Chip, Tooltip } from '@mui/material';
import { InstanceConditionEntry } from 'utils/instance-conditions';
import { Messages } from './instance-status.messages';

interface InstanceStatusMarkerProps {
  entries: InstanceConditionEntry[];
  dataTestId?: string;
}

// Points at what deserves a look without being part of the status: a notice,
// or a failing check the database is Ready without.
export const InstanceStatusMarker = ({
  entries,
  dataTestId,
}: InstanceStatusMarkerProps) => {
  if (entries.length === 0) {
    return null;
  }

  return (
    <Tooltip
      title={entries.map(({ condition, title }) => (
        <Box key={condition.type}>
          <b>{title}</b>
          {condition.message && `: ${condition.message}`}
        </Box>
      ))}
      placement="left"
      arrow
    >
      <Chip
        size="small"
        variant="outlined"
        icon={<InfoOutlinedIcon />}
        label={Messages.notices(entries.length)}
        data-testid={`${dataTestId ? `${dataTestId}-` : ''}notices`}
      />
    </Tooltip>
  );
};
