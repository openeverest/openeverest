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
import { createSecretFn, deleteSecretFn } from 'api/secrets';
import { createMonitoringDestinationFn } from 'api/monitoring-destinations';
import { MonitoringClass } from 'shared-types/monitoring-destinations.types';
import { TestWrapper } from 'utils/test';
import { MonitoringDestinationModal } from './monitoring-destination-modal';

vi.mock('api/secrets', () => ({
  createSecretFn: vi.fn(),
  deleteSecretFn: vi.fn(),
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

// The reference PMM class: url in the destination form, a token per mode.
const pmm: MonitoringClass = {
  metadata: { name: 'pmm' },
  spec: {
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

const renderModal = () => {
  const onClose = vi.fn();
  render(
    <QueryClientProvider client={new QueryClient()}>
      <TestWrapper>
        <MonitoringDestinationModal
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
      metadata: { name: 'pmm-prod-credentials' },
      stringData: { token: 's3cr3t' },
    });
    expect(createMonitoringDestinationFn).toHaveBeenCalledWith(
      'main',
      'team-a',
      {
        metadata: { name: 'pmm-prod' },
        spec: {
          classRef: { name: 'pmm' },
          authentication: {
            mode: 'ServiceAccountToken',
            credentialsSecretRef: { name: 'pmm-prod-credentials' },
          },
          parameters: { url: 'https://pmm.example.com' },
        },
      }
    );
    expect(vi.mocked(createSecretFn).mock.invocationCallOrder[0]).toBeLessThan(
      vi.mocked(createMonitoringDestinationFn).mock.invocationCallOrder[0]
    );
  });

  it('offers to reuse a Secret whose name is taken', async () => {
    vi.mocked(createSecretFn).mockRejectedValue(conflict);
    const onClose = renderModal();
    await fillPmmDestination();
    await submit();

    expect(
      await screen.findByText(
        /A Secret named pmm-prod-credentials already exists/
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
            credentialsSecretRef: { name: 'pmm-prod-credentials' },
          },
        }),
      })
    );
  });

  it('deletes the Secret it created when the destination is refused', async () => {
    vi.mocked(createMonitoringDestinationFn).mockRejectedValue(new Error('no'));
    const onClose = renderModal();
    await fillPmmDestination();
    await submit();

    await waitFor(() =>
      expect(deleteSecretFn).toHaveBeenCalledWith(
        'main',
        'team-a',
        'pmm-prod-credentials'
      )
    );
    expect(onClose).not.toHaveBeenCalled();
  });
});
