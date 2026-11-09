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

import { act, fireEvent, render, screen } from '@testing-library/react';
import type { CreateEditEndpointModalProps } from 'pages/settings/monitoring-endpoints/createEditModal/create-edit-modal.types';
import { MonitoringEmptyFallback } from './monitoring-empty-fallback';

const mockInvalidateQueries = vi.fn();
const mockCreateMonitoringConfig = vi.fn();

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const queryClient = {
    invalidateQueries: (...args: unknown[]) => mockInvalidateQueries(...args),
  };
  return {
    ...(await importOriginal<typeof import('@tanstack/react-query')>()),
    useQueryClient: () => queryClient,
  };
});

vi.mock('hooks/api/monitoring/useMonitoringConfigsList', () => {
  const mutation = {
    mutate: (...args: unknown[]) => mockCreateMonitoringConfig(...args),
    isPending: false,
  };
  return {
    MONITORING_CONFIGS_QUERY_KEY: 'monitoringConfigs',
    useCreateMonitoringConfig: () => mutation,
  };
});

vi.mock('hooks/rbac', () => {
  const permissions = { canCreate: true };
  return { useRBACPermissions: () => permissions };
});

vi.mock('components/actionable-alert', () => ({
  default: ({
    buttonMessage,
    onClick,
  }: {
    buttonMessage: string;
    onClick: () => void;
  }) => (
    <button type="button" onClick={onClick}>
      {buttonMessage}
    </button>
  ),
}));

vi.mock(
  'pages/settings/monitoring-endpoints/createEditModal/create-edit-modal',
  () => ({
    CreateEditEndpointModal: ({
      handleSubmit,
    }: CreateEditEndpointModalProps) => (
      <button
        type="button"
        onClick={() =>
          handleSubmit(false, {
            name: 'pmm',
            namespace: 'ns',
            url: 'https://pmm.example.com',
            verifyTLS: true,
            user: 'admin',
            password: 'secret',
          })
        }
      >
        submit endpoint
      </button>
    ),
  })
);

describe('MonitoringEmptyFallback', () => {
  it('selects the created endpoint only after the options are refetched', async () => {
    let finishRefetch: () => void = () => {};
    mockInvalidateQueries.mockReturnValue(
      new Promise<void>((resolve) => {
        finishRefetch = resolve;
      })
    );
    mockCreateMonitoringConfig.mockImplementation(
      (_variables: unknown, { onSuccess }: { onSuccess: () => unknown }) =>
        onSuccess()
    );
    const onCreated = vi.fn();

    render(
      <MonitoringEmptyFallback
        namespace="ns"
        cluster="main"
        onCreated={onCreated}
      />
    );
    fireEvent.click(
      screen.getByRole('button', { name: 'Add monitoring endpoint' })
    );
    fireEvent.click(screen.getByRole('button', { name: 'submit endpoint' }));

    expect(onCreated).not.toHaveBeenCalled();

    await act(async () => finishRefetch());

    expect(onCreated).toHaveBeenCalledWith('pmm');
  });
});
