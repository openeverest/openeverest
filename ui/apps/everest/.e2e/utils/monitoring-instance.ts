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

import { APIRequestContext, expect } from '@playwright/test';
import { execSync } from 'child_process';
import { getDBClientPod } from '@e2e/utils/db-cmd-line';
import { getK8sUid } from '@e2e/utils/kubernetes';

const { MONITORING_URL, MONITORING_USER, MONITORING_PASSWORD } = process.env;

export const testMonitoringName = 'ui-test-monitoring';
export const testMonitoringName2 = 'ui-test-monitoring-1';

export const createMonitoringInstance = async (
  request: APIRequestContext,
  name: string,
  namespace: string,
  token: string
) => {
  const data = {
    name,
    type: 'pmm',
    url: MONITORING_URL,
    allowedNamespaces: [],
    verifyTLS: false,
    pmm: {
      user: MONITORING_USER,
      password: MONITORING_PASSWORD,
    },
  };

  const response = await request.post(
    `/v1/namespaces/${namespace}/monitoring-instances`,
    {
      data,
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );

  expect(response.ok()).toBeTruthy();
};

export const deleteMonitoringInstance = async (
  request: APIRequestContext,
  namespace,
  name,
  token: string
) => {
  const response = await request.delete(
    `/v1/namespaces/${namespace}/monitoring-instances/${name}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  expect(response.ok()).toBeTruthy();
};

export const listMonitoringInstances = async (
  request: APIRequestContext,
  namespace,
  token: string
) => {
  const response = await request.get(
    `/v1/namespaces/${namespace}/monitoring-instances`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  expect(response.ok()).toBeTruthy();

  const responseBody = await response.json();
  return responseBody;
};

export const getPMMMajorVersion = async () => {
  try {
    const command = `kubectl get pods -l app.kubernetes.io/component=pmm-server -n everest-system -o jsonpath='{.items[0].spec.containers[0].image}' | cut -d':' -f2 | cut -d'.' -f1`;
    const output = execSync(command).toString();
    return output;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

export const checkDBMetrics = async (
  metric: string,
  instance: string,
  userPass: string
): Promise<void> => {
  const clientPod = await getDBClientPod('psmdb', 'db-client');
  const date = new Date();
  const end = Math.floor(date.getTime() / 1000);
  const start = end - 60;
  const endpoint = 'monitoring-service.everest-system:443';
  const url = `https://${userPass}@${endpoint}/graph/api/datasources/proxy/1/api/v1/query_range?query=min%28${metric}%7Bnode_name%3D%7E%22${instance}%22%7d%20or%20${metric}%7Bnode_name%3D%7E%22${instance}%22%7D%29&start=${start}&end=${end}&step=60`;

  try {
    const command = `kubectl exec --namespace db-client ${clientPod} -- curl -s -k "${url}"`;
    var output = JSON.parse(execSync(command).toString());
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }

  const metrics = parseInt(output?.data?.result[0]?.values[0][1]);
  expect(metrics).toBeDefined();
  expect(typeof metrics).toBe('number');
  expect(Number.isNaN(metrics)).toBeFalsy();
};

export const checkK8sMetrics = async (
  metric: string,
  userPass: string
): Promise<void> => {
  const clientPod = await getDBClientPod('psmdb', 'db-client');
  const k8sUid = await getK8sUid();
  const date = new Date();
  const end = Math.floor(date.getTime() / 1000);
  const start = end - 60;
  const endpoint = 'monitoring-service.everest-system:443';
  const url = `https://${userPass}@${endpoint}/graph/api/datasources/proxy/1/api/v1/query_range?query=count%28${metric}%7Bk8s_cluster_id%3D%22${k8sUid}%22%7D%29&start=${start}&end=${end}&step=60`;

  try {
    const command = `kubectl exec --namespace db-client ${clientPod} -- curl -s -k "${url}"`;
    var output = JSON.parse(execSync(command).toString());
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }

  const metrics = parseInt(output?.data?.result[0]?.values[0][1]);
  expect(metrics).toBeDefined();
  expect(typeof metrics).toBe('number');
  expect(Number.isNaN(metrics)).toBeFalsy();
};

export const checkQAN = async (
  serviceType: string,
  node: string,
  userPass: string
): Promise<void> => {
  const clientPod = await getDBClientPod('psmdb', 'db-client');
  const now = new Date();
  const start = new Date(now.getTime() - 12 * 60 * 60 * 1000).toISOString();
  const end = now.toISOString();
  const PMMMajorVersion = await getPMMMajorVersion();
  const endpoint = 'monitoring-service.everest-system:443';
  const url =
    PMMMajorVersion.trim() === '2'
      ? `https://${userPass}@${endpoint}/v0/qan/ObjectDetails/GetMetrics`
      : `https://${userPass}@${endpoint}/v1/qan:getMetrics`;

  const payload = JSON.stringify({
    period_start_from: start,
    period_start_to: end,
    group_by: 'queryid',
    filter_by: node,
    labels: [
      { key: 'service_type', value: [serviceType] },
      { key: 'node_name', value: [node] },
    ],
    limit: 10,
    totals: true,
  });

  const encodedPayload = Buffer.from(payload).toString('base64');

  try {
    const command = `kubectl exec -n db-client ${clientPod} -- sh -c "echo '${encodedPayload}' | base64 -d > /tmp/payload.json && curl -s -k -XPOST -d @/tmp/payload.json '${url}'"`;
    var output = JSON.parse(execSync(command).toString());
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }

  // we use node_name because if it's returned it means there is some data for the node
  const nodeName = output?.metadata?.node_name.toString();
  expect(nodeName).toBeTruthy();
  expect(nodeName?.trim()).not.toBe('');
};
