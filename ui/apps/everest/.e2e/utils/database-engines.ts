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
import { dbTypeToDbEngine } from '@percona/utils';
import { DbType } from '@percona/types';
import { getTokenFromLocalStorage } from './localStorage';

export const getEnginesList = async (
  token: string,
  namespace: string,
  request: APIRequestContext
) => {
  const enginesList = await request.get(
    `/v1/namespaces/${namespace}/database-engines`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  expect(enginesList.ok()).toBeTruthy();
  return (await enginesList.json()).items;
};
export const getEnginesVersions = async (
  token: string,
  namespace: string,
  request: APIRequestContext
) => {
  const engineVersions = {
    pxc: [],
    psmdb: [],
    postgresql: [],
  };

  const engines = await getEnginesList(token, namespace, request);
  engines.forEach((engine) => {
    const { type } = engine.spec;

    if (engine.status?.status === 'installed') {
      engineVersions[type].push(
        ...Object.keys(engine.status.availableVersions.engine)
      );
    }
  });

  return engineVersions;
};

export const getLastAvailableEngineVersion = async (
  token: string,
  namespace: string,
  request: APIRequestContext,
  dbType: string
): Promise<string> => {
  const dbEngines = await getEnginesVersions(token, namespace, request);
  const dbEngineType = dbTypeToDbEngine(dbType as DbType);
  const dbTypeVersions = dbEngines[dbEngineType] || [];

  return dbTypeVersions[dbTypeVersions.length - 1];
};

export const getEnginesLatestRecommendedVersions = async (
  namespace: string,
  request: APIRequestContext
) => {
  let latestRecommendedVersions = {
    pxc: '',
    psmdb: '',
    postgresql: '',
  };
  const token = await getTokenFromLocalStorage();
  const engines = await getEnginesList(token, namespace, request);
  engines.forEach((engine) => {
    const { type } = engine.spec;

    if (engine.status?.status === 'installed') {
      Object.entries(engine.status.availableVersions.engine).forEach(
        ([key, value]: any) => {
          if (value.status === 'recommended') {
            latestRecommendedVersions[type] = key;
          }
        }
      );
    }
  });
  return latestRecommendedVersions;
};
