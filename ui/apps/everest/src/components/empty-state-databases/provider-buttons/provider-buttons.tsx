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

import { Box, Button } from '@mui/material';
import { Link } from 'react-router-dom';
import { resolveProviderDisplay } from 'components/provider-identity';
import type { ProviderPickerProps } from '../provider-picker.types';
import { buttonSx, rowSx } from './provider-buttons.constants';

// Fallback provider picker used when the plugin-hub is not installed: without
// catalog metadata there is nothing to fill a card, so a plain labelled button
// per provider is the honest representation.
export const ProviderButtons = ({
  providers,
  getProviderMeta,
  showImport = false,
}: ProviderPickerProps) => {
  return (
    <Box sx={rowSx}>
      {providers.map((provider) => {
        const { name, label } = resolveProviderDisplay(
          provider,
          getProviderMeta
        );
        return (
          <Button
            key={name}
            data-testid={`provider-tile-${name}`}
            component={Link}
            to="/databases/new"
            state={{ selectedDbProvider: provider, showImport }}
            variant="outlined"
            sx={buttonSx}
          >
            {label}
          </Button>
        );
      })}
    </Box>
  );
};
