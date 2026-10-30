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

import { AxiosError, AxiosHeaders } from 'axios';
import {
  MonitoringDestination,
  MonitoringDestinationPayload,
} from 'shared-types/monitoring-destinations.types';
import {
  deleteMonitoringDestination,
  MonitoringDestinationRequests,
  saveMonitoringDestination,
  SecretAlreadyExistsError,
} from './monitoring-destination-requests';

const httpError = (status: number) =>
  new AxiosError('failed', String(status), undefined, undefined, {
    status,
    statusText: '',
    data: {},
    headers: {},
    config: { headers: new AxiosHeaders() },
  });

const destination: MonitoringDestinationPayload = {
  metadata: { name: 'pmm' },
  spec: {
    classRef: { name: 'pmm' },
    authentication: {
      mode: 'ServiceAccountToken',
      credentialsSecretRef: { name: 'pmm-credentials' },
    },
    parameters: { url: 'https://pmm.example.com' },
  },
};

const saved: MonitoringDestination = {
  metadata: { name: 'pmm' },
  spec: { classRef: { name: 'pmm' } },
};

const setup = () => {
  const calls: string[] = [];
  const track =
    <T>(name: string, result: T) =>
    (arg: unknown) => {
      calls.push(
        `${name} ${typeof arg === 'string' ? arg : JSON.stringify(arg)}`
      );
      return Promise.resolve(result);
    };
  const requests: MonitoringDestinationRequests = {
    createSecret: vi.fn(track('createSecret', {})),
    deleteSecret: vi.fn(track('deleteSecret', {})),
    createDestination: vi.fn(track('createDestination', saved)),
    updateDestination: vi.fn(track('updateDestination', saved)),
    deleteDestination: vi.fn(track('deleteDestination', {})),
  };
  // The request is still recorded, then fails.
  const fail = (name: keyof MonitoringDestinationRequests, error: Error) =>
    vi.mocked(requests[name]).mockImplementation((arg: unknown) => {
      calls.push(
        `${name} ${typeof arg === 'string' ? arg : JSON.stringify(arg)}`
      );
      return Promise.reject(error);
    });
  return { calls, requests, fail };
};

const token = { token: 's3cr3t' };

describe('saveMonitoringDestination', () => {
  it('creates the Secret before the destination that references it', async () => {
    const { calls, requests } = setup();

    await saveMonitoringDestination(
      {
        namespace: 'team-a',
        isEdit: false,
        destination,
        credentialsSecret: {
          action: 'create',
          name: 'pmm-credentials',
          stringData: token,
        },
      },
      requests
    );

    expect(calls).toEqual([
      `createSecret ${JSON.stringify({ metadata: { name: 'pmm-credentials' }, stringData: token })}`,
      `createDestination ${JSON.stringify(destination)}`,
    ]);
  });

  it('deletes the Secret it created when the destination is refused', async () => {
    const { calls, requests, fail } = setup();
    fail('createDestination', httpError(422));

    await expect(
      saveMonitoringDestination(
        {
          namespace: 'team-a',
          isEdit: false,
          destination,
          credentialsSecret: {
            action: 'create',
            name: 'pmm-credentials',
            stringData: token,
          },
        },
        requests
      )
    ).rejects.toThrow();

    expect(calls[calls.length - 1]).toBe('deleteSecret pmm-credentials');
  });

  it('reports a taken Secret name without creating the destination', async () => {
    const { requests, fail } = setup();
    fail('createSecret', httpError(409));

    await expect(
      saveMonitoringDestination(
        {
          namespace: 'team-a',
          isEdit: false,
          destination,
          credentialsSecret: {
            action: 'create',
            name: 'pmm-credentials',
            stringData: token,
          },
        },
        requests
      )
    ).rejects.toEqual(new SecretAlreadyExistsError('pmm-credentials'));
    expect(requests.createDestination).not.toHaveBeenCalled();
    expect(requests.deleteSecret).not.toHaveBeenCalled();
  });

  it('reuses an existing Secret and never deletes it on failure', async () => {
    const { calls, requests, fail } = setup();
    fail('createDestination', httpError(500));

    await expect(
      saveMonitoringDestination(
        {
          namespace: 'team-a',
          isEdit: false,
          destination,
          credentialsSecret: { action: 'reuse', name: 'pmm-credentials' },
        },
        requests
      )
    ).rejects.toThrow();

    expect(calls).toEqual([`createDestination ${JSON.stringify(destination)}`]);
  });

  it('replaces credentials by recreating the Secret before the update', async () => {
    const { calls, requests, fail } = setup();
    fail('updateDestination', httpError(409));

    await expect(
      saveMonitoringDestination(
        {
          namespace: 'team-a',
          isEdit: true,
          destination,
          credentialsSecret: {
            action: 'replace',
            name: 'pmm-credentials',
            stringData: token,
          },
        },
        requests
      )
    ).rejects.toEqual(httpError(409));

    expect(calls.map((call) => call.split(' ')[0])).toEqual([
      'deleteSecret',
      'createSecret',
      'updateDestination',
    ]);
  });
});

const stored = (name: string, secretName?: string): MonitoringDestination => ({
  metadata: { name },
  spec: {
    classRef: { name: 'pmm' },
    authentication: {
      mode: secretName ? 'ServiceAccountToken' : 'Anonymous',
      ...(secretName && { credentialsSecretRef: { name: secretName } }),
    },
  },
});

describe('deleteMonitoringDestination', () => {
  it('deletes the destination and the Secret created with it', async () => {
    const { calls, requests } = setup();
    const pmm = stored('pmm', 'pmm-credentials');

    await deleteMonitoringDestination(
      { destination: pmm, namespaceDestinations: [pmm] },
      requests
    );

    expect(calls).toEqual([
      'deleteDestination pmm',
      'deleteSecret pmm-credentials',
    ]);
  });

  it('keeps a Secret another destination uses or the form did not name', async () => {
    const { calls, requests } = setup();
    const pmm = stored('pmm', 'pmm-credentials');
    const sharing = stored('pmm-eu', 'pmm-credentials');
    const custom = stored('pmm-us', 'shared-token');

    await deleteMonitoringDestination(
      { destination: pmm, namespaceDestinations: [pmm, sharing] },
      requests
    );
    await deleteMonitoringDestination(
      { destination: custom, namespaceDestinations: [custom] },
      requests
    );

    expect(calls).toEqual([
      'deleteDestination pmm',
      'deleteDestination pmm-us',
    ]);
  });
});
