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
  MonitoringClass,
  MonitoringDestination,
  MonitoringDestinationState,
} from 'shared-types/monitoring-destinations.types';
import {
  getDestinationAddress,
  getDestinationStatus,
} from './monitoring-destinations';

type Condition = {
  type: string;
  status: 'True' | 'False' | 'Unknown';
  reason: string;
  message?: string;
};

const withStatus = (
  conditions: Condition[],
  serverVersion?: string
): MonitoringDestination => ({
  metadata: { name: 'pmm' },
  spec: { classRef: { name: 'pmm' } },
  status: {
    conditions: conditions.map((condition) => ({
      message: '',
      ...condition,
      lastTransitionTime: '2026-01-01T00:00:00Z',
    })),
    serverVersion,
  },
});

const accepted: Condition = {
  type: 'Accepted',
  status: 'True',
  reason: 'Accepted',
};

describe('getDestinationStatus', () => {
  it('is ready with the server version once the class controller is', () => {
    expect(
      getDestinationStatus(
        withStatus(
          [accepted, { type: 'Ready', status: 'True', reason: 'Ready' }],
          '3.1.0'
        )
      )
    ).toEqual({
      state: MonitoringDestinationState.Ready,
      serverVersion: '3.1.0',
    });
  });

  it('explains a refusal by core first', () => {
    expect(
      getDestinationStatus(
        withStatus([
          {
            type: 'Accepted',
            status: 'False',
            reason: 'CredentialsNotFound',
            message: 'Secret pmm-credentials not found',
          },
          { type: 'Ready', status: 'False', reason: 'Unauthorized' },
        ])
      )
    ).toEqual({
      state: MonitoringDestinationState.Failing,
      reason: 'CredentialsNotFound',
      message: 'Secret pmm-credentials not found',
      serverVersion: undefined,
    });
  });

  it('fails with the Ready reason when the backend is not usable', () => {
    expect(
      getDestinationStatus(
        withStatus([
          accepted,
          {
            type: 'Ready',
            status: 'False',
            reason: 'Unreachable',
            message: 'dial tcp: i/o timeout',
          },
        ])
      )
    ).toMatchObject({
      state: MonitoringDestinationState.Failing,
      reason: 'Unreachable',
      message: 'dial tcp: i/o timeout',
    });
  });

  it('is pending before the class controller checked it', () => {
    expect(getDestinationStatus(withStatus([accepted])).state).toBe(
      MonitoringDestinationState.Pending
    );
    expect(
      getDestinationStatus(
        withStatus([
          accepted,
          { type: 'Ready', status: 'False', reason: 'Pending' },
        ])
      )
    ).toMatchObject({
      state: MonitoringDestinationState.Pending,
      reason: 'Pending',
    });
    expect(getDestinationStatus(withStatus([])).state).toBe(
      MonitoringDestinationState.Pending
    );
  });
});

const classWithUiSchema = (uiSchema: unknown): MonitoringClass => {
  const monitoringClass: MonitoringClass = {
    metadata: { name: 'pmm' },
    spec: { authModes: [{ name: 'Anonymous' }] },
  };
  Reflect.set(monitoringClass.spec, 'uiSchema', uiSchema);
  return monitoringClass;
};

describe('getDestinationAddress', () => {
  const destination = withStatus([]);
  Reflect.set(destination.spec, 'parameters', {
    url: 'https://pmm.example.com',
  });

  it('reads the parameter the class form keeps at parameters.url', () => {
    const pmm = classWithUiSchema({
      destination: {
        components: {
          server: {
            uiType: 'group',
            components: { url: { path: 'parameters.url', uiType: 'text' } },
          },
        },
      },
    });

    expect(getDestinationAddress(destination, pmm)).toBe(
      'https://pmm.example.com'
    );
  });

  it('has none for classes without an address field', () => {
    expect(
      getDestinationAddress(destination, classWithUiSchema(undefined))
    ).toBe(undefined);
    expect(getDestinationAddress(destination, undefined)).toBe(undefined);
  });
});
