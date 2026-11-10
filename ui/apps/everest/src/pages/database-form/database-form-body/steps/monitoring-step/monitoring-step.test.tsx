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

import React from 'react';
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { DbWizardFormFields } from 'consts';
import { MonitoringStep } from './monitoring-step';
import { MONITORING_DESTINATIONS_FIELD } from './monitoring-step.constants';

const mocks = vi.hoisted(() => ({
  permissions: { canRead: true, isPermissionsLoading: false },
  destinationsFailed: false,
}));

vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => mocks.permissions,
}));

vi.mock('hooks/api/monitoring-classes', () => {
  const classes = [
    {
      metadata: { name: 'pmm' },
      spec: {
        displayName: 'Percona Monitoring and Management',
        authModes: [{ name: 'ServiceAccountToken' }],
        providerManaged: {
          integration: 'pmm',
          agentImage: 'pmm-client:3',
          agentVersion: '3.0.0',
        },
        uiSchema: {
          destination: {
            components: {
              url: { path: 'parameters.url', uiType: 'text' },
            },
          },
          instance: {
            components: {
              cpuRequest: {
                path: 'parameters.resources.requests.cpu',
                uiType: 'text',
                fieldParams: { label: 'CPU request' },
              },
            },
          },
        },
      },
      status: {
        providers: [
          { name: 'percona-server-mongodb', mode: 'ProviderManaged' },
        ],
      },
    },
    {
      metadata: { name: 'prometheus' },
      spec: {
        displayName: 'Prometheus',
        authModes: [{ name: 'Anonymous' }],
        uiSchema: {
          instance: {
            components: {
              interval: {
                path: 'parameters.interval',
                uiType: 'text',
                fieldParams: { label: 'Scrape interval' },
              },
            },
          },
        },
      },
      status: {
        providers: [
          { name: 'percona-server-mongodb', mode: 'ExtensionManaged' },
        ],
      },
    },
    {
      metadata: { name: 'victoriametrics' },
      spec: {
        displayName: 'VictoriaMetrics',
        authModes: [{ name: 'Anonymous' }],
      },
      status: {
        conditions: [{ type: 'Accepted', status: 'True' }],
        providers: [
          { name: 'percona-server-mongodb', mode: 'ExtensionManaged' },
        ],
      },
    },
  ];
  const result = { data: classes, isLoading: false };
  return { useMonitoringClassesForProvider: () => result };
});

vi.mock('hooks/api/monitoring-destinations', () => {
  const ready = [
    {
      type: 'Ready',
      status: 'True',
      reason: 'Ready',
      message: '',
      lastTransitionTime: '2026-01-01T00:00:00Z',
    },
  ];
  const destinations = [
    {
      metadata: { name: 'pmm-a' },
      spec: {
        classRef: { name: 'pmm' },
        parameters: { url: 'https://pmm-a.example.com' },
      },
      status: { conditions: ready },
    },
    {
      metadata: { name: 'pmm-b' },
      spec: { classRef: { name: 'pmm' } },
      status: { conditions: ready },
    },
    {
      metadata: { name: 'prometheus' },
      spec: { classRef: { name: 'prometheus' } },
      status: { conditions: ready },
    },
    {
      metadata: { name: 'of-an-incompatible-class' },
      spec: { classRef: { name: 'other' } },
      status: { conditions: ready },
    },
  ];
  const result = {
    data: destinations,
    isLoading: false,
    isSuccess: true,
  };
  const failed = { isLoading: false, isSuccess: false, isError: true };
  const save = { mutate: () => {}, isPending: false };
  const noDestination = {};
  return {
    useMonitoringDestinations: () =>
      mocks.destinationsFailed ? failed : result,
    useMonitoringDestination: () => noDestination,
    useSaveMonitoringDestination: () => save,
    SecretAlreadyExistsError: class extends Error {},
  };
});

vi.mock(
  'pages/settings/monitoring-destinations/hooks/use-monitoring-destination-permissions',
  () => {
    const permissions = {
      createNamespaces: ['team-a'],
      updateNamespaces: ['team-a'],
      deleteNamespaces: ['team-a'],
    };
    return { useMonitoringDestinationPermissions: () => permissions };
  }
);

const SelectedValue = () => {
  const value = useWatch({ name: MONITORING_DESTINATIONS_FIELD });
  return <pre data-testid="selected">{JSON.stringify(value ?? [])}</pre>;
};

const Wrapper = ({
  children,
  selected = [],
}: {
  children: React.ReactNode;
  selected?: unknown[];
}) => {
  const methods = useForm({
    defaultValues: {
      [DbWizardFormFields.provider]: 'percona-server-mongodb',
      [DbWizardFormFields.k8sNamespace]: 'team-a',
      monitoring: { destinations: selected },
    },
  });
  // The watcher renders first so it subscribes before the step's effects run.
  return (
    <TestWrapper>
      <FormProvider {...methods}>
        <SelectedValue />
        {children}
      </FormProvider>
    </TestWrapper>
  );
};

const selectedValue = () =>
  JSON.parse(screen.getByTestId('selected').textContent ?? '[]');

const cardPattern = (name: string) => new RegExp(`^${name}`);
const queryCard = (name: string) =>
  screen.queryByRole('button', { name: cardPattern(name) });
