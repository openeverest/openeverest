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

import { useState } from 'react';
import {
  Avatar,
  Box,
  Card,
  CardActionArea,
  CardContent,
  Chip,
  Typography,
} from '@mui/material';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import { Link } from 'react-router-dom';
import type { ProviderTileProps } from './provider-tile.types';
import {
  bodyAnchorSx,
  cardActionAreaSx,
  cardContentSx,
  cardFooterSx,
  cardFooterEndSx,
  cardHeaderSx,
  cardSx,
  categoriesRowSx,
  descriptionFullSx,
  descriptionSx,
  expandedSx,
  headerAvatarSx,
  headerChipSx,
} from '../provider-tiles.constants';

export const ProviderTile = ({
  provider,
  name,
  label,
  meta,
  showImport,
}: ProviderTileProps) => {
  const [active, setActive] = useState(false);

  // Render meta-derived fields only when the hub provides them — never fabricate.
  const { description, maturity, version } = meta ?? {};
  const categories = meta?.categories?.slice(0, 2) ?? [];

  const renderTail = (expanded: boolean) => (
    <>
      <CardContent sx={cardContentSx}>
        <Typography variant="sectionHeading" noWrap>
          {label}
        </Typography>
        {description && (
          <Typography
            variant="body2"
            sx={expanded ? descriptionFullSx : descriptionSx}
          >
            {description}
          </Typography>
        )}
        {categories.length > 0 && (
          <Box sx={categoriesRowSx}>
            {categories.map((category) => (
              <Chip key={category} label={category} size="xs" />
            ))}
          </Box>
        )}
      </CardContent>
      <Box sx={version ? cardFooterSx : cardFooterEndSx}>
        {version && <Typography variant="caption">{version}</Typography>}
        <ArrowForwardIcon fontSize="small" aria-hidden="true" />
      </Box>
    </>
  );

  return (
    <Card
      elevation={0}
      sx={cardSx}
      onMouseEnter={() => setActive(true)}
      onMouseLeave={() => setActive(false)}
      onFocus={() => setActive(true)}
      onBlur={() => setActive(false)}
    >
      <CardActionArea
        data-testid={`provider-tile-${name}`}
        component={Link}
        to="/databases/new"
        state={{
          selectedDbProvider: provider,
          showImport,
        }}
        sx={cardActionAreaSx}
      >
        <Box sx={cardHeaderSx}>
          <Avatar
            src={meta?.icon}
            variant="rounded"
            sx={{
              ...headerAvatarSx,
              bgcolor: meta?.icon ? 'transparent' : 'action.selected',
            }}
          >
            {label.charAt(0)}
          </Avatar>
          {maturity && (
            <Chip
              label={maturity}
              size="xs"
              variant="outlined"
              sx={headerChipSx}
            />
          )}
        </Box>
        <Box sx={bodyAnchorSx}>
          {renderTail(false)}
          {/* Extension overlay, revealed on hover/focus. aria-hidden: the real
              tail already conveys the content. */}
          {active && (
            <Box sx={expandedSx} aria-hidden="true">
              {renderTail(true)}
            </Box>
          )}
        </Box>
      </CardActionArea>
    </Card>
  );
};
