// everest
// Copyright (C) 2023 Percona LLC
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

import { useEffect, useState } from 'react';
import { useRBACPermissions } from 'hooks/rbac';

const BASE_URL = '/v1/';
const MAX_LOG_LINES = 10000;

export const useDbClusterComponentLogsStream = (
  namespace: string,
  dbClusterName: string,
  componentName: string,
  container?: string,
  options?: { enabled?: boolean; maxLines?: number }
) => {
  const [logs, setLogs] = useState('');
  const [isConnecting, setIsConnecting] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const maxLines = options?.maxLines ?? MAX_LOG_LINES;

  const { canRead } = useRBACPermissions(
    'database-clusters',
    `${namespace}/${dbClusterName}`
  );

  const enabled =
    (options?.enabled ?? true) &&
    canRead &&
    !!componentName &&
    componentName.length > 0;

  useEffect(() => {
    if (!enabled) {
      setLogs('');
      setIsConnecting(false);
      return;
    }

    const abortController = new AbortController();

    setLogs('');
    setIsConnecting(true);
    setError(null);

    const connectionTimeout = setTimeout(() => {
      setIsConnecting(false);
    }, 2000);

    const streamLogs = async () => {
      try {
        const params = new URLSearchParams();
        if (container) {
          params.append('container', container);
        }
        params.append('follow', 'true');
        const url = `${BASE_URL}namespaces/${namespace}/database-clusters/${dbClusterName}/components/${componentName}/logs?${params}`;

        const token = localStorage.getItem('everestToken');
        const response = await fetch(url, {
          signal: abortController.signal,
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });

        clearTimeout(connectionTimeout);

        if (abortController.signal.aborted) return;

        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }

        setIsConnecting(false);

        const reader = response.body?.getReader();
        if (!reader) throw new Error('No response body');

        const decoder = new TextDecoder();

        // eslint-disable-next-line no-constant-condition
        while (true) {
          const { value, done } = await reader.read();
          if (done || abortController.signal.aborted) break;
          if (value) {
            const chunk = decoder.decode(value, { stream: true });
            setLogs((prev) => {
              const newLogs = prev + chunk;
              const lines = newLogs.split('\n');
              if (lines.length > maxLines) {
                return lines.slice(-maxLines).join('\n');
              }
              return newLogs;
            });
          }
        }
      } catch (err) {
        if (err instanceof Error && err.name === 'AbortError') return;
        if (abortController.signal.aborted) return;
        setError(err instanceof Error ? err : new Error('Unknown error'));
        setIsConnecting(false);
      }
    };

    streamLogs();

    return () => {
      clearTimeout(connectionTimeout);
      abortController.abort();
    };
  }, [namespace, dbClusterName, componentName, container, enabled, maxLines]);

  const getFullLogs = async (): Promise<string> => {
    const params = new URLSearchParams();
    if (container) {
      params.append('container', container);
    }
    params.append('follow', 'false');

    const url = `${BASE_URL}namespaces/${namespace}/database-clusters/${dbClusterName}/components/${componentName}/logs?${params}`;
    const token = localStorage.getItem('everestToken');
    const response = await fetch(url, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return await response.text();
  };

  return { logs, isConnecting, error, getFullLogs };
};
