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

// Visual-regression test fixture. Composes over the shared auth fixture WITHOUT
// modifying it: freezes time and installs the default deterministic API mocks
// for every authenticated visual test.

import { test as authTest, expect } from '@e2e/fixtures/auth';
import { installVisualMocks } from './routes';

export const test = authTest.extend({
  page: async ({ page }, use) => {
    // Freeze wall-clock time so any relative/absolute date rendering is stable.
    // Use setFixedTime (NOT clock.install): it overrides Date.now()/new Date()
    // for deterministic rendering while keeping real timers running, so the
    // auth token-refresh scheduling in the app is not frozen mid-test.
    await page.clock.setFixedTime(new Date('2026-07-08T00:00:00Z'));
    await installVisualMocks(page);
    await use(page);
  },
});

export { expect };
