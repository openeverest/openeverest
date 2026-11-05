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

import { Instance, Provider } from 'shared-types/api.types';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { MonitoringChange } from './monitoring-edit-modal.types';
import {
  getMonitoringDisruption,
  hasSingleReplicaMonitoredComponent,
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
  spec: {
    controllerName: `openeverest.io/monitoring-${name}`,
    displayName,
    authModes: [{ name: 'Anonymous' }],
  },
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

const entry = (name: string, parameters?: Record<string, unknown>) => ({
  destinationRef: { name },
  parameters,
});

const disruption = (
  before: string[],
  after: string[],
  hasSingleReplica = false
) =>
  getMonitoringDisruption({
    before: before.map((name) => entry(name)),
    after: after.map((name) => entry(name)),
    destinations,
    classes,
    provider: PROVIDER,
    hasSingleReplica,
  });

describe('getMonitoringDisruption', () => {
  it('warns when an added or removed class rolls the pods of the provider', () => {
    expect(disruption([], ['pmm-a'])).toEqual({
      restarts: [{ change: MonitoringChange.Enable, classTitle: 'PMM' }],
      singleReplica: false,
    });
    expect(disruption(['pmm-a', 'coroot'], ['coroot'], true)).toEqual({
      restarts: [{ change: MonitoringChange.Remove, classTitle: 'PMM' }],
      singleReplica: true,
    });
  });

  it('warns when the parameters of a kept entry change, in any key order', () => {
    const change = (
      before: Record<string, unknown>,
      after: Record<string, unknown>
    ) =>
      getMonitoringDisruption({
        before: [entry('pmm-a', before)],
        after: [entry('pmm-a', after)],
        destinations,
        classes,
        provider: PROVIDER,
        hasSingleReplica: false,
      }).restarts;

    expect(change({ cpu: '1' }, { cpu: '2' })).toEqual([
      { change: MonitoringChange.Update, classTitle: 'PMM' },
    ]);
    expect(change({ cpu: '1', memory: '1G' }, { memory: '1G', cpu: '1' })).toEqual(
      []
    );
  });

  it('stays quiet for kept destinations and classes without a restart', () => {
    expect(disruption(['pmm-a'], ['pmm-a', 'coroot'], true)).toEqual({
      restarts: [],
      singleReplica: false,
    });
    expect(disruption([], ['prom', 'gone'])).toEqual({
      restarts: [],
      singleReplica: false,
    });
  });
});

describe('hasSingleReplicaMonitoredComponent', () => {
  const provider: Provider = {
    metadata: { name: PROVIDER },
    spec: { monitoring: { componentTypes: { mongod: { kind: 'mongodb' } } } },
  };
  const withComponents = (
    components: NonNullable<Instance['spec']>['components']
  ): Instance => ({
    metadata: { name: 'db' },
    spec: { providerRef: { name: PROVIDER }, components },
  });

  it('looks only at components the provider monitors', () => {
    expect(
      hasSingleReplicaMonitoredComponent(
        withComponents({
          engine: { type: 'mongod', replicas: 1 },
          backupAgent: { type: 'backup', replicas: 1 },
        }),
        provider
      )
    ).toBe(true);
    expect(
      hasSingleReplicaMonitoredComponent(
        withComponents({
          engine: { type: 'mongod', replicas: 3 },
          backupAgent: { type: 'backup', replicas: 1 },
        }),
        provider
      )
    ).toBe(false);
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
