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

import {expect, test} from '@fixtures'
import * as th from '@tests/utils/api';

const secretName1 = 'test-secret-name-1'
const secretName2 = 'test-secret-name-2'
const secretName3 = 'test-secret-name-3'

test.describe.parallel('Secrets tests', () => {
  test.afterAll(async ({request}) => {
    await th.deleteSecret(request, secretName1)
    await th.deleteSecret(request, secretName2)
    await th.deleteSecret(request, secretName3)
  })

  test('create/get/list/delete secret', async ({request}) => {
    await test.step('create secret with data', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'Secret',
        metadata: {
          name: secretName1,
          labels: {
            'openeverest.io/definition': 'test-credentials',
            'openeverest.io/provider': 'test-provider',
          },
        },
        type: 'Opaque',
        data: {
          accessKey: Buffer.from('myaccesskey').toString('base64'),
          secretKey: Buffer.from('mysecretkey').toString('base64'),
        },
      }

      const created = await th.createSecretWithData(request, data)
      expect(created.metadata.name).toBe(data.metadata.name)
      expect(created.type).toBe('Opaque')
      // Verify no data is returned on create
      expect(created.data).toBeUndefined()
      expect(created.stringData).toBeUndefined()
    })

    await test.step('get secret', async () => {
      const secret = await th.getSecret(request, secretName1)
      expect(secret.metadata.name).toBe(secretName1)
      expect(secret.metadata.labels['openeverest.io/managed']).toBe('true')
      expect(secret.metadata.labels['openeverest.io/definition']).toBe('test-credentials')
      expect(secret.metadata.labels['openeverest.io/provider']).toBe('test-provider')
      // Verify no sensitive data is returned
      expect(secret.data).toBeUndefined()
      expect(secret.stringData).toBeUndefined()
    })

    await test.step('list secrets', async () => {
      const list = await th.listSecrets(request)
      expect(list.items.length).toBeGreaterThan(0)
      
      // Find our created secrets
      const secret = list.items.find(s => s.metadata.name === secretName1)
      
      expect(secret).toBeDefined()
      expect(secret.data).toBeUndefined()
      expect(secret.stringData).toBeUndefined()
    })

    await test.step('list secrets with filters', async () => {
      const list = await th.listSecretsWithFilters(request, 'test-provider', 'test-credentials')
      
      const secret1 = list.items.find(s => s.metadata.name === secretName1)
      expect(secret1).toBeDefined()
      expect(secret1.metadata.labels['openeverest.io/definition']).toBe('test-credentials')
      
      const emptyList = await th.listSecretsWithFilters(request, 'test-provider', 'non-existent')
      expect(emptyList.items.length).toBe(0)
    })

    await test.step('create secret without definition label should fail', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'Secret',
        metadata: {
          name: secretName2,
          labels: {
            'openeverest.io/provider': 'test-provider',
          },
        },
        type: 'Opaque',
        data: {
          key: Buffer.from('value').toString('base64'),
        },
      }

      const response = await th.createSecretRaw(request, data)
      expect(response.status()).toBe(400)
      const error = await response.json()
      expect(error.message).toContain('definition')
    })

    await test.step('create secret without a source label should fail', async () => {
      const data = {
        apiVersion: 'v1',
        kind: 'Secret',
        metadata: {
          name: secretName3,
          labels: {
            'openeverest.io/definition': 'shared-secret',
          },
        },
        type: 'Opaque',
        stringData: {
          username: 'plainuser',
          password: 'plainpassword',
        },
      }

      const response = await th.createSecretRaw(request, data)
      expect(response.status()).toBe(400)
      const error = await response.json()
      expect(error.message).toContain('provider')
    })

    await test.step('delete secret', async () => {
      await th.deleteSecret(request, secretName1)
      
      // Verify secret is deleted
      const response = await th.getSecretRaw(request, secretName1)
      expect(response.status()).toBe(404)
    })
  })
})

const monitoringClass = 'test-monitoring-class'
const monitoringSecretName = 'monitoring-destination-test-pmm-prod-credentials'

const monitoringSecret = (name: string, definition: string, extra: object = {}) => ({
  apiVersion: 'v1',
  kind: 'Secret',
  metadata: {
    name,
    labels: {
      'openeverest.io/monitoring-class': monitoringClass,
      'openeverest.io/definition': definition,
    },
  },
  type: 'Opaque',
  stringData: {token: 'glsa_test-token'},
  ...extra,
})

