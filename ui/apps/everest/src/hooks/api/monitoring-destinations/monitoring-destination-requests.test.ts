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
import { Secret } from 'shared-types/api.types';
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
      credentialsSecretRef: { name: 'monitoring-destination-pmm-credentials' },
    },
    parameters: { url: 'https://pmm.example.com' },
  },
};

const saved: MonitoringDestination = {
  metadata: { name: 'pmm' },
  spec: { classRef: { name: 'pmm' } },
};

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

// namespaceDestinations: what the namespace lists after the save;
// existingSecret: what getSecret finds under a taken name.
const setup = (
  namespaceDestinations: MonitoringDestination[] = [],
  existingSecret: Secret = { metadata: { name: 'taken' } }
) => {
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
    getSecret: vi.fn(track('getSecret', existingSecret)),
    deleteSecret: vi.fn(track('deleteSecret', {})),
    createDestination: vi.fn(track('createDestination', saved)),
    updateDestination: vi.fn(track('updateDestination', saved)),
    deleteDestination: vi.fn(track('deleteDestination', {})),
    listDestinations: vi.fn(() => {
      calls.push('listDestinations');
      return Promise.resolve(namespaceDestinations);
    }),
  };
  // The request is still recorded, then fails.
  const fail = (
    name: Exclude<keyof MonitoringDestinationRequests, 'listDestinations'>,
    error: Error
  ) =>
    vi.mocked(requests[name]).mockImplementation((arg: unknown) => {
      calls.push(
        `${name} ${typeof arg === 'string' ? arg : JSON.stringify(arg)}`
      );
      return Promise.reject(error);
    });
  return { calls, requests, fail };
};

const token = { token: 's3cr3t' };

