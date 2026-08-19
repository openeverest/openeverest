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
import { resolveProviderDisplay } from 'components/provider-identity';
import type { ProviderPickerProps } from '../provider-picker.types';
import { gridSx } from './provider-tiles.constants';
import { ProviderTile } from './provider-tile';

export const ProviderTiles = ({
  providers,
  getProviderMeta,
  showImport = false,
}: ProviderPickerProps) => {
  return (
    <Box sx={gridSx}>
      {providers.map((provider) => {
        const { name, label, meta } = resolveProviderDisplay(
          provider,
          getProviderMeta
        );
        return (
          <ProviderTile
            key={name}
            provider={provider}
            name={name}
            label={label}
            meta={meta}
            showImport={showImport}
          />
        );
      })}
    </Box>
  );
};
