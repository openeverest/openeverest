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

import { CI_USER_STORAGE_STATE_FILE } from '@e2e/constants';
import { PlaywrightTestProject } from '@playwright/test';

export const dbClusterProject: PlaywrightTestProject[] = [
  {
    name: 'pr:db-cluster',
    testMatch: /.^/,
    dependencies: [
      'pr:db-cluster:db-overview',
      'pr:db-cluster:db-list',
      'pr:db-cluster:db-wizard',
    ],
  },
  {
    name: 'pr:db-cluster:db-overview:setup',
    testDir: './pr/db-cluster',
    testMatch: /db-cluster-overview\.setup\.ts/,
    dependencies: ['global:auth:ci:setup'],
    teardown: 'pr:db-cluster:db-overview:teardown',
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster:db-overview:teardown',
    testDir: './pr/db-cluster',
    testMatch: /db-cluster-overview\.teardown\.ts/,
  },
  {
    name: 'pr:db-cluster:db-overview',
    testDir: './pr/db-cluster',
    testMatch: /db-cluster-overview\.e2e\.ts/,
    dependencies: ['pr:db-cluster:db-overview:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster:db-list',
    testDir: './pr/db-cluster',
    testMatch: /db-clusters-list\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster:db-wizard',
    testMatch: /.^/,
    dependencies: [
      'pr:db-cluster:db-wizard:create',
      'pr:db-cluster:db-wizard:create:sharding:psmdb',
      'pr:db-cluster:db-wizard:errors',
    ],
  },
  {
    name: 'pr:db-cluster:db-wizard:create',
    testDir: './pr/db-cluster/db-wizard/create-db-cluster',
    testMatch: /create-db-cluster\.e2e\.ts/,
    dependencies: ['global:backup-storage:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster:db-wizard:errors',
    testDir: './pr/db-cluster/db-wizard/create-db-cluster',
    testMatch: /errors-handling\.e2e\.ts/,
    dependencies: ['global:backup-storage:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:db-cluster:db-wizard:create:sharding:psmdb',
    testDir: './pr/db-cluster/db-wizard/create-db-cluster',
    testMatch: /sharding\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
];
