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
import { satisfiesHostVersion, satisfiesUiContract } from './plugin-version';
import { useClusterName } from 'hooks/api/useClusterName';

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
  // API-compatibility gate: the host application semver range the plugin supports.
  compatibleHostVersions?: string;
  // UI-contract gate: the React-major semver range the plugin's frontend supports.
  compatibleUiContractVersions?: string;
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

// The UI contract a plugin shares with the host is the React major (issue
// #2661): plugins bundle their own MUI, so React is the only shared runtime.
// Exposed to plugins as `uiContractVersion` and enforced as the UI gate below.
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
    document.querySelector("meta[name='csp-nonce']")?.getAttribute('content') ||
    ''
  );
}

// Build the PluginApi object that the host passes to each plugin's register().
// When allowedTypes is provided, only extensions whose type is in the set will be registered.
function createPluginApi(
  clusterName: string,
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
      const url = `/v1/clusters/${clusterName}/plugins/${pluginName}${path}`;
      return window.fetch(url, { ...init, headers });
    },

    basePath: `/v1/clusters/${clusterName}/plugins/${pluginName}`,
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

async function loadPluginDescriptors(
  clusterName: string
): Promise<PluginDescriptor[]> {
  try {
    const resp = await window.fetch(`/v1/clusters/${clusterName}/plugins`, {
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
  const clusterName = useClusterName();

  useEffect(() => {
    if (authStatus !== 'loggedIn') {
      return;
    }

    let cancelled = false;

    (async () => {
      const descriptors = await loadPluginDescriptors(clusterName);
      const registrations: PluginRegistration[] = [];

      for (const descriptor of descriptors) {
        try {
          // Two independent gates, each guarding its own axis (see issue #2661).
          // UI gate: the React major is the only runtime a bundled-MUI plugin
          // shares with the host, so a mismatch is what actually breaks it.
          if (
            !satisfiesUiContract(
              React.version,
              descriptor.compatibleUiContractVersions
            )
          ) {
            // eslint-disable-next-line no-console
            console.error(
              `[plugins] Skipping "${descriptor.name}": needs UI contract (React) ${descriptor.compatibleUiContractVersions}, host is React ${UI_CONTRACT_VERSION}.`
            );
            continue;
          }
          // API gate: the host application version is a separate, coarser axis
          // (it bumps for backend reasons unrelated to the UI runtime).
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
              clusterName,
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
  }, [authStatus, clusterName]);

  return (
    <PluginContext.Provider value={{ plugins, loading }}>
      {children}
    </PluginContext.Provider>
  );
};
