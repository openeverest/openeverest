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
  MonitoringDestinationRequests,
  saveMonitoringDestination,
  SecretAlreadyExistsError,
  SecretChangedMeanwhileError,
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

// existingSecret: what getSecret finds under the name.
const setup = (
  existingSecret: Secret = { metadata: { name: 'taken', resourceVersion: '7' } }
) => {
  const calls: string[] = [];
  const describeCall = (name: string, args: unknown[]) =>
    [
      name,
      ...args
        .filter((arg) => arg !== undefined)
        .map((arg) => (typeof arg === 'string' ? arg : JSON.stringify(arg))),
    ].join(' ');
  const track =
    <T>(name: string, result: T) =>
    (...args: unknown[]) => {
      calls.push(describeCall(name, args));
      return Promise.resolve(result);
    };
  const requests: MonitoringDestinationRequests = {
    createSecret: vi.fn(track('createSecret', {})),
    getSecret: vi.fn(track('getSecret', existingSecret)),
    patchSecret: vi.fn(track('patchSecret', {})),
    deleteSecret: vi.fn(track('deleteSecret', {})),
    createDestination: vi.fn(track('createDestination', saved)),
    updateDestination: vi.fn(track('updateDestination', saved)),
  };
  // The request is still recorded, then fails.
  const fail = (name: keyof MonitoringDestinationRequests, error: Error) =>
    vi.mocked(requests[name]).mockImplementation((...args: unknown[]) => {
      calls.push(describeCall(name, args));
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
      'monitoring-destination-pmm-credentials',
      true
    );
    expect(requests.createDestination).not.toHaveBeenCalled();
    expect(requests.deleteSecret).not.toHaveBeenCalled();
  });

  it('offers a taken name for reuse when its Secret holds the class and mode credentials', async () => {
    const { requests, fail } = setup(
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

  it('writes no Secret for one the destination keeps or picks, and never deletes it', async () => {
    const { calls, requests, fail } = setup();
    fail('createDestination', httpError(500));

    await expect(
      saveMonitoringDestination(
        { namespace: 'team-a', isEdit: false, destination },
        requests
      )
    ).rejects.toThrow();

    expect(calls).toEqual([`createDestination ${JSON.stringify(destination)}`]);
  });

  const update = {
    namespace: 'team-a',
    isEdit: true,
    destination,
    credentialsSecret: {
      action: 'update' as const,
      name: 'monitoring-destination-pmm-credentials',
      stringData: token,
    },
  };

  it('updates the Secret in place at the version it read, before the destination', async () => {
    const { calls, requests } = setup();

    await saveMonitoringDestination(update, requests);

    expect(calls).toEqual([
      'getSecret monitoring-destination-pmm-credentials',
      `patchSecret monitoring-destination-pmm-credentials ${JSON.stringify({
        metadata: { resourceVersion: '7' },
        stringData: token,
      })}`,
      `updateDestination ${JSON.stringify(destination)}`,
    ]);
  });

  it('reports a Secret that changed meanwhile without saving the destination', async () => {
    const { requests, fail } = setup();
    fail('patchSecret', httpError(409));

    await expect(saveMonitoringDestination(update, requests)).rejects.toEqual(
      new SecretChangedMeanwhileError('monitoring-destination-pmm-credentials')
    );
    expect(requests.updateDestination).not.toHaveBeenCalled();
  });

  it('keeps the updated Secret when the destination is refused', async () => {
    const { requests, fail } = setup();
    fail('updateDestination', httpError(409));

    await expect(saveMonitoringDestination(update, requests)).rejects.toEqual(
      httpError(409)
    );
    expect(requests.deleteSecret).not.toHaveBeenCalled();
  });
});
