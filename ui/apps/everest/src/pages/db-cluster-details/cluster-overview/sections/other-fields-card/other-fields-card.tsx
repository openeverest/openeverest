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

import { Box, Stack } from '@mui/material';
import { DatabaseIcon, OverviewCard } from '@percona/ui-lib';

import type { OtherFieldsCardProps } from './other-fields-card.types';
import OverviewSection from '../../overview-section';
import OverviewSectionRow from '../../overview-section-row';

const OtherFieldsCard = ({ fields, loading }: OtherFieldsCardProps) => (
  <Box>
    <OverviewCard
      dataTestId="other-details"
      sx={{ width: '100%' }}
      cardHeaderProps={{
        title: 'Other',
        avatar: <DatabaseIcon />,
      }}
    >
      <Stack gap={3}>
        <OverviewSection dataTestId="other" loading={loading}>
          {fields.map(({ label, value }) => (
            <OverviewSectionRow key={label} label={label} content={value} />
          ))}
        </OverviewSection>
      </Stack>
    </OverviewCard>
  </Box>
);

export default OtherFieldsCard;
