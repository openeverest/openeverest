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
import {CliHelper} from '@helpers/cliHelper'
import {APIRequestContext, APIResponse} from '@playwright/test'
import {EVEREST_CI_NAMESPACE, TIMEOUTS} from '@root/constants';
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
    desc: 'the certificates are not under ca.crt',
    data: (name: string) => caBundle(name, {'bundle.pem': testCA}),
    message: 'missing key "ca.crt"',
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

test.describe('Secret dryRun', () => {
  test('writes answer as they would and store nothing', async ({request}) => {
    const url = `/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/secrets`,
      name = th.suffixedName('secret-dry-run'),
      payload = monitoringSecret(name, 'ServiceAccountToken')

    try {
      await test.step('create stores nothing', async () => {
        const response = await request.post(`${url}?dryRun=All`, {data: payload}),
          created = await response.json()

        expect(response.status()).toBe(201)
        expect(created.metadata.name).toBe(name)
        expect(created.data).toBeUndefined()

        expect((await th.getSecretRaw(request, name)).status()).toBe(404)
      })

      await test.step('create checks the Secret against its definition', async () => {
        const response = await request.post(`${url}?dryRun=All`, {
          data: monitoringSecret(name, 'ServiceAccountToken', {stringData: {orgID: '1'}}),
        })

        expect(response.status()).toBe(400)
        expect((await response.json()).message).toContain('missing key "token"')
      })

      await th.createSecretWithData(request, payload)

      await test.step('delete removes nothing', async () => {
        const response = await request.delete(`${url}/${name}?dryRun=All`)

        expect(response.status()).toBe(204)

        expect((await th.getSecret(request, name)).metadata.deletionTimestamp).toBeUndefined()
      })
    } finally {
      await th.deleteSecret(request, name)
    }
  })
})

test.describe('Secret patch', () => {
  test('sets and removes data keys and keeps everything else', async ({request}) => {
    const url = `/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/secrets`,
      name = th.suffixedName('secret-patch'),
      patch = async (target: string, data: object, contentType = 'application/merge-patch+json') =>
        await request.patch(`${url}/${target}`, {data, headers: {'Content-Type': contentType}})

    try {
      await th.createSecretWithData(request, monitoringSecret(name, 'ServiceAccountToken', {
        stringData: {token: 'glsa_test-token', orgID: '1'},
      }))
      let version = (await th.getCountedSecret(request, name)).metadata.resourceVersion

      await test.step('dry run changes nothing', async () => {
        const response = await request.patch(`${url}/${name}?dryRun=All`, {
          data: {metadata: {resourceVersion: version}, stringData: {token: 'glsa_rotated'}},
          headers: {'Content-Type': 'application/merge-patch+json'},
        })

        expect(response.status()).toBe(200)
        expect((await th.getSecret(request, name)).metadata.resourceVersion).toBe(version)
      })

      await test.step('a patch without a resourceVersion is refused', async () => {
        const response = await patch(name, {stringData: {token: 'glsa_rotated'}})

        expect(response.status()).toBe(400)
        expect((await response.json()).message).toContain('metadata.resourceVersion is required')
      })

      await test.step('a patch sets the keys it sends and keeps the others', async () => {
        // Without token the Secret would miss a required key, so this passes only if token is kept.
        const response = await patch(name, {metadata: {resourceVersion: version}, stringData: {orgID: '2'}}),
          patched = await response.json()

        expect(response.status()).toBe(200)
        expect(patched.data).toBeUndefined()
        expect(patched.stringData).toBeUndefined()
        expect(patched.metadata.resourceVersion).not.toBe(version)
        expect(patched.metadata.labels).toMatchObject({
          'openeverest.io/managed': 'true',
          'openeverest.io/monitoring-class': monitoringClass,
          'openeverest.io/definition': 'ServiceAccountToken',
        })
      })

      await test.step('a stale resourceVersion is refused', async () => {
        const response = await patch(name, {metadata: {resourceVersion: version}, stringData: {token: 'glsa_rotated'}})

        expect(response.status()).toBe(409)
        version = (await th.getSecret(request, name)).metadata.resourceVersion
      })

      await test.step('removing a required key is refused', async () => {
        const response = await patch(name, {metadata: {resourceVersion: version}, data: {token: null}})

        expect(response.status()).toBe(400)
        expect((await response.json()).message).toContain('missing key "token"')
      })

      await test.step('a Secret returned by GET changes nothing when sent back', async () => {
        const current = await th.getSecret(request, name),
          response = await patch(name, current as object)

        expect(response.status()).toBe(200)
        expect((await th.getSecret(request, name)).metadata.resourceVersion).toBe(current.metadata.resourceVersion)
      })

      await test.step('the definition labels cannot change', async () => {
        const response = await patch(name, {
          metadata: {
            resourceVersion: version,
            labels: {'openeverest.io/monitoring-class': monitoringClass, 'openeverest.io/definition': 'BasicAuth'},
          },
        })

        expect(response.status()).toBe(400)
        expect((await response.json()).message).toContain('differs from the stored Secret')
      })

      await test.step('a member a Secret does not have is refused', async () => {
        const response = await patch(name, {metadata: {resourceVersion: version}, stringdata: {token: 'glsa_rotated'}})

        expect(response.status()).toBe(400)
        expect((await response.json()).message).toContain('unknown field "stringdata"')
      })

      await test.step('only a merge patch is accepted', async () => {
        const response = await patch(name, {metadata: {resourceVersion: version}}, 'application/json')

        expect(response.status()).toBe(415)
      })

      await test.step('a missing Secret is not found', async () => {
        const response = await patch(th.suffixedName('secret-missing'), {
          metadata: {resourceVersion: '1'},
          stringData: {token: 'glsa_rotated'},
        })

        expect(response.status()).toBe(404)
      })
    } finally {
      await th.deleteSecret(request, name)
    }
  })
})

