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

import { ReactNode } from 'react';
import HistoryIcon from '@mui/icons-material/History';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { Box, Stack, Typography } from '@mui/material';
import {
  PendingIcon,
  StatusIconProps,
  SuccessIcon,
  WarningIcon,
} from '@percona/ui-lib';
import { InstanceCondition } from 'shared-types/instance.types';

const ICON_SIZE_PX = 16;

// How a check reads: fine, in progress, or a problem. A failing check warns
// rather than alarms; whether someone has to act is the summary's to say.
const CHECK_ICONS: Record<
  InstanceCondition['status'],
  (props: StatusIconProps) => ReactNode
> = {
  True: SuccessIcon,
  Unknown: PendingIcon,
  False: WarningIcon,
};

export const CheckIcon = ({
  status,
}: {
  status: InstanceCondition['status'];
}) => {
  const Icon = CHECK_ICONS[status];
  return <Icon size="small" />;
};

const plainIconSx = { fontSize: ICON_SIZE_PX, color: 'text.secondary' };

// Neither good nor bad: something to know about.
export const NoticeIcon = () => <InfoOutlinedIcon sx={plainIconSx} />;

// An operation that is over. How it ended is in its reason, which is not
// this UI's to interpret.
export const FinishedIcon = () => <HistoryIcon sx={plainIconSx} />;

interface StatusRowProps {
  label: string;
  children: ReactNode;
  dataTestId: string;
}

// One section of the card: what it is about, then what there is to say.
export const StatusRow = ({ label, children, dataTestId }: StatusRowProps) => (
  <Box
    data-testid={`status-${dataTestId}`}
    sx={{
      display: 'grid',
      gridTemplateColumns: { xs: '1fr', sm: '170px minmax(0, 1fr)' },
      columnGap: 2,
      rowGap: 0.5,
      alignItems: 'start',
    }}
  >
    <Typography variant="body2" sx={{ fontWeight: 700 }}>
      {label}
    </Typography>
    <Stack sx={{ gap: 0.75, minWidth: 0 }}>{children}</Stack>
  </Box>
);

interface StatusLineProps {
  icon?: ReactNode;
  title: string;
  // Stands out from the message: the name of what is wrong.
  reason?: string;
  message?: string;
  // Sits right after the title: something to click.
  action?: ReactNode;
  // Sits at the end of the line: a time.
  trailing?: ReactNode;
  dataTestId?: string;
}

// One condition in plain words: what it is, then what it says.
export const StatusLine = ({
  icon,
  title,
  reason,
  message,
  action,
  trailing,
  dataTestId,
}: StatusLineProps) => (
  <Box
    data-testid={dataTestId}
    sx={{
      display: 'grid',
      gridTemplateColumns: { xs: '1fr', md: '250px minmax(0, 1fr)' },
      columnGap: 2,
      alignItems: 'start',
    }}
  >
    <Stack direction="row" sx={{ gap: 1, alignItems: 'center' }}>
      {icon}
      <Typography variant="body2">{title}</Typography>
      {action}
    </Stack>
    <Box
      sx={{
        display: 'flex',
        flexWrap: 'wrap',
        columnGap: 1.5,
        alignItems: 'baseline',
        minWidth: 0,
      }}
    >
      {reason && <Typography variant="body2">{reason}</Typography>}
      {message && (
        <Typography
          variant="body2"
          color="text.secondary"
          sx={{ wordBreak: 'break-word', whiteSpace: 'pre-line' }}
        >
          {message}
        </Typography>
      )}
      {trailing}
    </Box>
  </Box>
);

export const StatusEmpty = ({ children }: { children: string }) => (
  <Typography variant="body2" color="text.secondary">
    {children}
  </Typography>
);
