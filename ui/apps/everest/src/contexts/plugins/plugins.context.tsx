import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import type { Extension, PluginApi, PluginRegisterFn } from '@openeverest/plugin-sdk';
import AuthContext from 'contexts/auth/auth.context';

export interface PluginRegistration {
  name: string;
  extensions: Extension[];
}

interface PluginDescriptor {
  name: string;
  displayName: string;
  bundleUrl: string;
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
function createPluginApi(pluginName: string, registrations: PluginRegistration[]): PluginApi {
  const registration: PluginRegistration = { name: pluginName, extensions: [] };
  registrations.push(registration);

  return {
    React,

    registerExtension(extension: Extension) {
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
            const pluginApi = createPluginApi(descriptor.name, registrations);
            registerFn(pluginApi);
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
