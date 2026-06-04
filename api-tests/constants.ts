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

import {fileURLToPath} from 'url';
import path from 'path';

export const EVEREST_CI_NAMESPACE = 'everest',
  {
    CI_USER,
    CI_PASSWORD,
    TEST_USER,
    TEST_PASSWORD,
  } = process.env,
  CI_USER_STORAGE_STATE_FILE = path.join(
    path.dirname(fileURLToPath(import.meta.url)),
    '.auth',
    'ci_user.json'
  ),
  API_CI_TOKEN = 'API_CI_TOKEN',
  API_TEST_TOKEN = 'API_TEST_TOKEN',
  CLUSTER_NAME = 'main',
  MONITORING_CONFIG_1 = 'pmm-conf-1',
  MONITORING_CONFIG_2 = 'pmm-conf-2';

const second = 1_000,
  minute = 60 * second;

export enum TIMEOUTS {
  FiveSeconds = 5 * second,
  TenSeconds = 10 * second,
  FifteenSeconds = 15 * second,
  ThirtySeconds = 30 * second,
  OneMinute = minute,
  ThreeMinutes = 3 * minute,
  FiveMinutes = 5 * minute,
  TenMinutes = 10 * minute,
  FifteenMinutes = 15 * minute,
  TwentyMinutes = 20 * minute,
  ThirtyMinutes = 30 * minute,
  SixtyMinutes = 60 * minute,
}
