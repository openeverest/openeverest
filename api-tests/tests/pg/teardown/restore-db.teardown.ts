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

import {test as teardown} from '@playwright/test';
import * as th from "@tests/utils/api";
import {PG_RESTORE_DB_CLUSTER_NAME_ENV} from "@tests/pg/consts";

teardown.describe.parallel('PG cluster fore restore teardown', () => {
  teardown('Removing PG cluster', async ({request}) => {
    await th.deleteDBCluster(request, process.env[PG_RESTORE_DB_CLUSTER_NAME_ENV]);
  });
});