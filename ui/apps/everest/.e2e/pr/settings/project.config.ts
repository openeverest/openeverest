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

export const settingsProject: PlaywrightTestProject[] = [
  {
    name: 'pr:settings',
    testMatch: /.^/,
    dependencies: [
      'pr:settings:monitoring-config',
      // 'pr:settings:backup-storage',
      // 'pr:settings:namespace',
      // 'pr:settings:psp',
      // 'pr:settings:operator-upgrade',
    ],
  },
  // {
  //   name: 'pr:settings:backup-storage',
  //   testDir: './pr/settings',
  //   testMatch: /backup-storage\.e2e\.ts/,
  //   dependencies: ['global:auth:ci:setup'],
  //   use: {
  //     storageState: CI_USER_STORAGE_STATE_FILE,
  //   },
  // },
  {
    name: 'pr:settings:monitoring-config',
    testDir: './pr/settings',
    testMatch: /monitoring-config\.e2e\.ts/,
    dependencies: ['global:auth:ci:setup'],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  // {
  //   name: 'pr:settings:namespace',
  //   testDir: './pr/settings',
  //   testMatch: /namespaces-list\.e2e\.ts/,
  //   dependencies: ['global:auth:ci:setup'],
  //   use: {
  //     storageState: CI_USER_STORAGE_STATE_FILE,
  //   },
  // },
  // {
  //   name: 'pr:settings:psp',
  //   testDir: './pr/settings',
  //   testMatch: /pod-scheduling-policies\.e2e\.ts/,
  //   dependencies: ['global:auth:ci:setup'],
  //   use: {
  //     storageState: CI_USER_STORAGE_STATE_FILE,
  //   },
  // },
  // {
  //   name: 'pr:settings:operator-upgrade',
  //   testDir: './pr/settings',
  //   testMatch: /operator-upgrade\.e2e\.ts/,
  //   dependencies: ['global:auth:ci:setup'],
  //   use: {
  //     storageState: CI_USER_STORAGE_STATE_FILE,
  //   },
  // },
];
