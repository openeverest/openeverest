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
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { DbWizardFormFields } from 'consts';
import { MonitoringStep } from './monitoring-step';
import { MONITORING_DESTINATIONS_FIELD } from './monitoring-step.constants';

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
  return { useMonitoringDestinations: () => result };
});

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

  it('writes spec.monitoring.destinations entries when cards are toggled', () => {
    render(
      <Wrapper>
        <MonitoringStep />
      </Wrapper>
    );

    fireEvent.click(getCard('prometheus'));
    expect(selectedValue()).toEqual([
      { name: 'prometheus', destinationRef: { name: 'prometheus' } },
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
        { name: 'prometheus', destinationRef: { name: 'prometheus' } },
      ])
    );
  });
});
