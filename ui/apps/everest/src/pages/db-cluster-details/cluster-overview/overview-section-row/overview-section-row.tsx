// everest
// Copyright (C) 2023 Percona LLC
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

import { Grid, Typography } from '@mui/material';
import { OverviewSectionRowProps } from './overview-section-row.types';
import { kebabize } from '@percona/utils';
import { OVERVIEW_SECTION_ROW_MIN_WIDTH } from './overview-section-row.constants';

export const OverviewSectionRow = ({
  label,
  labelProps,
  content,
  dataTestId,
}: OverviewSectionRowProps) => (
  <Grid
    container
    data-testid={`${dataTestId ?? kebabize(label)}-overview-section-row`}
    position="relative"
    alignItems="center"
  >
    <Grid
      item
      xs={6}
      minWidth={OVERVIEW_SECTION_ROW_MIN_WIDTH}
      {...labelProps}
    >
      <Typography variant="body2" sx={{ fontWeight: '700' }}>
        {label}
      </Typography>
    </Grid>
    <Grid item>
      {typeof content === 'string' ? (
        <Typography variant="body2">{content}</Typography>
      ) : (
        content
      )}
    </Grid>
  </Grid>
);

export default OverviewSectionRow;