// The Secret the save creates: labelled for the destination's class and mode.
const credentialsSecret = (name: string) => ({
  metadata: {
    name,
    labels: {
      'openeverest.io/monitoring-class': 'pmm',
      'openeverest.io/definition': 'ServiceAccountToken',
    },
  },
  stringData: token,
});

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
          name: 'monitoring-destination-pmm-credentials',
          stringData: token,
        },
      },
      requests
    );

    expect(calls).toEqual([
      `createSecret ${JSON.stringify(credentialsSecret('monitoring-destination-pmm-credentials'))}`,
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
            name: 'monitoring-destination-pmm-credentials',
            stringData: token,
          },
        },
        requests
      )
    ).rejects.toThrow();

    expect(calls[calls.length - 1]).toBe(
      'deleteSecret monitoring-destination-pmm-credentials'
    );
  });

  const takenName = {
    namespace: 'team-a',
    isEdit: false,
    destination,
    credentialsSecret: {
      action: 'create' as const,
      name: 'monitoring-destination-pmm-credentials',
      stringData: token,
    },
  };

  it('reports a taken Secret name without creating the destination', async () => {
    const { requests, fail } = setup();
    fail('createSecret', httpError(409));

    await expect(
      saveMonitoringDestination(takenName, requests)
    ).rejects.toEqual(
      new SecretAlreadyExistsError(
        'monitoring-destination-pmm-credentials',
        false
      )
    );
    expect(requests.getSecret).toHaveBeenCalledWith(
      'monitoring-destination-pmm-credentials'
    );
    expect(requests.createDestination).not.toHaveBeenCalled();
    expect(requests.deleteSecret).not.toHaveBeenCalled();
  });

  it('offers a taken name for reuse when its Secret holds the class and mode credentials', async () => {
    const { requests, fail } = setup(
      [],
      credentialsSecret('monitoring-destination-pmm-credentials')
    );
    fail('createSecret', httpError(409));

    await expect(
      saveMonitoringDestination(takenName, requests)
    ).rejects.toEqual(
      new SecretAlreadyExistsError(
        'monitoring-destination-pmm-credentials',
        true
      )
    );
  });

  it('does not offer a taken name for reuse when its Secret cannot be read', async () => {
    const { requests, fail } = setup();
    fail('createSecret', httpError(409));
    fail('getSecret', httpError(403));

    await expect(
      saveMonitoringDestination(takenName, requests)
    ).rejects.toEqual(
      new SecretAlreadyExistsError(
        'monitoring-destination-pmm-credentials',
        false
      )
    );
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
          credentialsSecret: {
            action: 'reuse',
            name: 'monitoring-destination-pmm-credentials',
          },
        },
        requests
      )
    ).rejects.toThrow();

    expect(calls).toEqual([`createDestination ${JSON.stringify(destination)}`]);
  });

  const rotation = {
    action: 'replace' as const,
    name: 'monitoring-destination-pmm-credentials-k2x9a',
    stringData: token,
    previousName: 'monitoring-destination-pmm-credentials',
  };

  it('moves the destination to a new Secret before deleting the old one', async () => {
    const { calls, requests } = setup([
      stored('pmm', 'monitoring-destination-pmm-credentials-k2x9a'),
    ]);

    await saveMonitoringDestination(
      {
        namespace: 'team-a',
        isEdit: true,
        destination,
        credentialsSecret: rotation,
      },
      requests
    );

    expect(calls).toEqual([
      `createSecret ${JSON.stringify(credentialsSecret('monitoring-destination-pmm-credentials-k2x9a'))}`,
      `updateDestination ${JSON.stringify(destination)}`,
      'listDestinations',
      'deleteSecret monitoring-destination-pmm-credentials',
    ]);
  });

  it('keeps the old Secret when another destination uses it or the form did not create it', async () => {
    const shared = setup([
      stored('pmm', 'monitoring-destination-pmm-credentials-k2x9a'),
      stored('pmm-eu', 'monitoring-destination-pmm-credentials'),
    ]);
    await saveMonitoringDestination(
      {
        namespace: 'team-a',
        isEdit: true,
        destination,
        credentialsSecret: rotation,
      },
      shared.requests
    );

    const brought = setup([
      stored('pmm', 'monitoring-destination-pmm-credentials-k2x9a'),
    ]);
    await saveMonitoringDestination(
      {
        namespace: 'team-a',
        isEdit: true,
        destination,
        credentialsSecret: { ...rotation, previousName: 'team-pmm-token' },
      },
      brought.requests
    );

    expect(shared.requests.deleteSecret).not.toHaveBeenCalled();
    expect(brought.requests.deleteSecret).not.toHaveBeenCalled();
  });

  it('deletes the new Secret and keeps the old one when the update fails', async () => {
    const { calls, requests, fail } = setup();
    fail('updateDestination', httpError(409));

    await expect(
      saveMonitoringDestination(
        {
          namespace: 'team-a',
          isEdit: true,
          destination,
          credentialsSecret: rotation,
        },
        requests
      )
    ).rejects.toEqual(httpError(409));

    expect(calls.map((call) => call.split(' ')[0])).toEqual([
      'createSecret',
      'updateDestination',
      'deleteSecret',
    ]);
    expect(requests.deleteSecret).toHaveBeenCalledWith(
      'monitoring-destination-pmm-credentials-k2x9a'
    );
  });
});

describe('deleteMonitoringDestination', () => {
  it('deletes the destination and the Secret created with it', async () => {
    const { calls, requests } = setup();
    const pmm = stored('pmm', 'monitoring-destination-pmm-credentials');

    await deleteMonitoringDestination(
      { destination: pmm, namespaceDestinations: [pmm] },
      requests
    );

    expect(calls).toEqual([
      'deleteDestination pmm',
      'deleteSecret monitoring-destination-pmm-credentials',
    ]);
  });

  it('recognises a rotated credentials Secret as created by the form', async () => {
    const { calls, requests } = setup();
    const pmm = stored('pmm', 'monitoring-destination-pmm-credentials-k2x9a');

    await deleteMonitoringDestination(
      { destination: pmm, namespaceDestinations: [pmm] },
      requests
    );

    expect(calls).toEqual([
      'deleteDestination pmm',
      'deleteSecret monitoring-destination-pmm-credentials-k2x9a',
    ]);
  });

  it('keeps a Secret another destination uses or the form did not name', async () => {
    const { calls, requests } = setup();
    const pmm = stored('pmm', 'monitoring-destination-pmm-credentials');
    const sharing = stored('pmm-eu', 'monitoring-destination-pmm-credentials');
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
