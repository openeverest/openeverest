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

export const sessionProject: PlaywrightTestProject[] = [
  {
    name: 'global:session:setup',
    testDir: './setup',
    testMatch: /session\.setup\.ts$/,
    teardown: 'global:session:teardown',
  },
  {
    name: 'global:session:teardown',
    testDir: './teardown',
    testMatch: /session\.teardown\.ts$/,
  },
  {
    name: 'release:session',
    testDir: './release/session',
    testMatch: /session-management\.e2e\.ts/,
    fullyParallel: false,
    dependencies: ['global:session:setup'],
  },
  {
    // Rate-limiting test poisons the token bucket — runs after all other session tests.
    name: 'release:session:rate-limiting',
    testDir: './release/session',
    testMatch: /rate-limiting\.e2e\.ts/,
    dependencies: ['release:session'],
  },
];