const withLabels = (labels: object) => (name: string) => ({
  ...monitoringSecret(name, 'ServiceAccountToken'),
  metadata: {name, labels},
})

const rejected = [
  {
    desc: 'a required key is missing',
    data: (name: string) => monitoringSecret(name, 'ServiceAccountToken', {stringData: {orgID: '1'}}),
    message: 'missing key "token"',
  },
  {
    desc: 'the class has no such mode',
    data: (name: string) => monitoringSecret(name, 'BasicAuth'),
    message: 'has no authentication mode "BasicAuth"',
  },
  {
    desc: 'the mode takes no Secret',
    data: (name: string) => monitoringSecret(name, 'Anonymous'),
    message: 'takes no Secret',
  },
  {
    desc: 'the class does not exist',
    data: withLabels({'openeverest.io/monitoring-class': 'non-existent', 'openeverest.io/definition': 'ServiceAccountToken'}),
    message: 'MonitoringClass "non-existent"',
  },
  {
    desc: 'the Secret is not Opaque',
    data: (name: string) => monitoringSecret(name, 'ServiceAccountToken', {type: 'kubernetes.io/basic-auth', stringData: {token: 't', username: 'u'}}),
    message: 'must be of type "Opaque"',
  },
  {
    desc: 'the Secret would receive a service account token',
    data: (name: string) => ({
      ...monitoringSecret(name, 'ServiceAccountToken', {type: 'kubernetes.io/service-account-token', stringData: undefined}),
      metadata: {
        name,
        labels: {'openeverest.io/monitoring-class': monitoringClass, 'openeverest.io/definition': 'ServiceAccountToken'},
        annotations: {'kubernetes.io/service-account.name': 'default'},
      },
    }),
    message: 'cannot be created',
  },
  {
    desc: 'the Secret has no labels',
    data: withLabels({}),
    message: 'label the Secret with one of',
  },
  {
    desc: 'the Secret names two sources',
    data: withLabels({
      'openeverest.io/monitoring-class': monitoringClass,
      'openeverest.io/provider': 'test-provider',
      'openeverest.io/definition': 'ServiceAccountToken',
    }),
    message: 'set only one of',
  },
]

const rejectedName = (i: number) => `monitoring-destination-test-rejected-${i}-credentials`

test.describe.parallel('Secrets defined by a monitoring class', () => {
  test.afterAll(async ({request}) => {
    await th.deleteSecret(request, monitoringSecretName)
    for (let i = 0; i < rejected.length; i++) {
      await th.deleteSecret(request, rejectedName(i))
    }
  })

  test('create, filter and get monitoring credentials', async ({request}) => {
    await test.step('create credentials with a name longer than 22 characters', async () => {
      const data = {
        ...monitoringSecret(monitoringSecretName, 'ServiceAccountToken'),
        metadata: {
          name: monitoringSecretName,
          labels: {'openeverest.io/monitoring-class': monitoringClass, 'openeverest.io/definition': 'ServiceAccountToken'},
          ownerReferences: [{apiVersion: 'v1', kind: 'ConfigMap', name: 'not-an-owner', uid: '00000000-0000-0000-0000-000000000000'}],
          finalizers: ['example.com/keep'],
        },
      }

      const created = await th.createSecretWithData(request, data)
      expect(created.metadata.name).toBe(monitoringSecretName)
      expect(created.data).toBeUndefined()
      expect(created.stringData).toBeUndefined()
    })

    await test.step('get drops owner references and finalizers', async () => {
      const secret = await th.getSecret(request, monitoringSecretName)
      expect(secret.metadata.labels['openeverest.io/managed']).toBe('true')
      expect(secret.metadata.labels['openeverest.io/monitoring-class']).toBe(monitoringClass)
      expect(secret.metadata.labels['openeverest.io/definition']).toBe('ServiceAccountToken')
      expect(secret.metadata.ownerReferences ?? []).toHaveLength(0)
      expect(secret.metadata.finalizers ?? []).toHaveLength(0)
    })

    await test.step('filter by monitoring class', async () => {
      const list = await th.listSecretsByMonitoringClass(request, monitoringClass, 'ServiceAccountToken')
      expect(list.items.map(s => s.metadata.name)).toContain(monitoringSecretName)

      const other = await th.listSecretsByMonitoringClass(request, 'non-existent', 'ServiceAccountToken')
      expect(other.items).toHaveLength(0)
    })
  })

  rejected.forEach(({desc, data, message}, i) => {
    test(`reject credentials when ${desc}`, async ({request}) => {
      const name = rejectedName(i)
      const response = await th.createSecretRaw(request, data(name))
      expect(response.status()).toBe(400)
      expect((await response.json()).message).toContain(message)

      expect((await th.getSecretRaw(request, name)).status()).toBe(404)
    })
  })
})

