// everest
// Copyright (C) 2023 Percona LLC
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

import { APIRequestContext, expect } from '@playwright/test';

type StorageClassItem = {
  metadata?: {
    name?: string;
    creationTimestamp?: string;
    annotations?: Record<string, string>;
  };
};

const isDefault = ({ metadata }: StorageClassItem) =>
  metadata?.annotations?.['storageclass.kubernetes.io/is-default-class'] ===
  'true';

const createdAt = ({ metadata }: StorageClassItem) =>
  Date.parse(metadata?.creationTimestamp ?? '') || 0;

// Returns the storage class names the way the removed /cluster-info did:
// default class first (newest default first), so callers can pick [0].
export const getClusterDetailedInfo = async (
  token: string,
  request: APIRequestContext
): Promise<{ storageClassNames: string[] }> => {
  const response = await request.get('/v1/clusters/main/storage-classes', {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  expect(response.ok()).toBeTruthy();
  const { items = [] }: { items?: StorageClassItem[] } = await response.json();

  const storageClassNames = [...items]
    .sort((a, b) => {
      if (isDefault(a) !== isDefault(b)) {
        return isDefault(a) ? -1 : 1;
      }
      return isDefault(a) ? createdAt(b) - createdAt(a) : 0;
    })
    .flatMap(({ metadata }) => (metadata?.name ? [metadata.name] : []));

  return { storageClassNames };
};
