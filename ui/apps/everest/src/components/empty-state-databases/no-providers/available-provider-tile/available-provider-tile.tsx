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

import { Box } from '@mui/material';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import { ProviderTileCard } from 'components/empty-state-databases/provider-tiles/provider-tile-card';
import { Messages } from 'components/empty-state-databases/messages';
import { inlineIconLabelSx } from '../no-providers.constants';
import type { AvailableProviderTileProps } from './available-provider-tile.types';

export const AvailableProviderTile = ({
  meta,
  link,
}: AvailableProviderTileProps) => (
  <ProviderTileCard
    variant="available"
    testId={`available-provider-tile-${meta.name}`}
    label={meta.displayName}
    meta={meta}
    link={link}
    footer={
      <Box component="span" sx={inlineIconLabelSx}>
        {Messages.howToInstall}
        <ArrowForwardIcon fontSize="small" aria-hidden="true" />
      </Box>
    }
  />
);
