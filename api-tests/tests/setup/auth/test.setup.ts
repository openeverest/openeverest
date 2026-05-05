// everest
// Copyright (C) 2023 Percona LLC
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

import {test as setup} from '@playwright/test';
import {loginTESTUser} from '@tests/utils/user';

setup.describe.serial('Auth setup TEST', () => {
  setup('Login TEST user', async ({request, page}) => {
    // need to wait a little bit because this test and ci.setup.ts running simultaneously
    // lead to '429 Too many Requests' error
    await page.waitForTimeout(1000);
    await loginTESTUser(request);
  });
});