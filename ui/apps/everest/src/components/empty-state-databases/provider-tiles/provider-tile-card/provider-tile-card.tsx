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
  CardContent,
  Chip,
  Typography,
} from '@mui/material';
import type { ProviderTileCardProps } from './provider-tile-card.types';
import {
  bodyAnchorSx,
  cardActionAreaSx,
  cardContentSx,
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
import { TileActionArea } from '../tile-action-area';

export const ProviderTileCard = ({
  testId,
  label,
  meta,
  link,
  footer,
  variant = 'installed',
  onActivate,
}: ProviderTileCardProps) => {
  const [active, setActive] = useState(false);
  const isAvailable = variant === 'available';

  // Render meta-derived fields only when the hub provides them — never fabricate.
  // The provider version is intentionally not shown here: it's not meaningful
  // to users and only adds confusion (see issue #3051).
  const { description, maturity } = meta ?? {};
  const categories = meta?.categories?.slice(0, 2) ?? [];

  const activate = () => {
    setActive(true);
    onActivate?.();
  };

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
      <Box className="provider-tile-footer" sx={cardFooterEndSx}>
        {footer}
      </Box>
    </>
  );

  const content = (
    <>
      <Box className="provider-tile-header" sx={cardHeaderSx}>
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
          <Box
            className="provider-tile-expanded"
            sx={expandedSx}
            aria-hidden="true"
          >
            {renderTail(true)}
          </Box>
        )}
      </Box>
    </>
  );

  return (
    <Card
      // Paper ignores `elevation` for non-default variants such as `dashed`.
      elevation={0}
      variant={isAvailable ? 'dashed' : undefined}
      className={
        isAvailable ? 'provider-tile-available' : 'provider-tile-installed'
      }
      sx={cardSx}
      onMouseEnter={activate}
      onMouseLeave={() => setActive(false)}
      onFocus={activate}
      onBlur={() => setActive(false)}
    >
      <TileActionArea link={link} testId={testId} sx={cardActionAreaSx}>
        {content}
      </TileActionArea>
    </Card>
  );
};
