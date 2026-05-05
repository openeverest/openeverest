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

test.describe('Everest CLI install', async () => {
  test('install all operators', async ({ page, cli, request }) => {
    const verifyClusterResources = async () => {
      await test.step('verify installed operators in k8s', async () => {
        const perconaEverestPodsOut = await cli.exec('kubectl get pods --namespace=everest-system');

        await perconaEverestPodsOut.outContainsNormalizedMany([
          'everest-operator',
        ]);

        const out = await cli.exec('kubectl get pods --namespace=everest-all');

        await out.outContainsNormalizedMany([
          'percona-xtradb-cluster-operator',
          'percona-server-mongodb-operator',
          'percona-postgresql-operator',
        ]);
      });
    };

    await test.step('run everest install command (pretty))', async () => {
      const out = await cli.everestExecSkipWizard(
        `install --namespaces=everest-all --version 0.0.0 --helm.set server.image=ghcr.io/openeverest/openeverest-dev --helm.set operator.image=ghcr.io/openeverest/openeverest-operator-dev --helm.set olm.catalogSourceImage=ghcr.io/openeverest/openeverest-catalog-dev`,
      );

      await out.assertSuccess();
      await out.outContainsNormalizedMany([
        '✅ Installing Everest Helm chart',
        '✅ Ensuring Everest API deployment is ready',
        '✅ Ensuring Everest operator deployment is ready',
        '✅ Ensuring OLM components are ready',
        '✅ Ensuring Everest CatalogSource is ready',
        '✅ Ensuring monitoring stack is ready',
        '✅ Provisioning database namespace \'everest-all\'',
        'Thank you for installing Everest',
      ]);
    });
    await page.waitForTimeout(10_000);
    await verifyClusterResources();

    await test.step('uninstall Everest', async () => {
      let out = await cli.everestExec(
        `uninstall --assume-yes -v`,
      );

      await out.assertSuccess();
      // check that the namespace does not exist
      out = await cli.exec('kubectl get ns everest-system everest-monitoring everest-olm everest-all');

      await out.outErrContainsNormalizedMany([
        'Error from server (NotFound): namespaces "everest-system" not found',
        'Error from server (NotFound): namespaces "everest-monitoring" not found',
        'Error from server (NotFound): namespaces "everest-olm" not found',
        'Error from server (NotFound): namespaces "everest-all" not found',
      ]);

    });

  });
});
