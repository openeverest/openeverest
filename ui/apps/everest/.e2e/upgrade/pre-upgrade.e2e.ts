import { expect, test } from '@playwright/test';
import { createDbClusterFn } from '@e2e/utils/db-cluster';
import { pxcDBCluster, mongoDBCluster, postgresDBCluster } from './testData';
import { getDbClustersListAPI } from '@e2e/utils/db-clusters-list';
import { TIMEOUTS } from '@e2e/constants';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { prepareTestDB } from '@e2e/utils/db-cmd-line';
import { getK8sObjectsNamespaceYaml } from '@e2e/utils/kubernetes';

test.describe.configure({ retries: 0 });
test.describe.configure({ timeout: TIMEOUTS.TwentyMinutes });

test(
  'Pre upgrade setup',
  { tag: '@pre-upgrade' },
  async ({ page, request }) => {
    const token = await getTokenFromLocalStorage();

    await test.step('Create DB clusters', async () => {
      await createDbClusterFn(request, {
        dbName: pxcDBCluster.name,
        dbType: 'mysql',
        numberOfNodes: pxcDBCluster.numberOfNodes,
        numberOfProxies: pxcDBCluster.numberOfProxies,
        cpu: pxcDBCluster.cpu,
        disk: pxcDBCluster.disk,
        memory: pxcDBCluster.memory,
        externalAccess: pxcDBCluster.externalAccess,
        //sourceRanges: pxcDBCluster.sourceRanges,
      });

      await createDbClusterFn(request, {
        dbName: mongoDBCluster.name,
        dbType: 'mongodb',
        numberOfNodes: mongoDBCluster.numberOfNodes,
        cpu: mongoDBCluster.cpu,
        disk: mongoDBCluster.disk,
        memory: mongoDBCluster.disk,
        externalAccess: mongoDBCluster.externalAccess,
      });

      await createDbClusterFn(request, {
        dbName: postgresDBCluster.name,
        dbType: 'postgresql',
        numberOfNodes: postgresDBCluster.numberOfNodes,
        numberOfProxies: postgresDBCluster.numberOfProxies,
        cpu: postgresDBCluster.cpu,
        disk: postgresDBCluster.disk,
        memory: postgresDBCluster.disk,
        externalAccess: postgresDBCluster.externalAccess,
      });

      await page.waitForTimeout(TIMEOUTS.TenSeconds);
    });

    await test.step('Wait for databases to become ready', async () => {
      await expect(async () => {
        const dbClusters = (
          await getDbClustersListAPI(
            EVEREST_CI_NAMESPACES.EVEREST_UI,
            request,
            token
          )
        ).items;

        const clustersInfo = dbClusters.map((c) => {
          return { status: c.status.status, name: c.metadata.name };
        });

        clustersInfo.forEach((c) => {
          expect(c.status, `expecting ${c.name} to have "ready" status`).toBe(
            'ready'
          );
        });
      }, 'waiting for db clusters to be "ready"').toPass({
        timeout: TIMEOUTS.FifteenMinutes,
        intervals: [TIMEOUTS.OneMinute],
      });
    });

    await test.step('Add data to databases', async () => {
      const dbClusters = (
        await getDbClustersListAPI(
          EVEREST_CI_NAMESPACES.EVEREST_UI,
          request,
          token
        )
      ).items;

      const clustersInfo = dbClusters.map((c) => {
        return {
          status: c.status.status,
          name: c.metadata.name,
          namespace: c.metadata.namespace,
        };
      });

      clustersInfo.forEach((c) => {
        prepareTestDB(c.name, c.namespace);
      });
    });

    await test.step('Collect info about k8s objects', async () => {
      await getK8sObjectsNamespaceYaml(
        EVEREST_CI_NAMESPACES.EVEREST_UI,
        '1-pre-upgrade'
      );
    });
  }
);