test.describe('Secret responses', () => {
  test('leave out the metadata that can hold or reveal the data', async ({request}) => {
    const name = th.suffixedName('secret-view'),
      // What client-side kubectl apply and External Secrets write; only the openeverest.io annotation may come back.
      annotations = {
        'kubectl.kubernetes.io/last-applied-configuration': JSON.stringify({stringData: {token: 'glsa_test-token'}}),
        'reconcile.external-secrets.io/data-hash': 'glsa_hash',
        'openeverest.io/note': 'kept',
      },
      expectView = (response: unknown) => {
        const secret = response as {metadata: {annotations?: Record<string, string>, managedFields?: object}},
          // Core's usage counts are openeverest.io annotations too.
          kept = Object.fromEntries(Object.entries(secret.metadata.annotations ?? {})
            .filter(([key]) => !th.secretUsageCountKeys.includes(key)))

        expect(kept).toEqual({'openeverest.io/note': 'kept'})
        expect(secret.metadata.managedFields).toBeUndefined()
        expect(JSON.stringify(secret)).not.toContain('glsa_')
      }

    try {
      await test.step('create', async () => {
        const secret = monitoringSecret(name, 'ServiceAccountToken')

        expectView(await th.createSecretWithData(request, {...secret, metadata: {...secret.metadata, annotations}}))
      })

      await test.step('get', async () => {
        const secret = await th.getCountedSecret(request, name)

        expectView(secret)
        expect(secret.metadata.annotations).toMatchObject({'monitoring.openeverest.io/attached-destinations': '0'})
      })

      await test.step('list', async () => {
        const list = await th.listSecretsByMonitoringClass(request, monitoringClass, 'ServiceAccountToken')

        expectView(list.items.find(s => s.metadata.name === name))
      })

      await test.step('patch', async () => {
        const version = (await th.getCountedSecret(request, name)).metadata.resourceVersion,
          response = await request.patch(`/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/secrets/${name}`, {
            data: {metadata: {resourceVersion: version}, stringData: {token: 'glsa_rotated'}},
            headers: {'Content-Type': 'application/merge-patch+json'},
          })

        expect(response.status()).toBe(200)
        expectView(await response.json())
      })
    } finally {
      await th.deleteSecret(request, name)
    }
  })
})

