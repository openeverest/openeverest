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

import {expect, test} from '@fixtures'
import * as th from '@tests/utils/api';
import {MONITORING_CONFIG_1, MONITORING_CONFIG_2} from "@root/constants";

const testPrefix = 'pg-mon'

test.describe.parallel('PG monitoring tests', () => {
  test.describe.configure({timeout: 120 * 1000});

  test('create db cluster with monitoring', async ({request}) => {
    const dbClusterName = th.limitedSuffixedName(testPrefix)
    let dbClusterData = th.getPGClusterDataSimple(dbClusterName)
    dbClusterData.spec.monitoring = {monitoringConfigName: MONITORING_CONFIG_1}

    try {
      await test.step('create db cluster with monitoring config', async () => {
        await th.createDBClusterWithData(request, dbClusterData)

        await expect(async () => {
          const dbCluster = await th.getDBCluster(request, dbClusterName)
          expect(dbCluster.spec.monitoring.monitoringConfigName).toBe(MONITORING_CONFIG_1)
        }).toPass({
          intervals: [1000],
          timeout: 300 * 1000,
        })
      })
    } finally {
      await th.deleteDBCluster(request, dbClusterName)
    }
  })

  test('create db cluster with invalid monitoring config', async ({request}) => {
    const dbClusterName = th.limitedSuffixedName(testPrefix + '-inv')
    let dbClusterData = th.getPGClusterDataSimple(dbClusterName)
    dbClusterData.spec.monitoring = {monitoringConfigName: 'absent-config'}

    try {
      await test.step('create db cluster with monitoring config', async () => {
        await th.createDBClusterWithData(request, dbClusterData)

        await expect(async () => {
          const dbCluster = await th.getDBCluster(request, dbClusterName)
          expect(dbCluster.spec.monitoring.monitoringConfigName).toBe(dbClusterData.spec.monitoring.monitoringConfigName)
          expect(dbCluster.status.status).toMatch('creating')
          expect(dbCluster.status.size).toBeFalsy()
        }).toPass({
          intervals: [1000],
          timeout: 120 * 1000,
        })
      })
    } finally {
      await th.deleteDBCluster(request, dbClusterName)
    }
  })

  test('update db cluster with a new monitoring config', async ({request}) => {
    const dbClusterName = th.limitedSuffixedName(testPrefix + '-upd')
    let dbClusterData = th.getPGClusterDataSimple(dbClusterName)
    dbClusterData.spec.monitoring = {monitoringConfigName: MONITORING_CONFIG_1}

    try {
      let dbCluster
      await test.step('create DB cluster with monitoring config 1', async () => {
        await th.createDBClusterWithData(request, dbClusterData)

        await expect(async () => {
          dbCluster = await th.getDBCluster(request, dbClusterName)
          expect(dbCluster.spec.monitoring.monitoringConfigName).toBe(MONITORING_CONFIG_1)
        }).toPass({
          intervals: [1000],
          timeout: 300 * 1000,
        })
      })

      await test.step('set new monitoring config 2', async () => {
        await expect(async () => {
          dbCluster = await th.getDBCluster(request, dbClusterName)
          dbCluster.spec.monitoring.monitoringConfigName = MONITORING_CONFIG_2

          dbCluster = await th.updateDBCluster(request, dbClusterName, dbCluster)
          expect(dbCluster.spec.monitoring.monitoringConfigName).toBe(MONITORING_CONFIG_2)
        }).toPass({
          intervals: [1000],
          timeout: 300 * 1000,
        })
      })
    } finally {
      await th.deleteDBCluster(request, dbClusterName)
    }
  })

  test('enable monitoring for existing cluster', async ({request}) => {
    const dbClusterName = th.limitedSuffixedName(testPrefix),
      dbClusterData = th.getPGClusterDataSimple(dbClusterName)

    try {
      let dbCluster

      await test.step('create DB cluster without monitoring config', async () => {
        await th.createDBClusterWithData(request, dbClusterData)

        await expect(async () => {
          dbCluster = await th.getDBCluster(request, dbClusterName)
          expect(dbCluster.spec.monitoring?.monitoringConfigName).toBeFalsy()
        }).toPass({
          intervals: [1000],
          timeout: 300 * 1000,
        })
      })

      await test.step('set new monitoring config', async () => {
        await expect(async () => {
          dbCluster = await th.getDBCluster(request, dbClusterName)
          dbCluster.spec.monitoring = {monitoringConfigName: MONITORING_CONFIG_1}

          dbCluster = await th.updateDBCluster(request, dbClusterName, dbCluster)
          expect(dbCluster.spec.monitoring.monitoringConfigName).toBe(MONITORING_CONFIG_1)
        }).toPass({
          intervals: [1000],
          timeout: 300 * 1000,
        })
      })

    } finally {
      await th.deleteDBCluster(request, dbClusterName)
    }
  })

  test('disable monitoring', async ({request}) => {
    const dbClusterName = th.limitedSuffixedName(testPrefix + '-dis')
    let dbClusterData = th.getPGClusterDataSimple(dbClusterName)
    dbClusterData.spec.monitoring = {monitoringConfigName: MONITORING_CONFIG_1}

    try {
      let dbCluster

      await test.step('create DB cluster with monitoring config', async () => {
        await th.createDBClusterWithData(request, dbClusterData)

        await expect(async () => {
          dbCluster = await th.getDBCluster(request, dbClusterName)
          expect(dbCluster.spec.monitoring.monitoringConfigName).toBe(MONITORING_CONFIG_1)
        }).toPass({
          intervals: [1000],
          timeout: 300 * 1000,
        })
      })

      await test.step('disable monitoring', async () => {
        await expect(async () => {
          dbCluster = await th.getDBCluster(request, dbClusterName)
          dbCluster.spec.monitoring = {}

          dbCluster = await th.updateDBCluster(request, dbClusterName, dbCluster)
          expect(dbCluster.spec.monitoring?.monitoringConfigName).toBeFalsy()
        }).toPass({
          intervals: [1000],
          timeout: 300 * 1000,
        })
      })
    } finally {
      await th.deleteDBCluster(request, dbClusterName)
    }
  })
});