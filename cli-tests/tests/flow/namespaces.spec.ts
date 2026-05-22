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
import { test, expect } from '@fixtures';

test.describe('Everest CLI install', async () => {
  test('install everest and db-namespaces separately', async ({ page, cli, request }) => {
    const verifyOperators = async (namespace: string, operators: string[]) => {
      await test.step('verify installed operators', async () => {
        const out = await cli.exec(`kubectl get pods --namespace=${namespace}`);

        await out.outContainsNormalizedMany(operators);
      });
    },

     verifyEverestSystem = async () => {
      await test.step('verify Everest', async () => {
        const out = await cli.exec(`kubectl get deploy --namespace=everest-system`);

        await out.outContainsNormalizedMany([
            'everest-operator',
            'everest-server'
        ]);
      });
    };

    await test.step('run everest install (no database namespace))', async () => {
      const out = await cli.everestExecSkipWizard(
        `install --skip-db-namespace --version 0.0.0 --helm.set server.image=ghcr.io/openeverest/openeverest-dev --helm.set operator.image=ghcr.io/openeverest/openeverest-operator-dev --helm.set olm.catalogSourceImage=ghcr.io/openeverest/openeverest-catalog-dev`,
      );

      await out.assertSuccess();
      await out.outContainsNormalizedMany([
        '✅ Installing Everest Helm chart',
        '✅ Ensuring Everest API deployment is ready',
        '✅ Ensuring Everest operator deployment is ready',
        '✅ Ensuring OLM components are ready',
        '✅ Ensuring Everest CatalogSource is ready',
        '✅ Ensuring monitoring stack is ready',
        'Thank you for installing Everest',
      ]);
    });
    await page.waitForTimeout(10_000);
    await verifyEverestSystem();

    await test.step('run everest install again (fail))', async () => {
      const out = await cli.everestExecSkipWizard(
        `install`,
      );

      await out.outErrContainsNormalizedMany([
          '❌ everest is already installed',
      ]);
    });
    await page.waitForTimeout(10_000);
    await verifyEverestSystem();

    await test.step('create database namespace', async () => {
      const out = await cli.everestExecNamespacesSkipWizard(
        `add everest --operator.mongodb=false --operator.postgresql=false --operator.mysql=true`,
      );

      await out.assertSuccess();
      await out.outContainsNormalizedMany([
          '✅ Provisioning database namespace \'everest\'',
      ]);
    });
    await page.waitForTimeout(10_000);
    await verifyEverestSystem();
    await verifyOperators('everest', ['percona-xtradb-cluster-operator']);

    await test.step('create database namespace again (fail))', async () => {
      const out = await cli.everestExecNamespacesSkipWizard(
        `add everest --operator.mongodb=false --operator.postgresql=false --operator.mysql=true`,
      );

      await out.outErrContainsNormalizedMany([
        '❌ \'everest\': namespace already exists and is managed by Everest',
      ]);
    });
    await page.waitForTimeout(10_000);
    await verifyEverestSystem();
    await verifyOperators('everest', ['percona-xtradb-cluster-operator']);

    await test.step('update database namespace', async () => {
      const out = await cli.everestExecNamespacesSkipWizard(
        `update everest --operator.mongodb=true --operator.postgresql=true --operator.mysql=true`,
      );

      await out.assertSuccess();
      await out.outContainsNormalizedMany([
          '✅ Updating database namespace \'everest\'',
      ]);
    });
    await page.waitForTimeout(10_000);
    await verifyEverestSystem();
    await verifyOperators('everest', [
        'percona-xtradb-cluster-operator',
        'percona-server-mongodb-operator',
        'percona-postgresql-operator',
    ]);

    await test.step('remove database namespace', async () => {
      let out = await cli.everestExecNamespaces(
        `remove everest`,
      );

      await out.assertSuccess();
      await out.outContainsNormalizedMany([
          '✅ Deleting database clusters in namespace \'everest\'',
          '✅ Deleting backup storages in namespace \'everest\'',
          '✅ Deleting monitoring instances in namespace \'everest\'',
          '✅ Deleting database namespace \'everest\'',
      ]);

      out = await cli.exec(`kubectl get namespace everest`);
      await out.outErrContainsNormalizedMany([
        'Error from server (NotFound): namespaces "everest" not found',
      ]);
    });
    await page.waitForTimeout(10_000);
    await verifyEverestSystem();

    await test.step('create database namespace with --take-ownership', async () => {
      let out = await cli.exec(`kubectl create namespace existing-ns`);

      await out.assertSuccess();

      out = await cli.everestExecNamespacesSkipWizard(
        `add existing-ns --take-ownership`,
      );
      await out.assertSuccess();
      await out.outContainsNormalizedMany([
      '✅ Provisioning database namespace \'existing-ns\'',
      ]);
    });
    await page.waitForTimeout(10_000);
    await verifyEverestSystem();

    await test.step('remove database namespace with --keep-namespace', async () => {
      let out = await cli.everestExecNamespaces(
        `remove existing-ns --keep-namespace`,
      );

      await out.assertSuccess();
      await out.outContainsNormalizedMany([
          '✅ Deleting database clusters in namespace \'existing-ns\'',
          '✅ Deleting backup storages in namespace \'existing-ns\'',
          '✅ Deleting monitoring instances in namespace \'existing-ns\'',
          '✅ Deleting resources from namespace \'existing-ns\'',
      ]);

      out = await cli.exec(`kubectl get namespace existing-ns`);
      await out.assertSuccess();
      await out.outContainsNormalizedMany([
          'existing-ns',
      ]);

    });
    await page.waitForTimeout(10_000);
    await verifyEverestSystem();

  });
});
