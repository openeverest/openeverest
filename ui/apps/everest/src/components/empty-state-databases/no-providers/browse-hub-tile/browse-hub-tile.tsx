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

import { Card, Typography } from '@mui/material';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import { cardSx } from '../../provider-tiles/provider-tiles.constants';
import { TileActionArea } from '../../provider-tiles/tile-action-area';
import { Messages } from '../../messages';
import {
  browseTileActionSx,
  browseTileContentSx,
} from '../no-providers.constants';
import type { BrowseHubTileProps } from './browse-hub-tile.types';

export const BrowseHubTile = ({ moreCount, link }: BrowseHubTileProps) => (
  <Card variant="dashed" sx={cardSx}>
    <TileActionArea
      link={link}
      testId="browse-extension-hub-tile"
      sx={browseTileContentSx}
    >
      {moreCount > 0 && (
        <Typography variant="sectionHeading">
          {Messages.moreProviders(moreCount)}
        </Typography>
      )}
      <Typography
        component="span"
        variant="body2"
        color="primary"
        sx={browseTileActionSx}
      >
        {Messages.browseHub}
        <ArrowForwardIcon fontSize="small" aria-hidden="true" />
      </Typography>
    </TileActionArea>
  </Card>
);
