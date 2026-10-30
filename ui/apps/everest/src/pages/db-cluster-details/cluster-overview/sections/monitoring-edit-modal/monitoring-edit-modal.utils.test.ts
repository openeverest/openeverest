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
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { MonitoringChange } from './monitoring-edit-modal.types';
import {
  getMonitoringDisruption,
  withMonitoringDestinations,
} from './monitoring-edit-modal.utils';

const PROVIDER = 'percona-server-mongodb';

const monitoringClass = (
  name: string,
  displayName: string,
  disruption: 'None' | 'RollingRestart',
  provider = PROVIDER
): MonitoringClass => ({
  metadata: { name },
  spec: { displayName, authModes: [{ name: 'Anonymous' }] },
  status: { providers: [{ name: provider, disruption }] },
});

const destination = (
  name: string,
  className: string
): MonitoringDestination => ({
  metadata: { name },
  spec: { classRef: { name: className } },
});

const classes = [
  monitoringClass('pmm', 'PMM', 'RollingRestart'),
  monitoringClass('coroot', 'Coroot', 'None'),
  monitoringClass('prometheus', 'Prometheus', 'RollingRestart', 'other'),
];
const destinations = [
  destination('pmm-a', 'pmm'),
  destination('coroot', 'coroot'),
  destination('prom', 'prometheus'),
];

const disruption = (
  before: string[],
  after: string[],
  engineReplicas?: number
) =>
  getMonitoringDisruption({
    before,
    after,
    destinations,
    classes,
    provider: PROVIDER,
    engineReplicas,
  });

describe('getMonitoringDisruption', () => {
  it('warns when an added or removed class rolls the pods of the provider', () => {
    expect(disruption([], ['pmm-a'], 3)).toEqual({
      restarts: [{ change: MonitoringChange.Enable, classTitle: 'PMM' }],
      singleReplica: false,
    });
    expect(disruption(['pmm-a', 'coroot'], ['coroot'], 1)).toEqual({
      restarts: [{ change: MonitoringChange.Remove, classTitle: 'PMM' }],
      singleReplica: true,
    });
  });

  it('stays quiet for kept destinations and classes without a restart', () => {
    expect(disruption(['pmm-a'], ['pmm-a', 'coroot'], 1)).toEqual({
      restarts: [],
      singleReplica: false,
    });
    expect(disruption([], ['prom', 'gone'])).toEqual({
      restarts: [],
      singleReplica: false,
    });
  });
});

describe('withMonitoringDestinations', () => {
  const instance: Instance = {
    metadata: { name: 'db', resourceVersion: '7' },
    spec: {
      providerRef: { name: PROVIDER },
      monitoring: {
        destinations: [{ name: 'pmm-a', destinationRef: { name: 'pmm-a' } }],
      },
    },
  };

  it('writes the entries onto a copy of the Instance', () => {
    const updated = withMonitoringDestinations(instance, [
      {
        name: 'coroot',
        destinationRef: { name: 'coroot' },
        parameters: { interval: '30s' },
      },
    ]);

    expect(updated.spec?.monitoring).toEqual({
      destinations: [
        {
          name: 'coroot',
          destinationRef: { name: 'coroot' },
          parameters: { interval: '30s' },
        },
      ],
    });
    expect(updated.metadata).toEqual(instance.metadata);
    expect(instance.spec?.monitoring?.destinations).toHaveLength(1);
  });

  it('drops spec.monitoring when no destination is left', () => {
    expect(withMonitoringDestinations(instance, []).spec).toEqual({
      providerRef: { name: PROVIDER },
    });
  });
});
