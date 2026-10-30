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

import { render, screen, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Instance } from 'shared-types/api.types';
import { TestWrapper } from 'utils/test';
import { MonitoringCard } from './monitoring-card';

const mocks = vi.hoisted(() => ({ classes: { data: [{}] } }));

vi.mock('hooks/api/monitoring-classes', () => ({
  useMonitoringClassesForProvider: () => mocks.classes,
}));
vi.mock('hooks/rbac', () => {
  const permissions = { canUpdate: true };
  return { useRBACPermissions: () => permissions };
});

const condition = (
  status: 'True' | 'False' | 'Unknown',
  reason: string,
  message = ''
) => ({
  type: 'MonitoringConfigured',
  status,
  reason,
  message,
  lastTransitionTime: '2026-01-01T00:00:00Z',
});

const monitoredInstance: Instance = {
  metadata: { name: 'orders-db', namespace: 'team-a' },
  spec: {
    providerRef: { name: 'percona-server-mongodb' },
    monitoring: {
      destinations: [
        { name: 'pmm', destinationRef: { name: 'pmm-prod' } },
        { name: 'prometheus', destinationRef: { name: 'prom' } },
      ],
    },
  },
  status: {
    conditions: [
      condition('False', 'DestinationNotConfigured', '1 of 2 not configured'),
    ],
    monitoring: {
      destinations: [
        {
          name: 'pmm',
          mode: 'ProviderManaged',
          configured: 'False',
          reason: 'AgentNotRegistered',
          message: 'pmm-client could not reach the server',
        },
      ],
    },
  },
};

const renderCard = (instance: Instance) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <TestWrapper>
        <MonitoringCard instance={instance} namespace="team-a" editable />
      </TestWrapper>
    </QueryClientProvider>
  );

describe('MonitoringCard', () => {
  it('shows the roll-up and the state core reports per destination', () => {
    renderCard(monitoredInstance);

    expect(
      screen.getByTestId('monitoring-configured-overview-section')
    ).toHaveTextContent(
      'Not configuredDetailsDestinationNotConfigured: 1 of 2 not configured'
    );
    const pmm = screen.getByTestId('monitoring-pmm-overview-section');
    expect(pmm).toHaveTextContent('pmm-prod');
    expect(pmm).toHaveTextContent('ProviderManaged');
    expect(pmm).toHaveTextContent(
      'AgentNotRegistered: pmm-client could not reach the server'
    );
    expect(
      within(
        screen.getByTestId('monitoring-prometheus-overview-section')
      ).getByText('Pending')
    ).toBeInTheDocument();
    expect(screen.getByTestId('monitoring-edit-button')).toBeInTheDocument();
  });

  it('is hidden when the provider has no monitoring and none is set', () => {
    mocks.classes = { data: [] };
    renderCard({
      ...monitoredInstance,
      spec: { providerRef: { name: 'percona-server-mongodb' } },
    });

    expect(screen.queryByTestId('monitoring-details')).not.toBeInTheDocument();
  });
});
