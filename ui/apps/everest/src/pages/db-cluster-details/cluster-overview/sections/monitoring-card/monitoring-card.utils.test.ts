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

import { Instance } from 'shared-types/api.types';
import { MONITORING_READY_CONDITION } from './monitoring-card.constants';
import { ConditionStatus } from './monitoring-card.types';
import { getMonitoringOverview } from './monitoring-card.utils';

const condition = (
  type: string,
  status: ConditionStatus,
  reason: string,
  message = ''
) => ({
  type,
  status,
  reason,
  message,
  lastTransitionTime: '2026-01-01T00:00:00Z',
});

const instanceWith = (
  conditions: ReturnType<typeof condition>[]
): Instance => ({
  metadata: { name: 'orders-db', namespace: 'team-a' },
  spec: {
    providerRef: { name: 'percona-server-mongodb' },
    monitoring: {
      destinations: [{ name: 'pmm', destinationRef: { name: 'pmm-prod' } }],
    },
  },
  status: {
    monitoring: {
      destinations: [{ name: 'pmm', mode: 'ProviderManaged', conditions }],
    },
  },
});

describe('getMonitoringOverview', () => {
  it.each([
    {
      name: 'ready: explained by Configured',
      conditions: [
        condition('Configured', 'True', 'Configured', 'Rendered by provider x'),
        condition(
          'Ready',
          'True',
          'Ready',
          'Configured, and pmm-prod is Ready'
        ),
      ],
      ready: 'True',
      details: 'Configured: Rendered by provider x',
    },
    {
      name: 'configured, but the destination is not ready: explained by Ready',
      conditions: [
        condition('Configured', 'True', 'AgentTLSNotVerified', 'no TLS check'),
        condition(
          'Ready',
          'False',
          'DestinationNotReady',
          'pmm-prod: Unauthorized'
        ),
      ],
      ready: 'False',
      details: 'DestinationNotReady: pmm-prod: Unauthorized',
    },
    {
      name: 'not configured: explained by Configured',
      conditions: [
        condition(
          'Configured',
          'False',
          'RenderFailed',
          'destination has no url'
        ),
        condition(
          'Ready',
          'False',
          'NotConfigured',
          'Binding is not Configured'
        ),
      ],
      ready: 'False',
      details: 'RenderFailed: destination has no url',
    },
    {
      name: 'destination not probed yet: explained by Ready',
      conditions: [
        condition('Configured', 'True', 'Configured', 'Rendered by provider x'),
        condition('Ready', 'Unknown', 'Pending', 'Waiting for pmm-prod'),
      ],
      ready: 'Unknown',
      details: 'Pending: Waiting for pmm-prod',
    },
    {
      name: 'not reported yet',
      conditions: [],
      ready: 'Unknown',
      details: '',
    },
    {
      name: 'configured, but Ready not reported: explained by Configured',
      conditions: [
        condition('Configured', 'True', 'Configured', 'Rendered by provider x'),
      ],
      ready: 'Unknown',
      details: 'Configured: Rendered by provider x',
    },
  ])('$name', ({ conditions, ready, details }) => {
    const [entry] = getMonitoringOverview(instanceWith(conditions)).entries;

    expect(entry).toEqual({
      name: 'pmm',
      destination: 'pmm-prod',
      mode: 'ProviderManaged',
      ready,
      details,
    });
  });

  it('reads the roll-up from MonitoringReady', () => {
    const instance = instanceWith([]);
    const withRollUps: Instance = {
      ...instance,
      status: {
        ...instance.status,
        conditions: [
          condition('MonitoringConfigured', 'True', 'MonitoringConfigured'),
          condition(
            MONITORING_READY_CONDITION,
            'False',
            'MonitoringNotReady',
            'pmm: DestinationNotReady'
          ),
        ],
      },
    };

    expect(getMonitoringOverview(withRollUps).ready).toEqual({
      status: 'False',
      details: 'MonitoringNotReady: pmm: DestinationNotReady',
    });
  });
});
