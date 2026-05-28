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

import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import type { Extension, PluginApi, PluginRegisterFn } from '@openeverest/plugin-sdk';
import AuthContext from 'contexts/auth/auth.context';

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

// Build the PluginApi object that the host passes to each plugin's register().
// When allowedTypes is provided, only extensions whose type is in the set will be registered.
function createPluginApi(
  pluginName: string,
  registrations: PluginRegistration[],
  allowedTypes?: Set<string>,
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
  };
}

function getAuthHeaders(): Record<string, string> {
  const token = localStorage.getItem('everestToken');
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
          const mod = await import(/* @vite-ignore */ descriptor.bundleUrl);
          const registerFn: PluginRegisterFn = mod.default || mod.register;
          if (typeof registerFn === 'function') {
            // Build the allowed extension types set from the CR's declared extensionPoints.
            const allowedTypes = descriptor.extensionPoints?.length
              ? new Set(descriptor.extensionPoints.map((ep) => ep.type))
              : undefined;
            const pluginApi = createPluginApi(descriptor.name, registrations, allowedTypes);
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
          console.error(`[plugins] Failed to load plugin "${descriptor.name}":`, err);
        }
      }

      if (!cancelled) {
        setPlugins(registrations);
        setLoading(false);
      }
    })();

    return () => { cancelled = true; };
  }, [authStatus]);

  return (
    <PluginContext.Provider value={{ plugins, loading }}>
      {children}
    </PluginContext.Provider>
  );
};
