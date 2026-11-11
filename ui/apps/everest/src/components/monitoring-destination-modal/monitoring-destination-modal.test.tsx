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

import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError, AxiosHeaders } from 'axios';
import { createSecretFn, deleteSecretFn, getSecretFn } from 'api/secrets';
import {
  createMonitoringDestinationFn,
  getMonitoringDestinationFn,
} from 'api/monitoring-destinations';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { TestWrapper } from 'utils/test';
import { MonitoringDestinationModal } from './monitoring-destination-modal';
import { MonitoringDestinationTarget } from './monitoring-destination-modal.types';

vi.mock('api/secrets', () => ({
  createSecretFn: vi.fn(),
  deleteSecretFn: vi.fn(),
  getSecretFn: vi.fn(),
  getSecretsFn: vi.fn(),
}));
vi.mock('api/monitoring-destinations', () => ({
  createMonitoringDestinationFn: vi.fn(),
  updateMonitoringDestinationFn: vi.fn(),
  deleteMonitoringDestinationFn: vi.fn(),
  getMonitoringDestinationFn: vi.fn(),
  getMonitoringDestinationsFn: vi.fn(),
}));

const conflict = new AxiosError('exists', '409', undefined, undefined, {
  status: 409,
  statusText: 'Conflict',
  data: {},
  headers: {},
  config: { headers: new AxiosHeaders() },
});

const pmmCredentialsLabels = {
  'openeverest.io/monitoring-class': 'pmm',
  'openeverest.io/definition': 'ServiceAccountToken',
};

// The reference PMM class: url in the destination form, a token per mode.
const pmm: MonitoringClass = {
  metadata: { name: 'pmm' },
  spec: {
    controllerName: 'openeverest.io/monitoring-pmm',
    displayName: 'PMM',
    authModes: [
      { name: 'ServiceAccountToken', credentials: { required: ['token'] } },
    ],
  },
  status: {
    conditions: [
      {
        type: 'Accepted',
        status: 'True',
        reason: 'Accepted',
        message: '',
        lastTransitionTime: '2026-01-01T00:00:00Z',
      },
    ],
    providers: [
      { name: 'percona-server-mongodb', disruption: 'RollingRestart' },
    ],
  },
};
Reflect.set(pmm.spec, 'uiSchema', {
  destination: {
    components: {
      url: {
        path: 'parameters.url',
        uiType: 'text',
        fieldParams: { label: 'PMM server URL' },
        validation: { required: true },
      },
    },
  },
  credentials: {
    ServiceAccountToken: {
      components: {
        token: {
          path: 'token',
          uiType: 'text',
          fieldParams: { label: 'Service account token', type: 'password' },
          validation: { required: true },
        },
      },
    },
  },
});

const renderModal = (target?: MonitoringDestinationTarget) => {
  const onClose = vi.fn();
  render(
    <QueryClientProvider client={new QueryClient()}>
      <TestWrapper>
        <MonitoringDestinationModal
          target={target}
          classes={[pmm]}
          namespaces={['team-a']}
          onClose={onClose}
        />
      </TestWrapper>
    </QueryClientProvider>
  );
  return onClose;
};

const fillPmmDestination = async () => {
  fireEvent.change(screen.getByTestId('text-input-name'), {
    target: { value: 'pmm-prod' },
  });
  fireEvent.mouseDown(
    within(screen.getByTestId('select-class-name-button')).getByRole('combobox')
  );
  fireEvent.click(screen.getByRole('option', { name: 'PMM' }));
  fireEvent.change(await screen.findByLabelText(/PMM server URL/), {
    target: { value: 'https://pmm.example.com' },
  });
  fireEvent.change(screen.getByLabelText(/Service account token/), {
    target: { value: 's3cr3t' },
  });
};

const submit = async () => {
  const button = screen.getByTestId('form-dialog-add');
  await waitFor(() => expect(button).toBeEnabled());
  fireEvent.click(button);
};

