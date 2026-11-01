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
import { CI_USER_STORAGE_STATE_FILE } from '@e2e/constants';
// Temporarily disabled while the release lane runs the PITR golden only.
// Restore incrementally together with the other release suites.
// import { sessionProject } from './session/project.config';

export const releaseProject: PlaywrightTestProject[] = [
  {
    name: 'release',
    testMatch: /.^/,
    // Release lane runs the PITR golden plus restore-to-new-cluster for now.
    // Restore the rest incrementally: 'pr', 'release:session:rate-limiting',
    // ...sessionProject.
    dependencies: ['release:pitr', 'release:restore-new-cluster'],
  },
  {
    // Live PITR / restore-from-PITR golden (PXC). Auth is provided by the shared
    // fixture (@e2e/fixtures/auth); backup storage + monitoring come from the
    // global setup projects. The PXC provider must already be installed in the
    // cluster (Helm: `make deploy-pxc-provider`) before this runs.
    name: 'release:pitr',
    testDir: './release',
    testMatch: /pitr\.e2e\.ts/,
    dependencies: [
      'global:auth:ci:setup',
      'global:backup-storage:setup',
      'global:monitoring-config:setup',
    ],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  {
    // Restore-to-new-cluster (create new DB from a backup) golden. Shares the
    // same global backup storage + monitoring setup as the PITR lane.
    name: 'release:restore-new-cluster',
    testDir: './release',
    testMatch: /restore-new-cluster\.e2e\.ts/,
    dependencies: [
      'global:auth:ci:setup',
      'global:backup-storage:setup',
      'global:monitoring-config:setup',
    ],
    use: {
      storageState: CI_USER_STORAGE_STATE_FILE,
    },
  },
  // ...sessionProject,
];
