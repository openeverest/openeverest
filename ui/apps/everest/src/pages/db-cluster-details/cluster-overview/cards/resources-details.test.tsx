// everest
// Copyright (C) 2023 Percona LLC
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

import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DbEngineType } from '@percona/types';
import { TestWrapper } from 'utils/test';
import {
  DbCluster,
  ProxyExposeType,
  Resources,
} from 'shared-types/dbCluster.types';
import { ResourcesDetails } from './resources-details';

const queryClient = new QueryClient();

const makeLegacyCluster = (
  engineType: DbEngineType,
  engineResources: Resources
): DbCluster =>
  ({
    apiVersion: 'everest.percona.com/v1alpha1',
    kind: 'DatabaseCluster',
    metadata: { name: 'legacy-cluster', namespace: 'default' },
    spec: {
      engine: {
        type: engineType,
        version: '1.0.0',
        replicas: 1,
        resources: engineResources,
        storage: { size: '25Gi', class: 'standard' },
      },
      // replicas: 0 -> the proxy section is skipped, keeping the node CPU/memory
      // rows unique so the assertions below are unambiguous.
      proxy: {
        type: 'haproxy' as const,
        replicas: 0,
        expose: { type: ProxyExposeType.ClusterIP },
      },
    },
  }) as unknown as DbCluster;

const renderCard = (dbCluster: DbCluster) =>
  render(
    <QueryClientProvider client={queryClient}>
      <TestWrapper>
        <ResourcesDetails
          dbCluster={dbCluster}
          sharding={dbCluster.spec.sharding}
          loading={false}
          canUpdate={false}
        />
      </TestWrapper>
    </QueryClientProvider>
  );

describe('ResourcesDetails - legacy clusters', () => {
  it.each([
    ['PXC', DbEngineType.PXC],
    ['PSMDB', DbEngineType.PSMDB],
    ['PostgreSQL', DbEngineType.POSTGRESQL],
  ])(
    'shows a single value for legacy %s because requests are synced with limits',
    (_engineName, engineType) => {
      renderCard(makeLegacyCluster(engineType, { cpu: '2', memory: '4G' }));

      // Legacy clusters are treated as if limits and requests were equal, so
      // the card shows a single value (no "limit / request" slash) for every
      // engine.
      const cpuRow = screen.getByTestId('node-cpu-overview-section-row');
      expect(cpuRow).toHaveTextContent(/CPU\s*2$/);
      expect(cpuRow).not.toHaveTextContent('/');

      const memoryRow = screen.getByTestId('memory-overview-section-row');
      expect(memoryRow).toHaveTextContent(/4\s*GB/);
      expect(memoryRow).not.toHaveTextContent('/');
    }
  );
});
