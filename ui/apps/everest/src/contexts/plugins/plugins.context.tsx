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

import React, {
  createContext,
  useContext,
  useEffect,
  useState,
  ReactNode,
} from 'react';
import type {
  Extension,
  PluginApi,
  PluginRegisterFn,
} from '@openeverest/plugin-sdk';
import AuthContext from 'contexts/auth/auth.context';
import { getAuthToken } from 'api/session-token';

export interface PluginRegistration {
  name: string;
  extensions: Extension[];
}

interface ExtensionPointDescriptor {
  type: string;
  label?: string;
  path?: string;
  icon?: string;
  providers?: string[];
}

interface PluginDescriptor {
  name: string;
  displayName: string;
  bundleUrl: string;
  compatibleHostVersions?: string;
  extensionPoints?: ExtensionPointDescriptor[];
}

interface PluginContextValue {
  plugins: PluginRegistration[];
  loading: boolean;
}

const PluginContext = createContext<PluginContextValue>({
  plugins: [],
  loading: true,
});

export const usePlugins = () => useContext(PluginContext);

// Coarse UI contract tied to the shared React major (see issue #2661). Plugins
// bundle their own MUI, so React is the only runtime they share with the host.
const UI_CONTRACT_VERSION = React.version.split('.')[0];

function getHostVersion(): string {
  return (
    document
      .querySelector("meta[name='everest-version']")
      ?.getAttribute('content') || 'dev'
  );
}

function getCSPNonce(): string {
  return (
    document
      .querySelector("meta[name='csp-nonce']")
      ?.getAttribute('content') || ''
  );
}

// Minimal semver-range satisfaction for the common comparators used in the
// Plugin CR's compatibleHostVersions (">=2.0.0 <3.0.0", "^2.1.0", "~2.1.0").
// Returns true when the range is empty/unparseable so an unknown range never
// silently blocks a plugin (advisory contract, first-party ecosystem).
function satisfiesHostVersion(version: string, range?: string): boolean {
  if (!range || version === 'dev') {
    return true;
  }
  const parse = (v: string): [number, number, number] | null => {
    const m = v.match(/^(\d+)\.(\d+)\.(\d+)/);
    return m ? [Number(m[1]), Number(m[2]), Number(m[3])] : null;
  };
  const host = parse(version);
  if (!host) {
    return true;
  }
  const cmp = (a: [number, number, number], b: [number, number, number]) =>
    a[0] - b[0] || a[1] - b[1] || a[2] - b[2];

  const evalClause = (clause: string): boolean => {
    const comparators = clause.trim().split(/\s+/).filter(Boolean);
    if (comparators.length === 0) {
      return true;
    }
    return comparators.every((c) => {
      const m = c.match(/^(>=|<=|>|<|=|\^|~)?(\d+\.\d+\.\d+)/);
      if (!m) {
        return true;
      }
      const op = m[1] || '=';
      const target = parse(m[2])!;
      if (op === '^') {
        const upper: [number, number, number] = [target[0] + 1, 0, 0];
        return cmp(host, target) >= 0 && cmp(host, upper) < 0;
      }
      if (op === '~') {
        const upper: [number, number, number] = [target[0], target[1] + 1, 0];
        return cmp(host, target) >= 0 && cmp(host, upper) < 0;
      }
      const c0 = cmp(host, target);
      if (op === '>=') return c0 >= 0;
      if (op === '<=') return c0 <= 0;
      if (op === '>') return c0 > 0;
      if (op === '<') return c0 < 0;
      return c0 === 0;
    });
  };

  return range
    .split('||')
    .some((clause) => evalClause(clause));
}

