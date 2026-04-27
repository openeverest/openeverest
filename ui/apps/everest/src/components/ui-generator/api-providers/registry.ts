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

import type {
  ProviderRegistryEntry,
  ProviderOptions,
  ProviderParams,
} from './types';

class ApiProviderRegistry {
  private entries = new Map<string, ProviderRegistryEntry>();

  register(key: string, entry: ProviderRegistryEntry): void {
    if (this.entries.has(key) && !import.meta.hot) {
      throw new Error(
        `ApiProviderRegistry: provider "${key}" is already registered.`
      );
    }
    // Vite HMR re-runs providers.ts while this singleton survives — allow overwrite.
    this.entries.set(key, entry);
  }

  get(key: string): ProviderRegistryEntry | undefined {
    return this.entries.get(key);
  }

  has(key: string): boolean {
    return this.entries.has(key);
  }

  getAll(): Map<string, ProviderRegistryEntry> {
    return new Map(this.entries);
  }

  getAvailableKeys(): string[] {
    return Array.from(this.entries.keys());
  }
}

// Singleton registry instance
export const providerRegistry = new ApiProviderRegistry();

export const useProviderOptions = (
  providerKey: string,
  params: ProviderParams
): ProviderOptions => {
  const entry = providerRegistry.get(providerKey);

  if (!entry) {
    const available = providerRegistry.getAvailableKeys().join(', ');
    throw new Error(
      `Unknown API provider "${providerKey}". Available providers: ${available}`
    );
  }

  return entry.useOptions(params);
};
