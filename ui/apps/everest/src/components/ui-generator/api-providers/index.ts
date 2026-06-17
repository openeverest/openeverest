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

// Side-effect import: registers all built-in providers (monitoringConfigs, etc.)
import './providers';

export { providerRegistry, useProviderOptions } from './registry';
export { DataSourceField, hasDataSource } from './data-source-field';
export { DataSourcePrefetcher } from './data-source-prefetcher';
export type {
  DataSourceFieldProps,
  ComponentWithDataSource,
} from './data-source-field';
export type {
  ProviderParams,
  ProviderOptions,
  ProviderRegistryEntry,
  EmptyStateFallback,
  EmptyStateFallbackProps,
} from './types';
