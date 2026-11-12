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
import type {CliHelper} from '@helpers/cliHelper';
import {
  EVEREST_CI_NAMESPACE,
  TEST_PASSWORD,
  TEST_USER,
  TIMEOUTS,
} from '@root/constants';
import * as th from '@tests/utils/api';

const url = `/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/monitoring-destinations`,
  instancesURL = `/v1/clusters/main/namespaces/${EVEREST_CI_NAMESPACE}/instances`,
  getDestination = async (request, name: string) => {
    const response = await request.get(`${url}/${name}`)

    await th.checkError(response)

    return (await response.json()) as {
      metadata: {labels?: Record<string, string>, annotations?: Record<string, string>, deletionTimestamp?: string},
      spec: {parameters?: Record<string, unknown>},
      status?: {attachedInstances?: number},
    }
  },
  getPresetSpec = async (request) => {
    const response = await request.get('/v1/clusters/main/cluster-instance-presets/test-preset')

    await th.checkError(response)

    return ((await response.json()) as {spec: Record<string, unknown>}).spec
  },
  rbacConfigMap = 'configmap/everest-rbac --namespace everest-system',
  getRBAC = async (cli: CliHelper) => {
    const output = await cli.exec(`kubectl get ${rbacConfigMap} -o json`)

    await output.assertSuccess()

    return (JSON.parse(output.stdout) as {data: Record<string, string>}).data
  },
  patchRBAC = async (cli: CliHelper, data: Record<string, string>) => {
    await (await cli.exec(`kubectl patch ${rbacConfigMap} --type merge -p '${JSON.stringify({data})}'`)).assertSuccess()
  },
  login = async (request, username: string, password: string) => {
    const response = await request.post('/v1/auth/token', {
      data: {
        grant_type: 'password',
        username,
        password,
      },
    })

    await th.checkError(response)

    return {Authorization: `Bearer ${(await response.json()).access_token}`}
  }

test.describe('Monitoring destination dryRun', () => {
  test('writes answer as they would and store nothing', async ({request}) => {
    const name = th.limitedSuffixedName('md-dry-run'),
      // The class need not exist: a destination is checked against it at reconcile time.
      payload = {metadata: {name}, spec: {classRef: {name: 'pmm'}}}

    try {
      await test.step('create stores nothing', async () => {
        const response = await request.post(`${url}?dryRun=All`, {data: payload})

        await th.checkError(response)
        expect((await response.json()).metadata.name).toBe(name)

        expect((await request.get(`${url}/${name}`)).status()).toBe(404)
      });

      await th.checkError(await request.post(url, {data: payload}))

      await test.step('update answers with the updated destination and stores nothing', async () => {
        const response = await request.put(`${url}/${name}?dryRun=All`, {
          data: {...payload, metadata: {name, labels: {team: 'dry-run'}}},
        })

        await th.checkError(response)
        expect((await response.json()).metadata.labels).toEqual({team: 'dry-run'})

        expect((await getDestination(request, name)).metadata.labels).toBeUndefined()
      });

      await test.step('delete removes nothing', async () => {
        const response = await request.delete(`${url}/${name}?dryRun=All`)

        expect(response.status()).toBe(204)

        expect((await getDestination(request, name)).metadata.deletionTimestamp).toBeUndefined()
      });
    } finally {
      await request.delete(`${url}/${name}`)
    }
  })
});

