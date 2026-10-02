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
import { segmentCardFields } from './schema-driven-card.utils';
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
      <Stack
        sx={{
          gap: 3,
        }}
      >
        {card.fields.length > 0 ? (
          segmentCardFields(card.fields).map((segment, index) =>
            segment.kind === 'summary' && segment.field.summary ? (
              <OverviewSection
                key={`${card.key}:${segment.field.path}`}
                dataTestId={`${card.key}-${segment.field.path}`}
                // A card holding only this widget already shows its name.
                title={
                  segment.field.label === card.title
                    ? undefined
                    : segment.field.label
                }
                // Widget summaries are long; keep the card scannable until opened.
                collapsible={segment.field.label !== card.title}
                defaultExpanded={false}
                loading={loading}
              >
                <segment.field.summary.Component
                  item={segment.field.summary.item}
                  value={segment.field.summary.value}
                />
              </OverviewSection>
            ) : segment.kind === 'rows' ? (
              <OverviewSection
                key={`${card.key}:rows-${index}`}
                dataTestId={index === 0 ? card.key : `${card.key}-${index}`}
                loading={loading}
              >
                {segment.fields.map((field) => (
                  <OverviewSectionRow
                    key={`${card.key}:${field.path}`}
                    label={field.label}
                    content={field.value}
                  />
                ))}
              </OverviewSection>
            ) : null
          )
        ) : (
          <OverviewSection dataTestId={card.key} loading={loading}>
            <OverviewSectionRow label="Info" content="No data available" />
          </OverviewSection>
        )}
      </Stack>
    </OverviewCard>
  </Box>
);
export default SchemaDrivenCard;