test.describe.parallel('Secrets written outside the API', () => {
  const url = `/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/secrets`,
    // Writes to the cluster directly, as kubectl, Git or a secret manager would, in the namespace the API is tested on.
    kubectl = async (cli: CliHelper, args: string, stdin?: object) => {
      const command = `kubectl -n ${EVEREST_CI_NAMESPACE} ${args}`,
        output = await cli.execSilent(stdin ? `echo '${JSON.stringify(stdin)}' | ${command}` : command)

      await output.assertSuccess()

      return output.stdout.trim()
    },
    rotate = async (request: APIRequestContext, name: string, resourceVersion = '1') =>
      await request.patch(`${url}/${name}`, {
        data: {metadata: {resourceVersion}, stringData: {token: 'glsa_rotated'}},
        headers: {'Content-Type': 'application/merge-patch+json'},
      }),
    names = (list: unknown) => (list as {items: {metadata: {name: string}}[]}).items.map((s) => s.metadata.name)

  test('a labelled Secret applied with kubectl is shown and can be deleted, not patched', async ({request, cli}) => {
    const name = th.suffixedName('secret-applied')

    try {
      await kubectl(cli, 'apply -f -', monitoringSecret(name, 'ServiceAccountToken'))

      await test.step('listed and readable without openeverest.io/managed', async () => {
        const list = await th.listSecretsByMonitoringClass(request, monitoringClass, 'ServiceAccountToken'),
          secret = await th.getSecret(request, name)

        expect(names(list)).toContain(name)
        expect(secret.metadata.labels['openeverest.io/managed']).toBeUndefined()
        expect(secret.metadata.labels['openeverest.io/definition']).toBe('ServiceAccountToken')
        expect(secret.data).toBeUndefined()
      })

      await test.step('a patch is refused: it is changed where it was written', async () => {
        const version = (await th.getCountedSecret(request, name)).metadata.resourceVersion as string,
          response = await rotate(request, name, version)

        expect(response.status()).toBe(409)
        expect(await response.json()).toMatchObject({
          reason: 'ManagedOutside',
          message: 'managed outside OpenEverest; change it at its source',
        })
        expect((await th.getSecret(request, name)).metadata.resourceVersion).toBe(version)
      })

      await test.step('delete removes it', async () => {
        expect((await th.deleteSecretRaw(request, name)).status()).toBe(204)
        expect((await th.getSecretRaw(request, name)).status()).toBe(404)
      })
    }
    finally {
      await kubectl(cli, `delete secret ${name} --ignore-not-found`)
    }
  })

  test('a Secret a controller owns is shown and neither patched nor deleted', async ({request, cli}) => {
    const name = th.suffixedName('secret-owned'),
      owner = `${name}-owner`,
      secret = monitoringSecret(name, 'ServiceAccountToken')

    try {
      await kubectl(cli, `create configmap ${owner}`)

      const ownerReference = {
        apiVersion: 'v1',
        kind: 'ConfigMap',
        name: owner,
        uid: await kubectl(cli, `get configmap ${owner} -o jsonpath={.metadata.uid}`),
        controller: true,
      }

      // Labelled as created through the API too, so the controller owner alone refuses the writes.
      await kubectl(cli, 'apply -f -', {
        ...secret,
        metadata: {
          ...secret.metadata,
          labels: {...secret.metadata.labels, 'openeverest.io/managed': 'true'},
          ownerReferences: [ownerReference],
        },
      })

      await test.step('readable, with its owner', async () => {
        expect((await th.getSecret(request, name)).metadata.ownerReferences).toEqual([ownerReference])
      })

      await test.step('a patch is refused', async () => {
        const version = (await th.getSecret(request, name)).metadata.resourceVersion as string,
          response = await rotate(request, name, version)

        expect(response.status()).toBe(409)
        expect(await response.json()).toMatchObject({reason: 'ManagedByController', message: `managed by ConfigMap/${owner}`})
      })

      await test.step('a delete is refused: the controller would recreate it', async () => {
        const response = await th.deleteSecretRaw(request, name)

        expect(response.status()).toBe(409)
        expect(await response.json()).toMatchObject({reason: 'ManagedByController', message: `managed by ConfigMap/${owner}`})
        expect((await th.getSecret(request, name)).metadata.deletionTimestamp).toBeUndefined()
      })
    }
    finally {
      await kubectl(cli, `delete secret ${name} --ignore-not-found`)
      await kubectl(cli, `delete configmap ${owner} --ignore-not-found`)
    }
  })

  test('an unlabelled Secret answers as a missing one', async ({request, cli}) => {
    const name = th.suffixedName('secret-unlabelled'),
      missing = th.suffixedName('secret-missing'),
      // What the API answers for a Secret, with the name made generic.
      answers = async (secret: string) => await Promise.all([
        th.getSecretRaw(request, secret),
        rotate(request, secret),
        th.deleteSecretRaw(request, secret),
      ].map(async (pending: Promise<APIResponse>) => {
        const response = await pending,
          {message} = (await response.json()) as {message: string}

        return {status: response.status(), message: message.replace(secret, '<name>')}
      }))

    try {
      // openeverest.io/managed alone does not make a Secret visible.
      await kubectl(cli, 'apply -f -', {
        apiVersion: 'v1',
        kind: 'Secret',
        metadata: {name, labels: {'openeverest.io/managed': 'true'}},
        type: 'Opaque',
        stringData: {token: 'glsa_test-token'},
      })

      const unlabelled = await answers(name)

      expect(unlabelled.map((answer) => answer.status)).toEqual([404, 404, 404])
      expect(unlabelled).toEqual(await answers(missing))
      expect(names(await th.listSecrets(request))).not.toContain(name)

      await kubectl(cli, `get secret ${name}`)
    }
    finally {
      await kubectl(cli, `delete secret ${name} --ignore-not-found`)
    }
  })
})

