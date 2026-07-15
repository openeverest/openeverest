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
import { Messages } from './messages';
import { ProviderTiles } from './provider-tiles';

type EmptyStateDatabasesProps = {
  hasCreatePermission?: boolean;
};

// `hasCreatePermission` defaults to `false`: RBAC decisions must be made by
// the caller (which owns the permission hook) — never assumed by presentation
// components. Rendering create affordances without a positive grant would
// leak actions to users who cannot perform them.
const EmptyStateDatabases = ({
  hasCreatePermission = false,
}: EmptyStateDatabasesProps) => {
  const { data: providers = [] } = useProviders();

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
                rel="noopener"
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
          <ProviderTiles providers={providers} />
        </>
      }
    />
  );
};

export default EmptyStateDatabases;
