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

import { InstanceCondition } from 'shared-types/instance.types';
import {
  getInstanceStatusDetails,
  INSTANCE_CONDITIONS,
  InstanceConditionEntry,
} from './instance-conditions';
import { getInstanceStatusFixture } from './tests/instance-status-fixtures';

const condition = (
  type: string,
  status: InstanceCondition['status'],
  reason = type
): InstanceCondition => ({
  type,
  status,
  reason,
  message: `${type} is ${status}`,
  lastTransitionTime: '2026-03-01T11:00:00Z',
});

const details = (...conditions: InstanceCondition[]) =>
  getInstanceStatusDetails({ status: { conditions } });

const types = (entries: InstanceConditionEntry[]) =>
  entries.map((entry) => entry.condition.type);

describe('INSTANCE_CONDITIONS', () => {
  it('lists each condition once', () => {
    const listed = INSTANCE_CONDITIONS.map(({ type }) => type);

    expect(new Set(listed).size).toBe(listed.length);
  });

  it('leaves monitoring and connection details out of what gates Ready', () => {
    const optional = INSTANCE_CONDITIONS.filter(
      ({ category, gatesReady }) => category === 'check' && !gatesReady
    ).map(({ type }) => type);

    expect(optional).toEqual([
      'ConnectionDetailsReady',
      'MonitoringConfigured',
      'MonitoringReady',
    ]);
  });
});

describe('getInstanceStatusDetails', () => {
  it('has nothing to say about an Instance without conditions', () => {
    expect(getInstanceStatusDetails(undefined)).toEqual({
      problems: [],
      passingChecks: [],
      openChecks: [],
      runningOperations: [],
      finishedOperations: [],
      notices: [],
      markers: [],
      conditions: [],
    });
  });

  it('reads a check as passing, in progress or failing', () => {
    const result = details(
      condition('Synced', 'True'),
      condition('PodsReady', 'Unknown', 'NotReady'),
      condition('PodsScheduled', 'False', 'Unschedulable')
    );

    expect(types(result.passingChecks)).toEqual(['Synced']);
    expect(types(result.openChecks)).toEqual(['PodsScheduled', 'PodsReady']);
    expect(types(result.problems)).toEqual(['PodsScheduled']);
  });

  it('does not count a check that is absent as failing', () => {
    const result = details(condition('Synced', 'True'));

    expect(types(result.openChecks)).toEqual([]);
    expect(result.problems).toEqual([]);
  });

  it('titles checks in plain language', () => {
    const [entry] = details(condition('PodsReady', 'True')).passingChecks;

    expect(entry.title).toBe('Pods ready');
  });

  it('names a problem that no check stands for after its reason', () => {
    const { instance } = getInstanceStatusFixture('rejected-spec');

    const { problems } = getInstanceStatusDetails(instance);

    expect(problems).toHaveLength(1);
    expect(problems[0]).toMatchObject({
      title: 'Invalid configuration',
      condition: {
        type: 'NeedsAttention',
        message: 'spec.components.engine.replicas: must be odd',
      },
    });
  });

  it('lists a problem once when a failing check already stands for it', () => {
    const { instance } = getInstanceStatusFixture('crash-loop-one-replica');

    expect(types(getInstanceStatusDetails(instance).problems)).toEqual([
      'PodsReady',
    ]);
  });

  it('lists the cause of the status before other problems', () => {
    const result = details(
      condition('NeedsAttention', 'True', 'CrashLoopBackOff'),
      condition('MonitoringReady', 'False', 'MonitoringNotReady'),
      condition('BackupConfigured', 'False', 'BackupStorageNotFound'),
      condition('PodsReady', 'False', 'CrashLoopBackOff')
    );

    expect(types(result.problems)).toEqual([
      'PodsReady',
      'BackupConfigured',
      'MonitoringReady',
    ]);
  });

  it('separates the operations that run from those that are over', () => {
    const result = details(
      condition('Upgrading', 'True', 'MinorVersionRolling'),
      condition('StorageResizing', 'False', 'Succeeded'),
      condition('Restoring', 'False', 'Failed')
    );

    expect(types(result.runningOperations)).toEqual(['Upgrading']);
    expect(types(result.finishedOperations)).toEqual([
      'Restoring',
      'StorageResizing',
    ]);
  });

  it('shows a notice only while it is True', () => {
    const result = details(
      condition('MaintenancePending', 'True', 'AwaitingApproval'),
      condition('ComponentVersionDeprecated', 'False', 'VersionsSupported')
    );

    expect(types(result.notices)).toEqual(['MaintenancePending']);
  });

  it('marks notices and the failing checks Ready does not wait for', () => {
    const result = details(
      condition('MaintenancePending', 'True', 'AwaitingApproval'),
      condition('MonitoringReady', 'False', 'MonitoringNotReady'),
      condition('ConnectionDetailsReady', 'Unknown'),
      // Already the status: a marker would say it twice.
      condition('PodsReady', 'False', 'CrashLoopBackOff')
    );

    expect(types(result.markers)).toEqual([
      'MaintenancePending',
      'MonitoringReady',
    ]);
  });

  it('keeps a condition type it does not know, without reading it', () => {
    const custom = condition('example.com/ShardsBalanced', 'False');

    const result = details(condition('Synced', 'True'), custom);

    expect(result.conditions).toContainEqual(custom);
    expect(result.problems).toEqual([]);
    expect(result.openChecks).toEqual([]);
    expect(result.markers).toEqual([]);
  });
});