test.describe('Monitoring destination in use', () => {
  test.describe.configure({timeout: TIMEOUTS.ThreeMinutes});

  test('deleting it is refused, dry run or not, and changes nothing', async ({request}) => {
    const name = th.limitedSuffixedName('md-in-use'),
      instanceName = th.limitedSuffixedName('md-in-use-db'),
      inUse = 'in use by 1 database; remove the destination from them first',
      presetSpec = await getPresetSpec(request)

    try {
      // The class need not exist: a destination is in use as soon as an Instance lists it.
      await th.checkError(await request.post(url, {data: {metadata: {name}, spec: {classRef: {name: 'pmm'}}}}))
      await th.checkError(await request.post(instancesURL, {
        data: {
          metadata: {name: instanceName},
          spec: {...presetSpec, monitoring: {destinations: [{name: 'metrics', destinationRef: {name}}]}},
        },
      }))

      await expect(async () => {
        expect((await getDestination(request, name)).status?.attachedInstances).toBe(1)
      }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})

      for (const query of ['?dryRun=All', '']) {
        await test.step(`DELETE${query} is refused`, async () => {
          const response = await request.delete(`${url}/${name}${query}`)

          expect(response.status()).toBe(409)
          expect(await response.json()).toMatchObject({reason: 'InUse', message: expect.stringContaining(inUse)})

          expect(await getDestination(request, name)).toMatchObject({status: {attachedInstances: 1}})
          expect((await getDestination(request, name)).metadata.deletionTimestamp).toBeUndefined()
        });
      }
    } finally {
      await request.patch(`${instancesURL}/${instanceName}`, {
        headers: {'Content-Type': 'application/merge-patch+json'},
        data: {spec: {monitoring: null}},
      })
      // Allowed once the controller has counted the entry gone.
      await expect(async () => {
        expect([204, 404]).toContain((await request.delete(`${url}/${name}`)).status())
      }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})
      await request.delete(`${instancesURL}/${instanceName}`)
    }
  })

  test('deleting it right after the Instance listing it is let through', async ({request, cli}) => {
    const name = th.limitedSuffixedName('md-with-db'),
      instanceName = th.limitedSuffixedName('md-with-db-db'),
      // The test provider runs no controller; this finalizer keeps the deleted Instance as a provider's does while the engine tears down.
      setFinalizers = (finalizers: string[] | null) => cli.exec(
        `kubectl patch instances.core.openeverest.io ${instanceName} -n ${EVEREST_CI_NAMESPACE} --type=merge -p '${JSON.stringify({metadata: {finalizers}})}'`,
      ),
      presetSpec = await getPresetSpec(request)

    try {
      await th.checkError(await request.post(url, {data: {metadata: {name}, spec: {classRef: {name: 'pmm'}}}}))
      await th.checkError(await request.post(instancesURL, {
        data: {
          metadata: {name: instanceName},
          spec: {...presetSpec, monitoring: {destinations: [{name: 'metrics', destinationRef: {name}}]}},
        },
      }))
      await (await setFinalizers(['test.openeverest.io/hold'])).assertSuccess()

      await expect(async () => {
        expect((await getDestination(request, name)).status?.attachedInstances).toBe(1)
      }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})

      // What the UI does: delete the Instance, then its destination.
      await th.checkError(await request.delete(`${instancesURL}/${instanceName}`))

      await test.step('DELETE of the destination answers 204 within seconds', async () => {
        await expect(async () => {
          expect((await request.delete(`${url}/${name}`)).status()).toBe(204)
        }).toPass({intervals: [500], timeout: TIMEOUTS.TenSeconds})
      });

      await test.step('the deletion waits for the Instance, and the count says so', async () => {
        await expect(async () => {
          expect(await getDestination(request, name)).toMatchObject({
            metadata: {deletionTimestamp: expect.any(String)},
            status: {attachedInstances: 1},
          })
        }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})
      });

      await test.step('DELETE of the destination being deleted answers 204 again', async () => {
        expect((await request.delete(`${url}/${name}`)).status()).toBe(204)
      });

      await test.step('both go once the Instance does', async () => {
        await (await setFinalizers(null)).assertSuccess()

        await expect(async () => {
          expect((await request.get(`${instancesURL}/${instanceName}`)).status()).toBe(404)
          expect((await request.get(`${url}/${name}`)).status()).toBe(404)
        }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})
      });
    } finally {
      await setFinalizers(null)
      await request.delete(`${instancesURL}/${instanceName}`)
      await expect(async () => {
        expect([204, 404]).toContain((await request.delete(`${url}/${name}`)).status())
      }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})
    }
  })
});

