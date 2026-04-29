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

import type React from 'react';

export interface ProviderParams {
  namespace: string;
  cluster: string;
  // TODO: Add currentValue?: string for filtering out unavailable options at edit time
}

export interface ProviderOptions {
  options: Array<{ label: string; value: string }>;
  isLoading: boolean;
  error: unknown;
  isEmpty: boolean;
  rawData?: unknown;
}

export interface ProviderRegistryEntry {
  useOptions: (params: ProviderParams) => ProviderOptions;
  description: string;
  emptyStateFallback?: {
    component: React.ComponentType<{ namespace: string; cluster: string }>;
  };
  // TODO RBAC: Add permission config (resource, action) so DataSourceField can check
  // useRBACPermissions and conditionally hide "create new" CTA or disable the field.
  // TODO: Add lifecycle hooks — beforeFetch, afterFetch, onError
  // TODO: Add optional `validate` callback for dev-time schema validation
}
