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

import { useMemo } from 'react';
import { Box, Button, Typography } from '@mui/material';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import { PLUGIN_HUB_NAME } from 'api/extension-catalog';
import { usePlugins } from 'contexts/plugins';
import { Messages } from '../messages';
import { gridSx } from 'components/empty-state-databases/provider-tiles/provider-tiles.constants';
import {
  EXTENSION_HUB_WEBSITE_URL,
  HUB_PROVIDERS_SUB_PATH,
  SPOTLIGHT_LIMIT,
  SPOTLIGHT_PREFERRED_PROVIDERS,
} from './no-providers.constants';
import { buildHubLink, pickSpotlightProviders } from './no-providers.utils';
import { AvailableProviderTile } from './available-provider-tile';
import { BrowseHubTile } from './browse-hub-tile';
import type { NoProvidersProps } from './no-providers.types';

// Empty-install state: previews the provider grid with a few installable
// providers from the hub catalog, so installing one simply turns its slot into
// a real tile (providers are polled). Without the catalog there is nothing
// honest to preview, so it falls back to a single link to the Extension Hub.
export const NoProviders = ({ catalogProviders }: NoProvidersProps) => {
  const { plugins } = usePlugins();
  const hubUiAvailable = plugins.some(
    (plugin) => plugin.name === PLUGIN_HUB_NAME
  );
  const spotlight = useMemo(
    () =>
      pickSpotlightProviders(
        catalogProviders,
        SPOTLIGHT_LIMIT,
        SPOTLIGHT_PREFERRED_PROVIDERS
      ),
    [catalogProviders]
  );

  return (
    <>
      <Typography variant="h5" sx={{ mb: 0.5 }}>
        {Messages.createFirstResource}
      </Typography>
      <Typography color="text.secondary" textAlign="center" sx={{ mb: 2 }}>
        {Messages.noProvidersBody}
      </Typography>
      {spotlight.length > 0 ? (
        <Box sx={gridSx}>
          {spotlight.map((meta) => (
            <AvailableProviderTile
              key={meta.name}
              meta={meta}
              link={buildHubLink(
                hubUiAvailable,
                `${HUB_PROVIDERS_SUB_PATH}/${meta.name}`
              )}
            />
          ))}
          <BrowseHubTile
            moreCount={catalogProviders.length - spotlight.length}
            link={buildHubLink(hubUiAvailable, HUB_PROVIDERS_SUB_PATH)}
          />
        </Box>
      ) : (
        <Button
          variant="outlined"
          href={EXTENSION_HUB_WEBSITE_URL}
          target="_blank"
          rel="noopener noreferrer"
          endIcon={<OpenInNewIcon />}
          data-testid="browse-extension-hub-button"
        >
          {Messages.browseHub}
        </Button>
      )}
    </>
  );
};
