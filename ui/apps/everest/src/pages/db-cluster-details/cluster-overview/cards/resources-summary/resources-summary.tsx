// everest
// Copyright (C) 2023 Percona LLC
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

import { Box, Typography } from '@mui/material';
import OverviewSectionRow from '../../overview-section-row';
import { Messages } from '../../cluster-overview.messages';
import { ResourcesSummaryProps } from './resources-summary.types';

export const ResourcesSummary = ({ rows, synced }: ResourcesSummaryProps) => (
  <>
    <Box sx={{ display: 'flex', alignItems: 'center' }}>
      <Box sx={{ flex: '0 0 50%', minWidth: '90px' }} />
      <Typography variant="caption" color="text.secondary">
        {synced
          ? Messages.fields.limitRequestSyncedLegend
          : Messages.fields.limitRequestLegend}
      </Typography>
    </Box>
    {rows.map((row) => (
      <OverviewSectionRow
        key={row.label}
        label={row.label}
        dataTestId={row.dataTestId}
        content={
          row.request === undefined || synced
            ? row.limit
            : `${row.limit} / ${row.request}`
        }
      />
    ))}
  </>
);

export default ResourcesSummary;
