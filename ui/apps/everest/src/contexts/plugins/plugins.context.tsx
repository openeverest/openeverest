import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import type { Extension, PluginApi, PluginRegisterFn } from '@openeverest/plugin-sdk';

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
      const url = `/v1/plugins/${pluginName}${path}`;
      return window.fetch(url, init);
    },
  };
}

async function loadPluginDescriptors(): Promise<PluginDescriptor[]> {
  try {
    const resp = await fetch('/v1/plugins');
    if (!resp.ok) return [];
    return await resp.json();
  } catch {
    return [];
  }
}

export const PluginProvider = ({ children }: { children: ReactNode }) => {
  const [plugins, setPlugins] = useState<PluginRegistration[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    (async () => {
      const descriptors = await loadPluginDescriptors();
      const registrations: PluginRegistration[] = [];

      for (const descriptor of descriptors) {
        try {
          const mod = await import(/* @vite-ignore */ descriptor.bundleUrl);
          const registerFn: PluginRegisterFn = mod.default || mod.register;
          if (typeof registerFn === 'function') {
            const api = createPluginApi(descriptor.name, registrations);
            registerFn(api);
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
  }, []);

  return (
    <PluginContext.Provider value={{ plugins, loading }}>
      {children}
    </PluginContext.Provider>
  );
};
