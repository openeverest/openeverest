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

import {expect} from '@playwright/test'
import type {APIRequestContext} from '@playwright/test'
import {EVEREST_CI_NAMESPACE, TIMEOUTS, CLUSTER_NAME} from '@root/constants';
import {setRequestContext} from '@helpers/playwright-fetcher'
import * as client from '@root/generated/http-api.client.gen'
import type {BackupStorage} from '@root/generated/http-api.client.gen'


// --------------------- General helpers --------------------------------------------------
// testPrefix is used to differentiate between several workers
// running this test to avoid conflicts in instance names
export const testPrefix = () => {
  let result = '';
  while (result.length < 16) {
    result += Math.random().toString(36).substring(2);
  }
  return result.substring(0, 16);
}

export const suffixedName = (name: string) => {
  return `${name}-${testPrefix()}`
}

export const limitedSuffixedName = (name: string) => {
  return `${name}-${testPrefix()}`.substring(0, 21)
}

export const checkError = async (response) => {
  // if (!response.ok()) {
  //   console.log(`${response.url()}: `, await response.json());
  // }

  expect(response.ok()).toBeTruthy()
}

// Checks whether the passed resource is marked for deletion(metadata.deletionTimestamp)
// or checks for '404 Not Found' response from server that means that resource has been deleted
// from K8S.
export const checkResourceDeletion = async (apiResp) => {
  if (apiResp.status() === 200) {
    expect((await apiResp.json()).metadata['deletionTimestamp']).not.toBe('');
  } else {
    expect(apiResp.status()).toBe(404)
  }
}

// --------------------- DB Cluster helpers -----------------------------------------------
// Returns DBCluster object for creating 1-node PG cluster
export const getPGClusterDataSimple = (name: string) => {
  const data = {
    apiVersion: 'everest.percona.com/v1alpha1',
    kind: 'DatabaseCluster',
    metadata: {
      name: name,
    },
    spec: {
      engine: {
        type: 'postgresql',
        replicas: 1,
        storage: {
          size: '1Gi',
        },
        resources: {
          cpu: '1',
          memory: '1G',
        },
      },
      proxy: {
        type: 'pgbouncer',
        replicas: 1,
        expose: {
          type: 'internal',
        },
      },
    },
  }
  return JSON.parse(JSON.stringify(data))
}

// Returns DBCluster object for creating 1-node PSMDB cluster
export const getPSMDBClusterDataSimple = (name: string) => {
  let data = {
    apiVersion: 'everest.percona.com/v1alpha1',
    kind: 'DatabaseCluster',
    metadata: {
      name: name,
    },
    spec: {
      backup: {
        pitr: {
          enabled: false,
        },
      },
      engine: {
        type: 'psmdb',
        replicas: 1,
        storage: {
          size: '1Gi',
        },
        resources: {
          cpu: '1',
          memory: '1G',
        },
      },
      proxy: {
        type: 'mongos',
        replicas: 1,
        expose: {
          type: 'internal',
        },
      },
      sharding: {
        configServer: {
          replicas: 1,
        },
        enabled: false,
        shards: 2,
      },
    }
  }
  return JSON.parse(JSON.stringify(data))
}

// Returns DBCluster object for creating 1-node PXC cluster
export const getPXCClusterDataSimple = (name: string) => {
  let data = {
    apiVersion: 'everest.percona.com/v1alpha1',
    kind: 'DatabaseCluster',
    metadata: {
      name: name,
    },
    spec: {
      engine: {
        type: 'pxc',
        replicas: 1,
        config: '[mysqld]\nwsrep_provider_options="debug=1;gcache.size=1G"\n',
        storage: {
          size: '1Gi',
        },
        resources: {
          cpu: '1',
          memory: '1G',
        },
      },
      proxy: {
        type: 'haproxy',
        replicas: 1,
        expose: {
          type: 'internal',
        },
      },
    },
  }
  return JSON.parse(JSON.stringify(data))
}

// Creates a simple 1-node PG cluster and returns DB creation response (JSON)
export const createDBCluster = async (request, name) => {
  const data = getPGClusterDataSimple(name)
  return await createDBClusterWithData(request, data)
}

// Creates DB cluster with provided data as body.
// Expects successful creation.
export const createDBClusterWithData = async (request, data) => {
  const createResp = await createDBClusterWithDataRaw(request, data)
  await checkError(createResp)
  return (await createResp.json())
}

// Creates DB cluster with provided data as body.
// Returns raw response object without any checks or parsing.
export const createDBClusterWithDataRaw = async (request, data) => {
  return await request.post(`/v1/namespaces/${EVEREST_CI_NAMESPACE}/database-clusters`, {data})
}

// Waits for DB cluster status to be Ready
export const waitForDBClusterToBeReady = async (request, name) => {
  await expect.poll(async () => {
    const db = await getDBCluster(request, name)
    return db.status.status
  }, {
    intervals: [TIMEOUTS.TenSeconds],
    timeout: TIMEOUTS.TenMinutes,
  }).toBe('ready')
}

export const getDBCluster = async (request, name) => {
  const response = await getDBClusterRaw(request, name)
  await checkError(response)
  // expect(dbClusterResp.ok()).toBeTruthy()
  return (await response.json())
}

export const getDBClusterRaw = async (request, name) => {
  return await request.get(`/v1/namespaces/${EVEREST_CI_NAMESPACE}/database-clusters/${name}`)
}

