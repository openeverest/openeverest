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
import { format, formatDuration, intervalToDuration, isValid } from 'date-fns';
import { DATE_FORMAT } from 'consts';

export type ComponentAgeProps = {
  date: string;
  render?: (date?: string) => React.ReactNode;
  typographyProps?: TypographyProps;
};

const ComponentAge = ({ date, render, typographyProps }: ComponentAgeProps) => {
  const dateObj = new Date(date);

  const formattedDate = isValid(dateObj)
    ? formatDuration(
        intervalToDuration({
          start: date,
          end: Date.now(),
        }),
        {
          format: ['years', 'months', 'days', 'hours', 'minutes'],
        }
      )
    : '';

  const resultStr = formattedDate ? `${formattedDate} ago` : '';

  return (
    <Tooltip
      title={isValid(dateObj) ? `Started at ${format(date, DATE_FORMAT)}` : ''}
      placement="right"
      arrow
    >
      <Typography
        variant="caption"
        {...typographyProps}
        sx={[
          {
            color: 'text.secondary',
          },
          ...(typographyProps?.sx
            ? Array.isArray(typographyProps.sx)
              ? typographyProps.sx
              : [typographyProps.sx]
            : []),
        ]}
      >
        {render ? render(resultStr) : resultStr}
      </Typography>
    </Tooltip>
  );
};

export default ComponentAge;