// Build the PluginApi object that the host passes to each plugin's register().
// When allowedTypes is provided, only extensions whose type is in the set will be registered.
function createPluginApi(
  pluginName: string,
  registrations: PluginRegistration[],
  allowedTypes?: Set<string>
): PluginApi {
  const registration: PluginRegistration = { name: pluginName, extensions: [] };
  registrations.push(registration);

  return {
    React,

    registerExtension(extension: Extension) {
      // If the Plugin CR declares extensionPoints, only allow those types through.
      if (allowedTypes && !allowedTypes.has(extension.type)) {
        return;
      }
      registration.extensions.push(extension);
    },

    fetch(path: string, init?: RequestInit): Promise<Response> {
      const headers: Record<string, string> = {
        ...getAuthHeaders(),
        ...(init?.headers as Record<string, string>),
      };
      const url = `/v1/plugins/${pluginName}${path}`;
      return window.fetch(url, { ...init, headers });
    },

    cssNonce: getCSPNonce(),
    hostVersion: getHostVersion(),
    uiContractVersion: UI_CONTRACT_VERSION,
  };
}

function getAuthHeaders(): Record<string, string> {
  const token = getAuthToken();
  if (token) {
    return { Authorization: `Bearer ${token}` };
  }
  return {};
}

async function loadPluginDescriptors(): Promise<PluginDescriptor[]> {
  try {
    const resp = await window.fetch('/v1/plugins', {
      headers: getAuthHeaders(),
    });
    if (!resp.ok) return [];
    return await resp.json();
  } catch {
    return [];
  }
}

export const PluginProvider = ({ children }: { children: ReactNode }) => {
  const [plugins, setPlugins] = useState<PluginRegistration[]>([]);
  const [loading, setLoading] = useState(true);
  const { authStatus } = useContext(AuthContext);

  useEffect(() => {
    if (authStatus !== 'loggedIn') {
      return;
    }

    let cancelled = false;

    (async () => {
      const descriptors = await loadPluginDescriptors();
      const registrations: PluginRegistration[] = [];

      for (const descriptor of descriptors) {
        try {
          // Reject incompatible plugins loudly rather than rendering a detached
          // or unthemed UI (see issue #2661).
          if (
            !satisfiesHostVersion(
              getHostVersion(),
              descriptor.compatibleHostVersions
            )
          ) {
            // eslint-disable-next-line no-console
            console.error(
              `[plugins] Skipping "${descriptor.name}": requires host ${descriptor.compatibleHostVersions}, host is ${getHostVersion()}.`
            );
            continue;
          }
          const mod = await import(/* @vite-ignore */ descriptor.bundleUrl);
          const registerFn: PluginRegisterFn = mod.default || mod.register;
          if (typeof registerFn === 'function') {
            // Build the allowed extension types set from the CR's declared extensionPoints.
            const allowedTypes = descriptor.extensionPoints?.length
              ? new Set(descriptor.extensionPoints.map((ep) => ep.type))
              : undefined;
            const pluginApi = createPluginApi(
              descriptor.name,
              registrations,
              allowedTypes
            );
            registerFn(pluginApi);

            // Forward icon from the CRD descriptor into registered sidebarItem extensions.
            // The plugin bundle may not include the icon, so we fill it in from the descriptor.
            // The backend already resolves relative paths to full proxy URLs.
            const registration = registrations[registrations.length - 1];
            if (registration && descriptor.extensionPoints?.length) {
              for (const ext of registration.extensions) {
                if (ext.type === 'sidebarItem' && !ext.icon) {
                  const match = descriptor.extensionPoints.find(
                    (ep) => ep.type === 'sidebarItem' && ep.label === ext.label
                  );
                  if (match?.icon) {
                    ext.icon = match.icon;
                  }
                }
              }
            }
          }
        } catch (err) {
          // eslint-disable-next-line no-console
          console.error(
            `[plugins] Failed to load plugin "${descriptor.name}":`,
            err
          );
        }
      }

      if (!cancelled) {
        setPlugins(registrations);
        setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [authStatus]);

  return (
    <PluginContext.Provider value={{ plugins, loading }}>
      {children}
    </PluginContext.Provider>
  );
};