const getCard = (name: string) =>
  screen.getByRole('button', { name: cardPattern(name) });

describe('MonitoringStep', () => {
  afterEach(() => {
    mocks.permissions = { canRead: true, isPermissionsLoading: false };
    mocks.destinationsFailed = false;
  });

  it('groups destinations under their monitoring system', () => {
    render(
      <Wrapper>
        <MonitoringStep />
      </Wrapper>
    );

    expect(screen.getByTestId('monitoring-class-pmm')).toHaveTextContent(
      'Percona Monitoring and Management'
    );
    expect(getCard('pmm-a')).toHaveTextContent('https://pmm-a.example.com');
    expect(getCard('prometheus')).toBeInTheDocument();
    expect(queryCard('of-an-incompatible-class')).not.toBeInTheDocument();
    expect(
      screen.getByTestId('monitoring-class-victoriametrics')
    ).toHaveTextContent(
      'No destinations of this system in the team-a namespace yet.'
    );
  });

  it('creates a destination of an empty system in place, in its class and namespace', async () => {
    render(
      <Wrapper>
        <MonitoringStep />
      </Wrapper>
    );

    fireEvent.click(
      within(
        screen.getByTestId('no-monitoring-destinations-victoriametrics')
      ).getByRole('button', { name: 'Add destination' })
    );

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('VictoriaMetrics')).toBeInTheDocument();
    expect(within(dialog).getByDisplayValue('team-a')).toBeInTheDocument();
  });

  it('writes spec.monitoring.destinations entries when cards are toggled', () => {
    render(
      <Wrapper>
        <MonitoringStep />
      </Wrapper>
    );

    fireEvent.click(getCard('prometheus'));
    expect(selectedValue()).toEqual([
      {
        name: 'prometheus',
        destinationRef: { name: 'prometheus' },
        parameters: { interval: '' },
      },
    ]);
    expect(getCard('prometheus')).toHaveAttribute('aria-pressed', 'true');

    fireEvent.click(getCard('prometheus'));
    expect(selectedValue()).toEqual([]);
  });

  it('allows only one destination per provider-managed integration', () => {
    render(
      <Wrapper>
        <MonitoringStep />
      </Wrapper>
    );

    fireEvent.click(getCard('pmm-a'));

    expect(queryCard('pmm-b')).not.toBeInTheDocument();
    expect(
      screen.getByText('Another destination of this system is already selected')
    ).toBeInTheDocument();
    expect(getCard('prometheus')).toBeInTheDocument();
  });

  it('drops a selection whose destination is not in the namespace', async () => {
    render(
      <Wrapper
        selected={[
          { name: 'gone', destinationRef: { name: 'gone' } },
          { name: 'prometheus', destinationRef: { name: 'prometheus' } },
        ]}
      >
        <MonitoringStep />
      </Wrapper>
    );

    await waitFor(() =>
      expect(selectedValue()).toEqual([
        expect.objectContaining({
          name: 'prometheus',
          destinationRef: { name: 'prometheus' },
        }),
      ])
    );
  });

  it("fills each entry's parameters from its class instance form", () => {
    render(
      <Wrapper>
        <MonitoringStep />
      </Wrapper>
    );
    const fieldOf = (destination: string, label: RegExp) =>
      within(
        screen.getByTestId(`monitoring-entry-parameters-${destination}`)
      ).getByLabelText(label);

    expect(
      screen.queryByTestId('monitoring-entry-parameters-pmm-a')
    ).not.toBeInTheDocument();

    fireEvent.click(getCard('pmm-a'));
    fireEvent.click(getCard('prometheus'));
    fireEvent.change(fieldOf('pmm-a', /CPU request/), {
      target: { value: '200m' },
    });
    fireEvent.change(fieldOf('prometheus', /Scrape interval/), {
      target: { value: '30s' },
    });

    expect(selectedValue()).toEqual([
      {
        name: 'pmm-a',
        destinationRef: { name: 'pmm-a' },
        parameters: { resources: { requests: { cpu: '200m' } } },
      },
      {
        name: 'prometheus',
        destinationRef: { name: 'prometheus' },
        parameters: { interval: '30s' },
      },
    ]);

    fireEvent.click(getCard('pmm-a'));

    expect(selectedValue()).toEqual([
      {
        name: 'prometheus',
        destinationRef: { name: 'prometheus' },
        parameters: { interval: '30s' },
      },
    ]);
    expect(fieldOf('prometheus', /Scrape interval/)).toHaveValue('30s');
  });

  it("says the destinations can't be read instead of offering to add one", () => {
    mocks.permissions = { canRead: false, isPermissionsLoading: false };
    render(
      <Wrapper>
        <MonitoringStep />
      </Wrapper>
    );

    expect(
      screen.getByTestId('monitoring-destinations-unavailable')
    ).toHaveTextContent(
      "You don't have permission to view monitoring destinations in the team-a namespace."
    );
    expect(screen.queryByText('Add destination')).not.toBeInTheDocument();
  });

  it('says the destinations could not be loaded', () => {
    mocks.destinationsFailed = true;
    render(
      <Wrapper>
        <MonitoringStep />
      </Wrapper>
    );

    expect(
      screen.getByTestId('monitoring-destinations-unavailable')
    ).toHaveTextContent(
      'Monitoring destinations could not be loaded. Try again later.'
    );
  });
});