// A self-signed CA certificate whose private key was discarded.
const testCA = `-----BEGIN CERTIFICATE-----
MIIBrDCCAVOgAwIBAgIUejvO8eGXCMbLRlBEfeQEBiosadMwCgYIKoZIzj0EAwIw
IzEhMB8GA1UEAwwYb3BlbmV2ZXJlc3QgYXBpLXRlc3RzIENBMCAXDTI2MTEwMzE3
MDQxMloYDzIxMjYxMDEwMTcwNDEyWjAjMSEwHwYDVQQDDBhvcGVuZXZlcmVzdCBh
cGktdGVzdHMgQ0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQqq+Kp1HErWpL/
1n0yHhczVyLmSvTDwIyP1rEsT3UIs/qglJbhQV+fo8kJGGJQMS3Jv30eQuZuBdJ7
yk+IDeK3o2MwYTAdBgNVHQ4EFgQUb8ftw05oROj/5tA/eVtG+Twr6SEwHwYDVR0j
BBgwFoAUb8ftw05oROj/5tA/eVtG+Twr6SEwDwYDVR0TAQH/BAUwAwEB/zAOBgNV
HQ8BAf8EBAMCAgQwCgYIKoZIzj0EAwIDRwAwRAIgSIr4/5J7cFJB2zaDx34qdFlU
jmb44BMPZLYDV0lCycICIFsNRStO8nvLnG+ZVfaBva8Bfoj28ZJTEgQexukz41yl
-----END CERTIFICATE-----
`
const caBundleSecretName = 'backup-storage-test-minio-ca'

const caBundle = (name: string, data: Record<string, string>, labels: object = {'openeverest.io/definition': 'ca-bundle'}) => ({
  apiVersion: 'v1',
  kind: 'Secret',
  metadata: {name, labels},
  type: 'Opaque',
  stringData: data,
})

const rejectedCABundles = [
  {
    desc: 'a key holds no certificate',
    data: (name: string) => caBundle(name, {'ca.crt': 'not a certificate'}),
    message: 'key "ca.crt": no PEM certificate found',
  },
  {
    desc: 'a private key is pasted with the certificate',
    data: (name: string) => caBundle(name, {'ca.crt': `${testCA}-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n`}),
    message: 'holds a "PRIVATE KEY" PEM block',
  },
  {
    desc: 'another definition has no source label',
    data: (name: string) => caBundle(name, {'ca.crt': testCA}, {'openeverest.io/definition': 'trust-bundle'}),
    message: 'only "ca-bundle" needs none',
  },
]

const rejectedCABundleName = (i: number) => `backup-storage-test-rejected-${i}-ca`

test.describe.parallel('CA bundle Secrets', () => {
  test.afterAll(async ({request}) => {
    await th.deleteSecret(request, caBundleSecretName)
    for (let i = 0; i < rejectedCABundles.length; i++) {
      await th.deleteSecret(request, rejectedCABundleName(i))
    }
  })

  test('create, filter and get a CA bundle without a source label', async ({request}) => {
    const created = await th.createSecretWithData(request, caBundle(caBundleSecretName, {'ca.crt': testCA}))
    expect(created.metadata.name).toBe(caBundleSecretName)
    expect(created.stringData).toBeUndefined()

    const secret = await th.getSecret(request, caBundleSecretName)
    expect(secret.metadata.labels['openeverest.io/managed']).toBe('true')
    expect(secret.metadata.labels['openeverest.io/definition']).toBe('ca-bundle')

    const list = await th.listSecretsByDefinition(request, 'ca-bundle')
    expect(list.items.map(s => s.metadata.name)).toContain(caBundleSecretName)
  })

  rejectedCABundles.forEach(({desc, data, message}, i) => {
    test(`reject a CA bundle when ${desc}`, async ({request}) => {
      const name = rejectedCABundleName(i)
      const response = await th.createSecretRaw(request, data(name))
      expect(response.status()).toBe(400)
      expect((await response.json()).message).toContain(message)

      expect((await th.getSecretRaw(request, name)).status()).toBe(404)
    })
  })
})
