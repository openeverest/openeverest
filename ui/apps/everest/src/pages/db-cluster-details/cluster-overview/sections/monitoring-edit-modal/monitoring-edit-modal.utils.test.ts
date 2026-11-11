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
  getSingleReplicaComponents,
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
  singleReplicaComponents: string[] = []
) =>
  getMonitoringDisruption({
    before: before.map((name) => entry(name)),
    after: after.map((name) => entry(name)),
    destinations,
    classes,
    provider: PROVIDER,
    singleReplicaComponents,
  });

describe('getMonitoringDisruption', () => {
  it('warns when an added or removed class rolls the pods of the provider', () => {
    expect(disruption([], ['pmm-a'])).toEqual({
      restarts: [{ change: MonitoringChange.Enable, classTitle: 'PMM' }],
      singleReplicaComponents: [],
    });
    expect(disruption(['pmm-a', 'coroot'], ['coroot'], ['engine'])).toEqual({
      restarts: [{ change: MonitoringChange.Remove, classTitle: 'PMM' }],
      singleReplicaComponents: ['engine'],
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
        singleReplicaComponents: [],
      }).restarts;

    expect(change({ cpu: '1' }, { cpu: '2' })).toEqual([
      { change: MonitoringChange.Update, classTitle: 'PMM' },
    ]);
    expect(
      change({ cpu: '1', memory: '1G' }, { memory: '1G', cpu: '1' })
    ).toEqual([]);
  });

  it('stays quiet for kept destinations and classes without a restart', () => {
    expect(disruption(['pmm-a'], ['pmm-a', 'coroot'], ['engine'])).toEqual({
      restarts: [],
      singleReplicaComponents: [],
    });
    expect(disruption([], ['prom', 'gone'])).toEqual({
      restarts: [],
      singleReplicaComponents: [],
    });
  });
});

describe('getSingleReplicaComponents', () => {
  type Components = NonNullable<Instance['spec']>['components'];
  const instance = (
    components: Components,
    running: NonNullable<Instance['status']>['components'] = []
  ): Instance => ({
    metadata: { name: 'db' },
    spec: { providerRef: { name: PROVIDER }, components },
    status: { components: running },
  });

  // PostgreSQL declares no monitored component types, so none may be skipped.
  it.each<{ provider: string; components: Components; expected: string[] }>([
    {
      provider: 'PostgreSQL engine',
      components: {
        engine: { type: 'postgresql', replicas: 1 },
        proxy: { type: 'pgbouncer', replicas: 2 },
      },
      expected: ['engine'],
    },
    {
      provider: 'sharded MongoDB mongos',
      components: {
        engine: { type: 'mongod', replicas: 3 },
        proxy: { type: 'mongos', replicas: 1 },
        configServer: { type: 'cfg', replicas: 3 },
      },
      expected: ['proxy'],
    },
    {
      provider: 'MySQL proxysql',
      components: {
        engine: { type: 'pxc', replicas: 3 },
        proxy: { type: 'proxysql', replicas: 1 },
      },
      expected: ['proxy'],
    },
  ])('names a single-replica $provider', ({ components, expected }) => {
    expect(getSingleReplicaComponents(instance(components))).toEqual(expected);
  });

  it('takes the running replicas when the spec leaves them out, and counts an unknown one as single', () => {
    expect(
      getSingleReplicaComponents(
        instance({ engine: { type: 'mongod' }, proxy: { type: 'mongos' } }, [
          { name: 'engine', replicas: 3 },
        ])
      )
    ).toEqual(['proxy']);
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
