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

import { useMutation } from '@tanstack/react-query';
import { getAuthToken } from 'api/session-token';

export interface PluginInstanceConfig {
  pluginName: string;
  instanceName: string;
  namespace: string;
  config: Record<string, unknown>;
}

async function submitPluginInstanceConfig(
  payload: PluginInstanceConfig
): Promise<void> {
  const token = getAuthToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const resp = await window.fetch(
    `/v1/plugins/${encodeURIComponent(payload.pluginName)}/instance-config`,
    {
      method: 'POST',
      headers,
      body: JSON.stringify({
        instance: payload.instanceName,
        namespace: payload.namespace,
        config: payload.config,
      }),
    }
  );
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`Plugin instance-config failed (${resp.status}): ${text}`);
  }
}

/**
 * Mutation hook that POSTs plugin-specific configuration to
 * POST /v1/plugins/{name}/instance-config. The host calls this after
 * form submission to hand off plugin config collected via
 * instanceCreateFormSection / instanceEditFormSection extensions.
 */
export const useSubmitPluginInstanceConfig = () => {
  return useMutation({
    mutationFn: submitPluginInstanceConfig,
  });
};
