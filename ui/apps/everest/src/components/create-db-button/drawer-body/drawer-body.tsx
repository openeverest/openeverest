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
import { Link } from 'react-router-dom';
import {
  ProviderIdentity,
  resolveProviderDisplay,
} from 'components/provider-identity';
import { useExtensionCatalog } from 'hooks/api/extension-catalog';
import type { DrawerBodyProps } from './drawer-body.types';
import { listSx, rowSx } from './drawer-body.constants';

export const DrawerBody = ({
  providers,
  createFromImport,
  onClose,
}: DrawerBodyProps) => {
  const { getProviderMeta } = useExtensionCatalog();
  const testIdPrefix = createFromImport ? 'import' : 'add';

  return (
    <Box role="list" sx={listSx}>
      {providers.map((provider) => {
        const { name, label, meta } = resolveProviderDisplay(
          provider,
          getProviderMeta
        );
        return (
          <Box role="listitem" key={name}>
            <Box
              data-testid={`${testIdPrefix}-db-cluster-button-${name}`}
              component={Link}
              to="/databases/new"
              state={{
                selectedDbProvider: provider,
                showImport: createFromImport,
              }}
              onClick={onClose}
              sx={rowSx}
            >
              <ProviderIdentity label={label} meta={meta} ellipsis />
            </Box>
          </Box>
        );
      })}
    </Box>
  );
};
