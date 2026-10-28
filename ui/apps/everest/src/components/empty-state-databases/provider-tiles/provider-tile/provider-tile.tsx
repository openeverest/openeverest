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

import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import { usePrefetchClusterInstancePresets } from 'hooks/api/cluster-instance-presets';
import type { ProviderTileProps } from './provider-tile.types';
import { ProviderTileCard } from '../provider-tile-card';

export const ProviderTile = ({
  provider,
  name,
  label,
  meta,
  showImport,
}: ProviderTileProps) => {
  const prefetchPresets = usePrefetchClusterInstancePresets();

  return (
    <ProviderTileCard
      testId={`provider-tile-${name}`}
      label={label}
      meta={meta}
      link={{
        to: '/databases/new',
        state: { selectedDbProvider: provider, showImport },
      }}
      footer={<ArrowForwardIcon fontSize="small" aria-hidden="true" />}
      onActivate={() => prefetchPresets(provider)}
    />
  );
};
