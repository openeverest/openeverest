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

import { Link, Typography } from '@mui/material';
import EmptyState from 'components/empty-state';
import { useProviders } from 'hooks/api/providers/useProviders';
import { useExtensionCatalog } from 'hooks/api/extension-catalog';
import { Messages } from './messages';
import { ProviderTiles } from './provider-tiles';
import { ProviderButtons } from './provider-buttons';
import type { EmptyStateDatabasesProps } from './empty-state-databases.types';

const EmptyStateDatabases = ({
  hasCreatePermission = false,
}: EmptyStateDatabasesProps) => {
  const { data: providers = [] } = useProviders();
  const { getProviderMeta, providerMeta } = useExtensionCatalog();

  if (!hasCreatePermission) {
    return (
      <EmptyState
        showCreationButton={false}
        contentSlot={
          <>
            <Typography>{Messages.noPermissions}</Typography>
            <Typography>
              Click{' '}
              <Link
                target="_blank"
                rel="noopener noreferrer"
                href="https://openeverest.io/documentation/current/administer/rbac.html"
              >
                here
              </Link>{' '}
              to learn how to get permissions.
            </Typography>
          </>
        }
      />
    );
  }

  if (providers.length === 0) {
    return (
      <EmptyState
        showCreationButton={false}
        contentSlot={
          <>
            <Typography variant="h6">{Messages.noProvidersTitle}</Typography>
            <Typography textAlign="center">
              {Messages.noProvidersBody}
            </Typography>
          </>
        }
      />
    );
  }

  return (
    <EmptyState
      showCreationButton={false}
      contentSlot={
        <>
          <Typography variant="h5" sx={{ mb: 0.5 }}>
            {Messages.createFirstResource}
          </Typography>
          <Typography color="text.secondary" sx={{ mb: 2 }}>
            {Messages.pickProvider}
          </Typography>
          {/* With the plugin-hub catalog we can render rich tiles; without it
              there is no metadata to fill a card, so fall back to plain
              provider buttons. */}
          {providerMeta.size > 0 ? (
            <ProviderTiles
              providers={providers}
              getProviderMeta={getProviderMeta}
            />
          ) : (
            <ProviderButtons
              providers={providers}
              getProviderMeta={getProviderMeta}
            />
          )}
        </>
      }
    />
  );
};

export default EmptyStateDatabases;
