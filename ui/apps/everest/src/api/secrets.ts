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

import { api } from 'api/api';
import {
  Secret,
  SecretDataPatch,
  SecretList,
  SecretListFilter,
} from 'shared-types/api.types';

export const getSecretsFn = async (
  cluster: string,
  namespace: string,
  filter: SecretListFilter
): Promise<Secret[]> => {
  const response = await api.get<SecretList>(
    `clusters/${cluster}/namespaces/${namespace}/secrets`,
    { params: filter }
  );
  return response.data.items;
};

export const createSecretFn = async (
  cluster: string,
  namespace: string,
  secret: Secret
): Promise<Secret> => {
  const response = await api.post<Secret>(
    `clusters/${cluster}/namespaces/${namespace}/secrets`,
    secret,
    // A taken name is shown on the form's name field.
    { disableNotifications: (e) => e.status === 409 }
  );
  return response.data;
};

// Metadata only: the API never returns Secret data.
export const getSecretFn = async (
  cluster: string,
  namespace: string,
  name: string,
  disableNotifications = false
): Promise<Secret> => {
  const response = await api.get<Secret>(
    `clusters/${cluster}/namespaces/${namespace}/secrets/${name}`,
    { disableNotifications }
  );
  return response.data;
};

// With dryRun the API checks the patch, the definition and permissions, and
// stores nothing.
export const patchSecretFn = async (
  cluster: string,
  namespace: string,
  name: string,
  patch: SecretDataPatch,
  { dryRun = false }: { dryRun?: boolean } = {}
): Promise<Secret> => {
  const response = await api.patch<Secret>(
    `clusters/${cluster}/namespaces/${namespace}/secrets/${name}`,
    patch,
    {
      headers: { 'Content-Type': 'application/merge-patch+json' },
      ...(dryRun && { params: { dryRun: 'All' } }),
    }
  );
  return response.data;
};

export const deleteSecretFn = async (
  cluster: string,
  namespace: string,
  name: string
): Promise<void> => {
  await api.delete(
    `clusters/${cluster}/namespaces/${namespace}/secrets/${name}`
  );
};
