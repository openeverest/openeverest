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

import { PlaywrightTestProject } from '@playwright/test';

// The options of the auth fixture (fixtures/auth.ts): who a worker logs in as.
interface AuthWorkerOptions {
  authUser: string;
  authPassword: string;
}

export const settingsProject: PlaywrightTestProject<
  object,
  AuthWorkerOptions
>[] = [
  {
    name: 'pr:settings',
    testMatch: /.^/,
    dependencies: [
      'pr:settings:backup-storage',
      'pr:settings:backup-storage-rbac',
      // 'pr:settings:namespace',
      // 'pr:settings:psp',
      // 'pr:settings:operator-upgrade',
    ],
  },
  {
    name: 'pr:settings:backup-storage',
    testDir: './pr/settings',
    testMatch: /backup-storage(-credentials|-checks)?\.e2e\.ts/,
  },
  // Logs in as the RBAC user; the other tests keep running as the admin.
  {
    name: 'pr:settings:backup-storage-rbac',
    testDir: './pr/settings',
    testMatch: /backup-storage-rbac\.e2e\.ts/,
    use: {
      authUser: process.env.RBAC_USER ?? '',
      authPassword: process.env.RBAC_PASSWORD ?? '',
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
