// everest
// Copyright (C) 2023 Percona LLC
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
import { test } from '@fixtures';
import { faker } from '@faker-js/faker';

test.describe('Everest CLI install', async () => {
  test('install only mongodb-operator', async ({ page, cli, request }) => {
    const verifyClusterResources = async () => {
      await test.step('verify installed operators in k8s', async () => {
        const perconaEverestPodsOut = await cli.exec('kubectl get pods --namespace=everest-system');

        await perconaEverestPodsOut.outContainsNormalizedMany([
          'everest-operator',
        ]);

        const out = await cli.exec('kubectl get pods --namespace=everest-operators');

        await out.outContainsNormalizedMany([
          'percona-server-mongodb-operator',
        ]);

        await out.outNotContains([
          'percona-postgresql-operator',
          'percona-xtradb-cluster-operator',
        ]);
      });
    },
     clusterName = `test-${faker.number.int()}`;

    await test.step('run everest install command (pretty))', async () => {
      const out = await cli.everestExecSkipWizard(
        `install --operator.mongodb=true --operator.postgresql=false --operator.mysql=false --namespaces=everest-operators --version 0.0.0 --helm.set server.image=ghcr.io/openeverest/openeverest-dev --helm.set operator.image=ghcr.io/openeverest/openeverest-operator-dev --helm.set olm.catalogSourceImage=ghcr.io/openeverest/openeverest-catalog-dev`,
      );

      await out.assertSuccess();
      await out.outContainsNormalizedMany([
        '✅ Installing Everest Helm chart',
        '✅ Ensuring Everest API deployment is ready',
        '✅ Ensuring Everest operator deployment is ready',
        '✅ Ensuring OLM components are ready',
        '✅ Ensuring Everest CatalogSource is ready',
        '✅ Ensuring monitoring stack is ready',
        '✅ Provisioning database namespace \'everest-operators\'',
        'Thank you for installing Everest',
      ]);
    });
    await page.waitForTimeout(10_000);
    await verifyClusterResources();
  });
});
