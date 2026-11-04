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

import { Secret } from 'shared-types/api.types';
import {
  MonitoringClass,
  MonitoringDestination,
  MonitoringDestinationState,
} from 'shared-types/monitoring-destinations.types';
import {
  getCredentialsSecretLabels,
  getDefaultCredentialsSecretName,
  getDestinationAddress,
  getDestinationStatus,
  getRotatedCredentialsSecretName,
  isCredentialsSecretFor,
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
    spec: {
      controllerName: 'openeverest.io/monitoring-pmm',
      authModes: [{ name: 'Anonymous' }],
    },
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

describe('credentials Secret names', () => {
  it('names the Secret after the kind and the destination', () => {
    expect(getDefaultCredentialsSecretName('pmm')).toBe(
      'monitoring-destination-pmm-credentials'
    );
  });

  it('gives rotated credentials a fresh name within the Secret name limit', () => {
    expect(getRotatedCredentialsSecretName('pmm', 'k2x9a')).toBe(
      'monitoring-destination-pmm-credentials-k2x9a'
    );
    expect(getRotatedCredentialsSecretName('pmm')).toMatch(
      /^monitoring-destination-pmm-credentials-[a-z0-9]+$/
    );
    expect(
      getRotatedCredentialsSecretName('x'.repeat(260), 'k2x9a')
    ).toHaveLength(253);
  });
});

describe('credentials Secret labels', () => {
  it('labels the Secret with the class and the authentication mode', () => {
    expect(getCredentialsSecretLabels('pmm', 'ServiceAccountToken')).toEqual({
      'openeverest.io/monitoring-class': 'pmm',
      'openeverest.io/definition': 'ServiceAccountToken',
    });
  });

  it('reuses only a Secret labelled for the same class and mode', () => {
    const secret = (labels?: Record<string, string>): Secret => ({
      metadata: { name: 'monitoring-destination-pmm-credentials', labels },
    });
    const labels = getCredentialsSecretLabels('pmm', 'ServiceAccountToken');

    expect(
      isCredentialsSecretFor(secret(labels), 'pmm', 'ServiceAccountToken')
    ).toBe(true);
    expect(
      isCredentialsSecretFor(secret(labels), 'grafana', 'ServiceAccountToken')
    ).toBe(false);
    expect(isCredentialsSecretFor(secret(labels), 'pmm', 'BasicAuth')).toBe(
      false
    );
    expect(isCredentialsSecretFor(secret(), 'pmm', 'ServiceAccountToken')).toBe(
      false
    );
  });
});
