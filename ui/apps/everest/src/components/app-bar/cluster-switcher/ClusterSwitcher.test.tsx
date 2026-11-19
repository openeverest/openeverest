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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { registerClusterFn } from 'api/clusters';
import ClusterSwitcher from './ClusterSwitcher';

const clusterContext = vi.hoisted(() => ({
  selectCluster: vi.fn(),
  refreshClusters: vi.fn(),
}));

vi.mock('contexts/clusters/useClusters', () => ({
  useClusters: () => ({
    clusterName: 'local',
    clusters: [
      { name: 'local', server: 'https://kubernetes.default.svc' },
      { name: 'prod', server: 'https://agent.example.test' },
    ],
    ...clusterContext,
  }),
}));
vi.mock('api/clusters', () => ({ registerClusterFn: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar: vi.fn() }));

describe('ClusterSwitcher', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clusterContext.refreshClusters.mockResolvedValue(undefined);
    vi.mocked(registerClusterFn).mockResolvedValue({
      name: 'staging',
      server: 'https://staging-agent.example.test',
    });
  });

  it('selects a registered cluster', () => {
    render(<ClusterSwitcher />);

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Cluster' }));
    fireEvent.click(screen.getByRole('option', { name: 'prod' }));

    expect(clusterContext.selectCluster).toHaveBeenCalledWith('prod');
  });

  it('registers and selects a newly connected cluster', async () => {
    render(<ClusterSwitcher />);

    fireEvent.click(screen.getByRole('button', { name: 'Connect' }));
    fireEvent.change(await screen.findByLabelText(/Cluster name/), {
      target: { value: 'staging' },
    });
    fireEvent.change(screen.getByLabelText(/Agent HTTPS URL/), {
      target: { value: 'https://staging-agent.example.test' },
    });
    fireEvent.change(screen.getByLabelText(/Agent bearer token/), {
      target: { value: 'token' },
    });
    fireEvent.change(
      screen.getByLabelText('Agent CA certificate (optional PEM)'),
      {
        target: { value: 'certificate' },
      }
    );
    fireEvent.click(screen.getByRole('button', { name: 'Connect cluster' }));

    await waitFor(() =>
      expect(registerClusterFn).toHaveBeenCalledWith({
        name: 'staging',
        server: 'https://staging-agent.example.test',
        bearerToken: 'token',
        caData: btoa('certificate'),
      })
    );
    expect(clusterContext.refreshClusters).toHaveBeenCalled();
    expect(clusterContext.selectCluster).toHaveBeenCalledWith('staging');
  });
});
