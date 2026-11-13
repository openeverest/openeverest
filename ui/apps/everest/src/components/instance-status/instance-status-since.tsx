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

import { Tooltip, Typography, TypographyProps } from '@mui/material';
import { format } from 'date-fns';
import { DATE_FORMAT } from 'consts';
import { useNow } from 'hooks/utils/useNow';
import { formatElapsed } from 'utils/instance-status';

interface InstanceStatusSinceProps extends Omit<TypographyProps, 'children'> {
  since: string;
  // "since 4 min", "started 4 min ago": the wording depends on what began.
  children: (elapsed: string) => string;
}

// A relative time that gives the exact one on hover.
export const InstanceStatusSince = ({
  since,
  children,
  ...typographyProps
}: InstanceStatusSinceProps) => {
  const now = useNow();

  return (
    <Tooltip title={format(new Date(since), DATE_FORMAT)} placement="top" arrow>
      <Typography
        component="span"
        variant="body2"
        color="text.secondary"
        noWrap
        {...typographyProps}
      >
        {children(formatElapsed(since, now))}
      </Typography>
    </Tooltip>
  );
};
