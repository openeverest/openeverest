// everest
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

// Smoke test for the plugin subsystem. OpenEverest now ships plugin-hub as a
// plugin by default, so we guard the end-to-end server wiring that every
// plugin depends on: discovery (`GET /v1/plugins`), the reverse proxy
// (`GET /v1/plugins/:name/*`), and the enabled/unknown gating. The test is
// self-contained — it creates its own Plugin CR and does not depend on
// plugin-hub being deployed in the test cluster.

import {expect, test} from '@fixtures'
import {checkError} from '@tests/utils/api';

const PLUGIN_NAME = 'test-api-plugin'
const PLUGIN_MANIFEST = 'manifests/test-plugin.yaml'

interface PluginDescriptor {
  name: string
  displayName?: string
  bundleUrl?: string
  extensionPoints?: { type: string }[]
}

const findPlugin = (plugins: PluginDescriptor[], name: string): PluginDescriptor | undefined =>
  Array.isArray(plugins) ? plugins.find((p) => p?.name === name) : undefined

test.describe.serial('Plugin subsystem', () => {
  test.beforeAll(async ({cli}) => {
    await (await cli.exec(`kubectl apply -f ${PLUGIN_MANIFEST}`)).assertSuccess()
  })

  test.afterAll(async ({cli}) => {
    await cli.exec(`kubectl delete -f ${PLUGIN_MANIFEST} --ignore-not-found=true`)
  })

  test('discovery lists the plugin with a resolved descriptor', async ({request}) => {
    const r = await request.get('/v1/plugins')

    await checkError(r)

    const plugins: PluginDescriptor[] = await r.json()
    const plugin = findPlugin(plugins, PLUGIN_NAME)

    expect(plugin, `${PLUGIN_NAME} should be listed by /v1/plugins`).toBeTruthy()
    if (!plugin) return

    expect(plugin.displayName).toEqual('Test API Plugin')
    // The server resolves the bundle path into the proxy URL the UI imports.
    expect(plugin.bundleUrl).toEqual(`/v1/plugins/${PLUGIN_NAME}/main.js`)
    const extensionTypes = (plugin.extensionPoints ?? []).map((ep) => ep.type)

    expect(extensionTypes).toContain('sidebarItem')
    expect(extensionTypes).toContain('route')
  })

  test('proxy routes an enabled plugin to its backend', async ({request}) => {
    // The backend points at a closed port, so a successful route to the proxy
    // surfaces as a 502 (backend dial failed) rather than a 404 (unknown
    // plugin). This proves the request reached the proxy and resolved the
    // backend for an enabled plugin.
    const r = await request.get(`/v1/plugins/${PLUGIN_NAME}/main.js`)

    expect(r.status()).toBe(502)
  })

  test('proxy returns 404 for an unknown plugin', async ({request}) => {
    const r = await request.get('/v1/plugins/does-not-exist/main.js')

    expect(r.status()).toBe(404)
  })

  test('disabled plugin is hidden from discovery and the proxy', async ({request, cli}) => {
    await (await cli.exec(
      `kubectl patch plugin ${PLUGIN_NAME} --type merge -p '{"spec":{"enabled":false}}'`
    )).assertSuccess()

    try {
      await expect(async () => {
        const list = await request.get('/v1/plugins')

        await checkError(list)

        const plugins: PluginDescriptor[] = await list.json()
        expect(findPlugin(plugins, PLUGIN_NAME)).toBeUndefined()

        const proxied = await request.get(`/v1/plugins/${PLUGIN_NAME}/main.js`)

        expect(proxied.status()).toBe(404)
      }).toPass({timeout: 15_000})
    } finally {
      await (await cli.exec(
        `kubectl patch plugin ${PLUGIN_NAME} --type merge -p '{"spec":{"enabled":true}}'`
      )).assertSuccess()
    }
  })
})
