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

import { test as teardown } from '@playwright/test';
import { deleteDbClusterFn } from '@e2e/utils/db-cluster';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';

const dbClusterName = 'pr-db-det-comp';

teardown.describe.serial('DB Cluster Overview teardown', () => {
  teardown(`Delete DB cluster`, async ({ request }) => {
    await deleteDbClusterFn(
      request,
      dbClusterName,
      EVEREST_CI_NAMESPACES.PG_ONLY
    );
  });
});
