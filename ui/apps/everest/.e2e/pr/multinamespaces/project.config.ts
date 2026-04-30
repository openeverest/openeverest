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

export const multinamespacesProject: PlaywrightTestProject[] = [
  {
    name: 'pr:multinamespaces',
    testMatch: /.^/,
    dependencies: [
      'pr:multinamespaces:db-wizard',
      'pr:multinamespaces:monitoring',
      'pr:multinamespaces:storage-location',
    ],
  },
  // pr:multinamespaces:db-wizard tests
  {
    name: 'pr:multinamespaces:db-wizard',
    testDir: './pr/multinamespaces',
    testMatch: /db-wizard\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  // pr:multinamespaces:monitoring tests
  {
    name: 'pr:multinamespaces:monitoring',
    testDir: './pr/multinamespaces',
    testMatch: /monitoring\.e2e\.ts/,
    dependencies: ['global:monitoring-instance:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  // pr:multinamespaces:storage-location tests
  {
    name: 'pr:multinamespaces:storage-location:setup',
    testDir: './pr/multinamespaces',
    testMatch: /storage-location\.setup\.ts/,
    dependencies: ['global:auth:ci:setup'],
    teardown: 'pr:multinamespaces:storage-location:teardown',
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    name: 'pr:multinamespaces:storage-location:teardown',
    testDir: './pr/multinamespaces',
    testMatch: /storage-location\.teardown\.ts/,
  },
  {
    name: 'pr:multinamespaces:storage-location',
    testDir: './pr/multinamespaces',
    testMatch: /storage-location\.e2e\.ts/,
    dependencies: [
      'global:backup-storage:setup',
      'pr:multinamespaces:storage-location:setup',
    ],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
];