test.describe('Monitoring destination Secret use', () => {
  test.describe.configure({timeout: TIMEOUTS.ThreeMinutes});

  test('changing spec needs use on its Secrets, changing annotations does not', async ({request, cli}) => {
    const name = th.limitedSuffixedName('md-use'),
      secret = `${name}-credentials`,
      role = `role:${name}`,
      // Neither the class nor the Secret need exist: the check is on the names.
      payload = {
        metadata: {name},
        spec: {
          classRef: {name: 'pmm'},
          authentication: {mode: 'ServiceAccountToken', credentialsSecretRef: {name: secret}},
          parameters: {url: 'https://pmm.example.com'},
        },
      },
      otherURL = {...payload, spec: {...payload.spec, parameters: {url: 'https://collector.example.com'}}},
      rbac = await getRBAC(cli)

    try {
      await th.checkError(await request.post(url, {data: payload}))

      // RBAC is off in the api-tests environment: turn it on, keeping the CI user's admin role, with a role that may
      // update the destination and read its Secret, but not use it.
      await patchRBAC(cli, {
        enabled: 'true',
        'policy.csv': [
          rbac['policy.csv'] ?? '',
          `p, ${role}, monitoring-destinations, read, main/${EVEREST_CI_NAMESPACE}/${name}`,
          `p, ${role}, monitoring-destinations, update, main/${EVEREST_CI_NAMESPACE}/${name}`,
          `p, ${role}, monitoring-classes, read, main/pmm`,
          `p, ${role}, secrets, read, main/${EVEREST_CI_NAMESPACE}/${secret}`,
          `g, ${TEST_USER}, ${role}`,
        ].join('\n'),
      })

      const asTestUser = await login(request, TEST_USER, TEST_PASSWORD)

      await test.step('changing parameters.url is refused', async () => {
        // The server reloads the policy asynchronously; until it has, the dry run is allowed and stores nothing.
        await expect(async () => {
          expect((await request.put(`${url}/${name}?dryRun=All`, {headers: asTestUser, data: otherURL})).status()).toBe(403)
        }).toPass({intervals: [1000], timeout: TIMEOUTS.ThirtySeconds})

        expect((await request.put(`${url}/${name}`, {headers: asTestUser, data: otherURL})).status()).toBe(403)

        expect((await getDestination(request, name)).spec.parameters).toEqual(payload.spec.parameters)
      });

      await test.step('changing annotations only is allowed', async () => {
        const response = await request.put(`${url}/${name}`, {
          headers: asTestUser,
          data: {...payload, metadata: {name, annotations: {'example.com/owner': 'team-a'}}},
        })

        await th.checkError(response)

        expect(await getDestination(request, name)).toMatchObject({
          metadata: {annotations: {'example.com/owner': 'team-a'}},
          spec: {parameters: payload.spec.parameters},
        })
      });
    } finally {
      await request.delete(`${url}/${name}`)
      await patchRBAC(cli, rbac)
    }
  })
});

test.describe('Monitoring destination filters', () => {
  test('?class= and ?secret= return what the field selectors select', async ({request}) => {
    const suffix = th.testPrefix().substring(0, 8),
      classA = `md-filter-a-${suffix}`,
      classB = `md-filter-b-${suffix}`,
      shared = `md-filter-shared-${suffix}`,
      corpCA = `md-filter-ca-${suffix}`,
      // Name, class, credentials Secret and CA bundle Secret. Neither the classes nor the Secrets need exist:
      // a destination is checked against them at reconcile time.
      destinations: [string, string, string?, string?][] = [
        [`md-filter-1-${suffix}`, classA, shared, corpCA],
        // Its credentials are the CA bundle of the first one.
        [`md-filter-2-${suffix}`, classA, corpCA],
        // The same Secret as credentials and CA bundle.
        [`md-filter-3-${suffix}`, classB, shared, shared],
        [`md-filter-4-${suffix}`, classB],
      ],
      [first, second, third, fourth] = destinations.map(([name]) => name),
      list = async (query: string) => {
        const response = await request.get(`${url}${query}`)

        await th.checkError(response)

        return ((await response.json()) as {items: {metadata: {name: string}, spec: {classRef: {name: string}}}[]}).items
      },
      names = async (query: string) => (await list(query)).map((d) => d.metadata.name).sort()

    try {
      for (const [name, classRef, credentials, ca] of destinations) {
        await th.checkError(await request.post(url, {
          data: {
            metadata: {name},
            spec: {
              classRef: {name: classRef},
              ...(credentials && {authentication: {mode: 'BasicAuth', credentialsSecretRef: {name: credentials}}}),
              ...(ca && {tls: {ca: {secretKeyRef: {name: ca, key: 'ca.crt'}}}}),
            },
          },
        }))
      }

      await test.step('?class= returns the destinations of the class, as filtering the whole list does', async () => {
        const all = await list('')

        for (const [classRef, want] of [[classA, [first, second]], [classB, [third, fourth]]] as const) {
          expect(await names(`?class=${classRef}`)).toEqual(want)
          expect(all.filter((d) => d.spec.classRef.name === classRef).map((d) => d.metadata.name).sort()).toEqual(want)
        }

        expect(await names(`?class=md-filter-none-${suffix}`)).toEqual([])
      });

      await test.step('?secret= returns the destinations referencing it as credentials or CA, each once', async () => {
        expect(await names(`?secret=${shared}`)).toEqual([first, third])
        expect(await names(`?secret=${corpCA}`)).toEqual([first, second])
        expect(await names(`?secret=md-filter-none-${suffix}`)).toEqual([])
      });

      await test.step('?class= and ?secret= together must both match', async () => {
        expect(await names(`?class=${classA}&secret=${shared}`)).toEqual([first])
        expect(await names(`?class=${classB}&secret=${shared}`)).toEqual([third])
        expect(await names(`?class=${classB}&secret=${corpCA}`)).toEqual([])
      });
    } finally {
      for (const [name] of destinations) {
        await request.delete(`${url}/${name}`)
      }
    }
  })
});
