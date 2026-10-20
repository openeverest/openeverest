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

import { DbEngineType } from 'shared-types/dbEngines.types';

// Providers are cluster-scoped (available everywhere), but a namespace can only
// host an instance if the matching engine operator is installed there. There is
// no contract field linking a provider to an engine type, so we derive it from
// the provider's name — the same naming convention the e2e suite relies on.
// Returns undefined when the provider can't be recognised, so callers can fall
// back to "any installed engine" instead of guessing.
export const providerToEngineType = (
  providerName?: string
): DbEngineType | undefined => {
  if (!providerName) {
    return undefined;
  }

  const name = providerName.toLowerCase();

  if (name.includes('xtradb') || name.includes('mysql')) {
    return DbEngineType.PXC;
  }
  if (name.includes('mongodb') || name.includes('psmdb')) {
    return DbEngineType.PSMDB;
  }
  if (name.includes('postgre')) {
    return DbEngineType.POSTGRESQL;
  }

  return undefined;
};