test.describe.parallel('Secrets in use', () => {
  test.describe.configure({timeout: TIMEOUTS.ThreeMinutes})

  const url = `/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/secrets`,
    destinationsURL = `/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/monitoring-destinations`,
    // The class need not exist: a destination uses the Secrets it names as soon as it is created.
    createDestination = async (request: APIRequestContext, name: string, ca: string) => {
      await th.checkError(await request.post(destinationsURL, {
        data: {metadata: {name}, spec: {classRef: {name: 'pmm'}, tls: {ca: {secretKeyRef: {name: ca, key: 'ca.crt'}}}}},
      }))
    },
    deleteDestination = async (request: APIRequestContext, name: string) => {
      expect([204, 404]).toContain((await request.delete(`${destinationsURL}/${name}`)).status())
    }

  test('core counts the destinations that use a Secret', async ({request}) => {
    const name = th.suffixedName('secret-counted-ca'),
      destination = th.limitedSuffixedName('secret-counted-md'),
      destinations = async () => {
        const secret = (await th.getSecret(request, name)) as {metadata: {annotations?: Record<string, string>}}

        return secret.metadata.annotations?.['monitoring.openeverest.io/attached-destinations']
      },
      poll = {intervals: [500], timeout: TIMEOUTS.ThirtySeconds}

    try {
      await th.createSecretWithData(request, caBundle(name, {'ca.crt': testCA}))

      await test.step('an unused Secret counts 0', async () => {
        await expect.poll(destinations, poll).toBe('0')
      })

      await test.step('a destination that names it counts', async () => {
        await createDestination(request, destination, name)
        await expect.poll(destinations, poll).toBe('1')
      })

      await test.step('a destination deleted stops counting', async () => {
        await deleteDestination(request, destination)
        await expect.poll(destinations, poll).toBe('0')
      })
    } finally {
      await deleteDestination(request, destination)
      await th.deleteSecret(request, name)
    }
  })

  test('deleting a Secret in use is refused with the counts, dry run or not', async ({request}) => {
    const name = th.suffixedName('secret-in-use-ca'),
      destination = th.limitedSuffixedName('secret-in-use-md')

    try {
      await th.createSecretWithData(request, caBundle(name, {'ca.crt': testCA}))
      await createDestination(request, destination, name)
      await th.getCountedSecret(request, name, '1')

      for (const query of ['?dryRun=All', '']) {
        await test.step(`DELETE${query} is refused`, async () => {
          const response = await request.delete(`${url}/${name}${query}`)

          expect(response.status()).toBe(409)
          expect(await response.json()).toMatchObject({
            reason: 'InUse',
            message: 'in use by 1 monitoring destination(s) and 0 backup storage(s); replace its values instead, ' +
              `or point them at another Secret first (kubectl get monitoringdestinations,backupstorages -n ${EVEREST_CI_NAMESPACE} ` +
              `--field-selector spec.tls.ca.secretKeyRef.name=${name})`,
          })
          expect((await th.getSecret(request, name)).metadata.deletionTimestamp).toBeUndefined()
        })
      }

      await test.step('DELETE is let through once the destination is deleted', async () => {
        await deleteDestination(request, destination)

        // Core counts the deletion within moments; until it has, the answer is still 409.
        await expect(async () => {
          expect((await request.delete(`${url}/${name}`)).status()).toBe(204)
        }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})
        await expect(async () => {
          expect((await th.getSecretRaw(request, name)).status()).toBe(404)
        }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})
      })
    } finally {
      await deleteDestination(request, destination)
      await th.deleteSecret(request, name)
    }
  })
})
