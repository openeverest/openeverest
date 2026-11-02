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

import { Provider } from 'shared-types/api.types';
import { BackupSourceMetadata } from 'shared-types/backups.types';

export type ImportPrefill = {
  topology?: string;
  version?: string;
};

// Only an exact engine version match is safe to preselect; anything else is the user's pick.
export const getImportPrefill = (
  sourceMetadata: BackupSourceMetadata | undefined,
  provider: Provider | undefined,
  topologies: string[]
): ImportPrefill => {
  const { topology, engineVersion } = sourceMetadata ?? {};
  return {
    topology: topology && topologies.includes(topology) ? topology : undefined,
    version: engineVersion
      ? provider?.spec.versions?.find(
          (bundle) => bundle.components?.engine === engineVersion
        )?.name
      : undefined,
  };
};
