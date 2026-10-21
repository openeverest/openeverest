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

import { z } from 'zod';
import { Provider } from 'shared-types/api.types';

const importSourceSchema = z.object({
  namespace: z.string().min(1),
  backupClassName: z.string().min(1),
  // A restore's Backup source: the wizard seeds the instance the same way.
  dataSource: z.object({
    type: z.literal('Backup'),
    backupName: z.string().min(1),
    storageName: z.string().min(1),
  }),
});

export type ImportSource = z.infer<typeof importSourceSchema>;

export interface ImportNavigationRouteState {
  // The key the wizard already reads its provider from.
  selectedDbProvider: Provider;
  importSource: ImportSource;
}

export interface ImportNavigation {
  providerName: string;
  importSource: ImportSource;
}

const importNavigationRouteStateSchema = z.object({
  selectedDbProvider: z.object({
    metadata: z.object({ name: z.string().min(1) }),
  }),
  importSource: importSourceSchema,
});

export const buildImportNavigationRouteState = (
  provider: Provider,
  importSource: ImportSource
): ImportNavigationRouteState => ({
  selectedDbProvider: provider,
  importSource,
});

// Router state is untyped, so it is validated before it is trusted.
export const parseImportNavigationRouteState = (
  state: unknown
): ImportNavigation | undefined => {
  const parsed = importNavigationRouteStateSchema.safeParse(state);
  return parsed.success
    ? {
        providerName: parsed.data.selectedDbProvider.metadata.name,
        importSource: parsed.data.importSource,
      }
    : undefined;
};