export const updateDBCluster = async (request, name, updateData) => {
  const response = await request.put(`/v1/namespaces/${EVEREST_CI_NAMESPACE}/database-clusters/${name}`, {data: updateData})
  await checkError(response)
  return (await response.json())
}

export const deleteDBCluster = async (request, name) => {
  // Wait for deletion mark.
  await expect(async () => {
    await deleteDBClusterRaw(request, name)
    const res = await getDBClusterRaw(request, name)
    await checkResourceDeletion(res)
  }).toPass({
    intervals: [1000],
    timeout: 60 * 1000,
  })
}

export const deleteDBClusterRaw = async (request, name) => {
  return await request.delete(`/v1/namespaces/${EVEREST_CI_NAMESPACE}/database-clusters/${name}`)
}

// --------------------- Backup Storage helpers -----------------------------------------

export const getBackupStoragePayload = (bsName: string) => {
  return {
    metadata: {
      name: bsName,
    },
    spec: {
      type: 's3' as const,
      s3: {
        bucket: 'bucket-4',
        region: 'us-east-1',
        endpointURL: 'https://minio.minio.svc',
        credentialsSecretName: `${bsName}-creds`,
        accessKeyId: 'minioadmin',
        secretAccessKey: 'minioadmin',
        forcePathStyle: true,
        verifyTLS: false,
      },
    },
  }
}

export const generateBackupStorage = async (request: APIRequestContext, data: BackupStorage) => {
  const response = await createBackupStorageRaw(request, data)
  await checkError(response)
  return (await response.json())
}

export const createBackupStorageRaw = async (request: APIRequestContext, data: BackupStorage) => {
  setRequestContext(request)
  return await client.createBackupStorage(CLUSTER_NAME, EVEREST_CI_NAMESPACE, data) as any
}

export const getBackupStorage = async (request: APIRequestContext, name: string) => {
  const response = await getBackupStorageRaw(request, name)
  await checkError(response)
  return (await response.json())
}

export const getBackupStorageRaw = async (request: APIRequestContext, name: string) => {
  setRequestContext(request)
  return await client.getBackupStorage(CLUSTER_NAME, EVEREST_CI_NAMESPACE, name) as any
}

export const listBackupStorages = async (request: APIRequestContext) => {
  const response = await listBackupStoragesRaw(request)
  await checkError(response)
  return (await response.json())
}

export const listBackupStoragesRaw = async (request: APIRequestContext) => {
  setRequestContext(request)
  return await client.listBackupStorages(CLUSTER_NAME, EVEREST_CI_NAMESPACE) as any
}

export const updateBackupStorage = async (request: APIRequestContext, name: string, data: BackupStorage) => {
  const response = await updateBackupStorageRaw(request, name, data)
  await checkError(response)
  return (await response.json())
}

export const updateBackupStorageRaw = async (request: APIRequestContext, name: string, data: BackupStorage) => {
  setRequestContext(request)
  return await client.updateBackupStorage(CLUSTER_NAME, EVEREST_CI_NAMESPACE, name, data) as any
}

export const deleteBackupStorage = async (request: APIRequestContext, name: string) => {
  // Wait for deletion mark.
  await expect(async () => {
    await deleteBackupStorageRaw(request, name)
    const res = await getBackupStorageRaw(request, name)
    await checkResourceDeletion(res)
  }).toPass({
    intervals: [1000],
    timeout: 60 * 1000,
  })
}

export const deleteBackupStorageRaw = async (request: APIRequestContext, name: string) => {
  setRequestContext(request)
  return await client.deleteBackupStorage(CLUSTER_NAME, EVEREST_CI_NAMESPACE, name) as any
}

// --------------------- Monitoring Config helpers (V2 - using new endpoints) -----

export const createMonitoringConfigV2 = async (request, name) => {
  const miData = {
    type: 'pmm',
    name: name,
    url: `https://${process.env.PMM1_IP}`,
    pmm: {
      apiKey: `${process.env.PMM1_API_KEY}`,
    },
    verifyTLS: false,
  }
  return await createMonitoringConfigWithDataRawV2(request, miData)
}

export const createMonitoringConfigWithDataV2 = async (request, data) => {
  const response = await createMonitoringConfigWithDataRawV2(request, data)
  await checkError(response)
  return (await response.json())
}

export const createMonitoringConfigWithDataRawV2 = async (request, data) => {
  return await request.post(`/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/monitoring-configs`, {data: data})
}

export const getMonitoringConfigV2 = async (request, name) => {
  const response = await getMonitoringConfigRawV2(request, name)
  await checkError(response)
  return (await response.json())
}

export const getMonitoringConfigRawV2 = async (request, name) => {
  return await request.get(`/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/monitoring-configs/${name}`)
}

export const updateMonitoringConfigV2 = async (request, name, data) => {
  const response = await updateMonitoringConfigRawV2(request, name, data)
  await checkError(response)
  return (await response.json())
}

export const updateMonitoringConfigRawV2 = async (request, name, data) => {
  return await request.patch(`/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/monitoring-configs/${name}`, {data: data})
}

export const deleteMonitoringConfigV2 = async (request, name) => {
  // Wait for deletion mark.
  await expect(async () => {
    await deleteMonitoringConfigRawV2(request, name)
    const res = await getMonitoringConfigRawV2(request, name)
    await checkResourceDeletion(res)
  }).toPass({
    intervals: [1000],
    timeout: 300 * 1000,
  })
}

export const deleteMonitoringConfigRawV2 = async (request, name) => {
  return await request.delete(`/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/monitoring-configs/${name}`)
}


