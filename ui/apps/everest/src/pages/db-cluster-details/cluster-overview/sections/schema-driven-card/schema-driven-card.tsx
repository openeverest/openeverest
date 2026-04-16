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

import { Box, IconButton, Stack } from '@mui/material';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import { DatabaseIcon, OverviewCard } from '@percona/ui-lib';
import type { SchemaDrivenCardProps } from './schema-driven-card.types';
import OverviewSectionRow from '../../overview-section-row';
import OverviewSection from '../../overview-section';

const SchemaDrivenCard = ({
  card,
  loading,
  editable,
  onEdit,
}: SchemaDrivenCardProps) => (
  <Box>
    <OverviewCard
      dataTestId={`${card.key}-details`}
      sx={{ width: '100%' }}
      cardHeaderProps={{
        title: card.title,
        avatar: <DatabaseIcon />,
        action: editable ? (
          <IconButton
            data-testid={`${card.key}-edit-button`}
            size="small"
            onClick={onEdit}
            aria-label={`Edit ${card.title}`}
          >
            <EditOutlinedIcon fontSize="small" />
          </IconButton>
        ) : undefined,
      }}
    >
      <Stack gap={3}>
        <OverviewSection dataTestId={card.key} loading={loading}>
          {card.fields.length > 0 ? (
            card.fields.map(({ label, path, value }) => (
              <OverviewSectionRow
                key={`${card.key}:${path}`}
                label={label}
                content={value}
              />
            ))
          ) : (
            <OverviewSectionRow label="Info" content="No data available" />
          )}
        </OverviewSection>
      </Stack>
    </OverviewCard>
  </Box>
);

export default SchemaDrivenCard;
