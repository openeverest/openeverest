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
import { MONITORING_READY_CONDITION } from './monitoring-card.constants';
import { ConditionStatus } from './monitoring-card.types';

const mocks = vi.hoisted(() => ({ classes: { data: [{}] } }));

vi.mock('hooks/api/monitoring-classes', () => ({
  useMonitoringClassesForProvider: () => mocks.classes,
}));
vi.mock('hooks/rbac', () => {
  const permissions = { canUpdate: true };
  return { useRBACPermissions: () => permissions };
});

// Tests may swap the classes response; put the default back so a test
// added later does not inherit it.
const initialClasses = mocks.classes;
afterEach(() => {
  mocks.classes = initialClasses;
});

const condition = (
  type: string,
  status: ConditionStatus,
  reason: string,
  message = ''
) => ({
  type,
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
      condition(
        MONITORING_READY_CONDITION,
        'False',
        'MonitoringNotReady',
        'pmm: DestinationNotReady; prometheus: Pending'
      ),
    ],
    monitoring: {
      destinations: [
        {
          name: 'pmm',
          mode: 'ProviderManaged',
          conditions: [
            condition(
              'Configured',
              'True',
              'AgentTLSNotVerified',
              'pmm-client does not verify the server certificate'
            ),
            condition(
              'Ready',
              'False',
              'DestinationNotReady',
              'MonitoringDestination "pmm-prod" is not Ready (Unauthorized)'
            ),
          ],
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
      screen.getByTestId('monitoring-ready-overview-section')
    ).toHaveTextContent('Not ready');
    const pmm = screen.getByTestId('monitoring-pmm-overview-section');
    expect(pmm).toHaveTextContent('pmm-prod');
    expect(pmm).toHaveTextContent('Not ready');
    expect(pmm).toHaveTextContent('Provider-managed');
    expect(pmm).toHaveTextContent(
      'pmm-prod is not ready, so no data reaches it.'
    );
    expect(
      within(
        screen.getByTestId('monitoring-prometheus-overview-section')
      ).getByText('Pending')
    ).toBeInTheDocument();
    expect(screen.getByTestId('monitoring-edit-button')).toBeInTheDocument();
  });

  it('tells to remove a destination that is being deleted', () => {
    renderCard({
      ...monitoredInstance,
      status: {
        monitoring: {
          destinations: [
            {
              name: 'pmm',
              conditions: [
                condition(
                  'Configured',
                  'False',
                  'DestinationTerminating',
                  'MonitoringDestination "pmm-prod" is being deleted'
                ),
                condition('Ready', 'False', 'NotAccepted', ''),
              ],
            },
          ],
        },
      },
    });

    expect(
      screen.getByTestId('monitoring-pmm-overview-section')
    ).toHaveTextContent(
      'Not readyDetailspmm-prod is being deleted. Remove it from monitoring to let the deletion finish.'
    );
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
