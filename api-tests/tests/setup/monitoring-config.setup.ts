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
import * as th from "@tests/utils/api";
import {MONITORING_CONFIG_1, MONITORING_CONFIG_2} from "@root/constants";

setup.describe.parallel('Monitoring config setup', () => {
  setup('Create Monitoring config 1', async ({request}) => {
    const data = {
      type: 'pmm',
      name: MONITORING_CONFIG_1,
      url: `https://${process.env.PMM1_IP}`,
      pmm: {
        apiKey: `${process.env.PMM1_API_KEY}`,
      },
      verifyTLS: false,
    }
    await th.createMonitoringConfigWithData(request, data);
  });

  setup('Create Monitoring config 2', async ({request}) => {
    const data = {
      type: 'pmm',
      name: MONITORING_CONFIG_2,
      url: `https://${process.env.PMM2_IP}`,
      pmm: {
        apiKey: `${process.env.PMM2_API_KEY}`,
      },
      verifyTLS: false,
    }
    await th.createMonitoringConfigWithData(request, data);
  });
});