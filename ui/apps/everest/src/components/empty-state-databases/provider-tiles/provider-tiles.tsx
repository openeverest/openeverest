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

import { Box, Card, CardActionArea, CardContent } from '@mui/material';
import { Link } from 'react-router-dom';
import {
  ProviderIdentity,
  resolveProviderDisplay,
} from 'components/provider-identity';
import { useExtensionCatalog } from 'hooks/api/extension-catalog';
import type { ProviderTilesProps } from './provider-tiles.types';
import {
  cardActionAreaSx,
  cardContentSx,
  cardSx,
  gridSx,
} from './provider-tiles.constants';

export const ProviderTiles = ({
  providers,
  showImport = false,
}: ProviderTilesProps) => {
  const { getProviderMeta } = useExtensionCatalog();

  return (
    <Box sx={gridSx}>
      {providers.map((provider) => {
        const { name, label, meta } = resolveProviderDisplay(
          provider,
          getProviderMeta
        );
        return (
          <Card key={name} elevation={0} sx={cardSx}>
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
              <CardContent sx={cardContentSx}>
                <ProviderIdentity
                  label={label}
                  meta={meta}
                  showDescription
                  labelVariant="sectionHeading"
                />
              </CardContent>
            </CardActionArea>
          </Card>
        );
      })}
    </Box>
  );
};