describe('MonitoringDestinationModal', () => {
  beforeEach(() => {
    vi.mocked(createSecretFn).mockReset().mockResolvedValue({});
    vi.mocked(getSecretFn)
      .mockReset()
      .mockResolvedValue({
        metadata: {
          name: 'monitoring-destination-pmm-prod-credentials',
          labels: pmmCredentialsLabels,
        },
      });
    vi.mocked(deleteSecretFn).mockReset().mockResolvedValue();
    vi.mocked(createMonitoringDestinationFn)
      .mockReset()
      .mockResolvedValue({ spec: { classRef: { name: 'pmm' } } });
  });

  it('stores the credentials in a Secret the destination references', async () => {
    const onClose = renderModal();
    await fillPmmDestination();
    await submit();

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(createSecretFn).toHaveBeenCalledWith('main', 'team-a', {
      metadata: {
        name: 'monitoring-destination-pmm-prod-credentials',
        labels: pmmCredentialsLabels,
      },
      stringData: { token: 's3cr3t' },
    });
    expect(createMonitoringDestinationFn).toHaveBeenCalledWith(
      'main',
      'team-a',
      {
        metadata: {
          name: 'pmm-prod',
          annotations: {
            'monitoring.openeverest.io/ui-created-credentials-secret':
              'monitoring-destination-pmm-prod-credentials',
          },
        },
        spec: {
          classRef: { name: 'pmm' },
          authentication: {
            mode: 'ServiceAccountToken',
            credentialsSecretRef: {
              name: 'monitoring-destination-pmm-prod-credentials',
            },
          },
          parameters: { url: 'https://pmm.example.com' },
        },
      }
    );
  });

  it('offers to reuse a Secret whose name is taken', async () => {
    vi.mocked(createSecretFn).mockRejectedValue(conflict);
    const onClose = renderModal();
    await fillPmmDestination();
    await submit();

    expect(
      await screen.findByText(
        /A Secret named monitoring-destination-pmm-prod-credentials already exists/
      )
    ).toBeInTheDocument();
    expect(createMonitoringDestinationFn).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: 'Reuse it' }));

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(createSecretFn).toHaveBeenCalledTimes(1);
    expect(createMonitoringDestinationFn).toHaveBeenCalledWith(
      'main',
      'team-a',
      expect.objectContaining({
        spec: expect.objectContaining({
          authentication: {
            mode: 'ServiceAccountToken',
            credentialsSecretRef: {
              name: 'monitoring-destination-pmm-prod-credentials',
            },
          },
        }),
      })
    );
  });

  it('does not offer a taken name whose Secret holds other credentials', async () => {
    vi.mocked(createSecretFn).mockRejectedValue(conflict);
    vi.mocked(getSecretFn).mockResolvedValue({
      metadata: {
        name: 'monitoring-destination-pmm-prod-credentials',
        labels: { 'openeverest.io/provider': 'psmdb' },
      },
    });
    renderModal();
    await fillPmmDestination();
    await submit();

    expect(
      await screen.findByText(
        /doesn't hold credentials for this monitoring class/
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Reuse it' })
    ).not.toBeInTheDocument();
    expect(createMonitoringDestinationFn).not.toHaveBeenCalled();
  });

  it('stays open after rolling back a refused destination', async () => {
    vi.mocked(createMonitoringDestinationFn).mockRejectedValue(new Error('no'));
    const onClose = renderModal();
    await fillPmmDestination();
    await submit();

    await waitFor(() =>
      expect(deleteSecretFn).toHaveBeenCalledWith(
        'main',
        'team-a',
        'monitoring-destination-pmm-prod-credentials'
      )
    );
    expect(onClose).not.toHaveBeenCalled();
  });

  it('warns about restarting the attached instances only once the parameters change', async () => {
    const stored: MonitoringDestination = {
      metadata: { name: 'pmm-prod', namespace: 'team-a', resourceVersion: '1' },
      spec: {
        classRef: { name: 'pmm' },
        authentication: {
          mode: 'ServiceAccountToken',
          credentialsSecretRef: { name: 'pmm-credentials' },
        },
      },
      status: { attachedInstances: 2 },
    };
    Reflect.set(stored.spec, 'parameters', { url: 'https://pmm.example.com' });
    vi.mocked(getMonitoringDestinationFn).mockResolvedValue(stored);
    renderModal({ namespace: 'team-a', name: 'pmm-prod' });

    const url = await screen.findByLabelText(/PMM server URL/);
    await waitFor(() => expect(url).toHaveValue('https://pmm.example.com'));
    expect(
      screen.queryByTestId('monitoring-destination-restart')
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByLabelText(/Replace the credentials/));
    await screen.findAllByLabelText(/token/i);
    expect(
      screen.queryByTestId('monitoring-destination-restart')
    ).not.toBeInTheDocument();

    fireEvent.change(url, { target: { value: 'https://pmm2.example.com' } });
    expect(
      await screen.findByTestId('monitoring-destination-restart')
    ).toHaveTextContent('the 2 instances that send to this destination');
  });
});
