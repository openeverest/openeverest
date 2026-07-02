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
import { ResourceGroupProps } from './resource-group.types';

export const ResourceGroup = ({
  title,
  rows,
  synced,
  dataTestId,
}: ResourceGroupProps) => {
  const rowsWithRequests = rows.filter((row) => row.request !== undefined);
  const limitOnlyRows = rows.filter((row) => row.request === undefined);
  const hasRowsWithRequests = rowsWithRequests.length > 0;

  const limitsText = rowsWithRequests
    .map((row) => `${row.label}: ${row.limit}`)
    .join('; ');
  const requestsText = rowsWithRequests
    .map((row) => `${row.label}: ${row.request}`)
    .join('; ');

  return (
    <Box data-testid={dataTestId} sx={{ mt: 0.5 }}>
      <Typography
        variant="caption"
        color="text.secondary"
        fontWeight={600}
        sx={{ display: 'block' }}
      >
        {`${title}:`}
      </Typography>
      {hasRowsWithRequests &&
        (synced ? (
          <Typography
            variant="caption"
            color="text.secondary"
            data-testid={`${dataTestId}-limits-line`}
            sx={{ display: 'block' }}
          >
            {'Limits '}
            <Tooltip title="Requests are synced with limits" arrow>
              <Box component="span" data-testid={`${dataTestId}-sync-icon`}>
                &amp;
              </Box>
            </Tooltip>
            {` Requests: ${limitsText}`}
          </Typography>
        ) : (
          <>
            <Typography
              variant="caption"
              color="text.secondary"
              data-testid={`${dataTestId}-limits-line`}
              sx={{ display: 'block' }}
            >
              {`Limits: ${limitsText}`}
            </Typography>
            <Typography
              variant="caption"
              color="text.secondary"
              data-testid={`${dataTestId}-requests-line`}
              sx={{ display: 'block' }}
            >
              {`Requests: ${requestsText}`}
            </Typography>
          </>
        ))}
      {limitOnlyRows.map((row) => (
        <Typography
          key={row.label}
          variant="caption"
          color="text.secondary"
          sx={{ display: 'block' }}
        >
          {`${row.label}: ${row.limit}`}
        </Typography>
      ))}
    </Box>
  );
};
