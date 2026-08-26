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

import { vi, describe, it, expect, afterEach } from 'vitest';
import { render, waitFor, act } from '@testing-library/react';
import type { Extension } from '@openeverest/plugin-sdk';
import AuthContext from 'contexts/auth/auth.context';
import type {
  AuthContextProps,
  UserAuthStatus,
} from 'contexts/auth/auth.context.types';
import { PluginProvider, usePlugins } from './plugins.context';

// The loader resolves descriptor.bundleUrl with a dynamic import() evaluated
// inside plugins.context.tsx, so relative specifiers resolve against the
// fixtures sitting next to it. A non-existent path exercises the failure path.
const GOOD_PLUGIN = './__fixtures__/good-plugin';
const DEFAULT_EXPORT_PLUGIN = './__fixtures__/default-export-plugin';
const MISSING_PLUGIN = './__fixtures__/does-not-exist';

const AUTH_TOKEN = 'test-token';

vi.mock('api/session-token', () => ({
  getAuthToken: () => AUTH_TOKEN,
}));

interface ExtensionPointDescriptor {
  type: string;
  label?: string;
  icon?: string;
}

interface PluginDescriptor {
  name: string;
  displayName: string;
  bundleUrl: string;
  extensionPoints?: ExtensionPointDescriptor[];
}

const isSidebarItem = (
  ext: Extension
): ext is Extract<Extension, { type: 'sidebarItem' }> =>
  ext.type === 'sidebarItem';

function installFetch(descriptors: PluginDescriptor[], listOk = true) {
  const fetchMock = vi.fn((input: string) => {
    if (input === '/v1/plugins') {
      return Promise.resolve({
        ok: listOk,
        status: listOk ? 200 : 500,
        json: () => Promise.resolve(descriptors),
      });
    }
    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    });
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

function authValue(authStatus: UserAuthStatus): AuthContextProps {
  return {
    login: vi.fn(),
    logout: vi.fn(),
    setRedirectRoute: vi.fn(),
    authStatus,
    redirectRoute: null,
    isSsoEnabled: false,
  };
}

function renderPlugins(authStatus: UserAuthStatus = 'loggedIn') {
  const captured: { value: ReturnType<typeof usePlugins> } = {
    value: { plugins: [], loading: true },
  };
  const Capture = () => {
    captured.value = usePlugins();
    return null;
  };
  render(
    <AuthContext.Provider value={authValue(authStatus)}>
      <PluginProvider>
        <Capture />
      </PluginProvider>
    </AuthContext.Provider>
  );
  return captured;
}

describe('PluginProvider', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('registers only the extension types the CR declares', async () => {
    installFetch([
      {
        name: 'hub',
        displayName: 'Hub',
        bundleUrl: GOOD_PLUGIN,
        extensionPoints: [{ type: 'sidebarItem' }, { type: 'route' }],
      },
    ]);

    const captured = renderPlugins();
    await waitFor(() => expect(captured.value.loading).toBe(false));

    const hub = captured.value.plugins.find((p) => p.name === 'hub');
    expect(hub).toBeTruthy();
    if (!hub) return;

    // good-plugin also registers a globalDashboardWidget, which is not declared
    // in extensionPoints and must be dropped.
    expect(hub.extensions.map((e) => e.type)).toEqual(['sidebarItem', 'route']);
  });

  it('registers every extension when the CR declares no extensionPoints', async () => {
    installFetch([{ name: 'hub', displayName: 'Hub', bundleUrl: GOOD_PLUGIN }]);

    const captured = renderPlugins();
    await waitFor(() => expect(captured.value.loading).toBe(false));

    const hub = captured.value.plugins.find((p) => p.name === 'hub');
    expect(hub).toBeTruthy();
    if (!hub) return;

    expect(hub.extensions.map((e) => e.type)).toContain(
      'globalDashboardWidget'
    );
  });

  it('backfills a sidebarItem icon from the CR descriptor', async () => {
    installFetch([
      {
        name: 'hub',
        displayName: 'Hub',
        bundleUrl: GOOD_PLUGIN,
        extensionPoints: [
          {
            type: 'sidebarItem',
            label: 'Hub',
            icon: '/v1/plugins/hub/icon.png',
          },
          { type: 'route', label: 'Hub' },
        ],
      },
    ]);

    const captured = renderPlugins();
    await waitFor(() => expect(captured.value.loading).toBe(false));

    const hub = captured.value.plugins.find((p) => p.name === 'hub');
    expect(hub).toBeTruthy();
    if (!hub) return;

    const sidebar = hub.extensions.find(isSidebarItem);
    expect(sidebar?.icon).toBe('/v1/plugins/hub/icon.png');
  });

  it('scopes the plugin fetch through the host proxy with the auth header', async () => {
    const fetchMock = installFetch([
      {
        name: 'hub',
        displayName: 'Hub',
        bundleUrl: GOOD_PLUGIN,
        extensionPoints: [{ type: 'sidebarItem' }, { type: 'route' }],
      },
    ]);

    const captured = renderPlugins();
    await waitFor(() => expect(captured.value.loading).toBe(false));

    expect(fetchMock).toHaveBeenCalledWith('/v1/plugins', {
      headers: { Authorization: `Bearer ${AUTH_TOKEN}` },
    });
    expect(fetchMock).toHaveBeenCalledWith(
      '/v1/plugins/hub/context',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: `Bearer ${AUTH_TOKEN}`,
          'X-Test': '1',
        }),
      })
    );
  });

  it('isolates a failing plugin so the others still load', async () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    installFetch([
      { name: 'broken', displayName: 'Broken', bundleUrl: MISSING_PLUGIN },
      {
        name: 'hub',
        displayName: 'Hub',
        bundleUrl: GOOD_PLUGIN,
        extensionPoints: [{ type: 'sidebarItem' }, { type: 'route' }],
      },
    ]);

    const captured = renderPlugins();
    await waitFor(() => expect(captured.value.loading).toBe(false));

    const hub = captured.value.plugins.find((p) => p.name === 'hub');
    expect(hub).toBeTruthy();
    if (!hub) return;

    expect(hub.extensions.map((e) => e.type)).toEqual(['sidebarItem', 'route']);
    expect(errorSpy).toHaveBeenCalled();
  });

  it('accepts a plugin whose register function is the default export', async () => {
    installFetch([
      {
        name: 'def',
        displayName: 'Default',
        bundleUrl: DEFAULT_EXPORT_PLUGIN,
        extensionPoints: [{ type: 'sidebarItem' }],
      },
    ]);

    const captured = renderPlugins();
    await waitFor(() => expect(captured.value.loading).toBe(false));

    const def = captured.value.plugins.find((p) => p.name === 'def');
    expect(def?.extensions.map((e) => e.type)).toEqual(['sidebarItem']);
  });

  it('does not load plugins until the user is logged in', async () => {
    const fetchMock = installFetch([]);

    const captured = renderPlugins('loggingIn');
    await act(async () => {
      await Promise.resolve();
    });

    expect(fetchMock).not.toHaveBeenCalled();
    expect(captured.value.loading).toBe(true);
  });

  it('resolves with no plugins when discovery fails', async () => {
    installFetch([], false);

    const captured = renderPlugins();
    await waitFor(() => expect(captured.value.loading).toBe(false));

    expect(captured.value.plugins).toEqual([]);
  });
});
