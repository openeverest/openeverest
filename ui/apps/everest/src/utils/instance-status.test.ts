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
  InstanceCondition,
  InstanceStatusDisplay,
  InstanceStatusSource,
} from 'shared-types/instance.types';
import {
  canBackUpNow,
  compareInstanceStatus,
  getInstanceStatusDisplay,
  getInstanceStatusExplanation,
  getReasonLabel,
  hasConnectionDetails,
  getRunningRestore,
  isInstanceDeleting,
} from './instance-status';
import { REASON_LABELS } from './instance-status.messages';
import {
  INSTANCE_STATUS_FIXTURES,
  INSTANCE_STATUS_FIXTURE_LABELS,
} from './tests/instance-status-fixtures';

const NOW = new Date('2026-03-01T12:00:00Z');

const ago = (minutes: number) =>
  new Date(NOW.getTime() - minutes * 60 * 1000).toISOString();

// A write to the object, or to one of its subresources.
const written = (subresource: string, minutesAgo: number) => ({
  time: ago(minutesAgo),
  ...(subresource && { subresource }),
});

const condition = (
  type: string,
  status: InstanceCondition['status'],
  reason = type,
  lastTransitionTime = '2026-03-01T11:00:00Z'
): InstanceCondition => ({
  type,
  status,
  reason,
  message: `${type} is ${status}`,
  lastTransitionTime,
});

const instance = (
  conditions: InstanceCondition[],
  overrides: InstanceStatusSource = {}
): InstanceStatusSource => ({
  metadata: {
    generation: 2,
    creationTimestamp: '2026-01-20T12:00:00Z',
    ...overrides.metadata,
  },
  status: {
    phase: 'Provisioned',
    observedGeneration: 2,
    conditions,
    ...overrides.status,
  },
});

describe('getInstanceStatusDisplay', () => {
  it('runs the fixtures it shares with everestctl', () => {
    expect(INSTANCE_STATUS_FIXTURES.length).toBeGreaterThan(0);
  });

  it.each(INSTANCE_STATUS_FIXTURES.map((fixture) => [fixture.name, fixture]))(
    '%s',
    (_, fixture) => {
      const display = getInstanceStatusDisplay(
        fixture.instance,
        new Date(fixture.now)
      );

      expect({
        tone: display.tone,
        reason: display.reason,
        label: display.label,
        available: display.available,
        sentence: display.sentence,
        since: display.since,
      }).toEqual({
        tone: fixture.want.tone,
        reason: fixture.want.reason,
        label: fixture.want.label,
        available: fixture.want.available,
        sentence: fixture.want.sentence,
        since: fixture.want.since,
      });
    }
  );

  // The cases below are the ones the shared fixtures leave open. everestctl's
  // tests have the same ones, with the same answers.
  const HEALTHY = [
    condition('Available', 'True', 'Available', ago(24 * 60)),
    condition('Ready', 'True', 'Ready', ago(60)),
    condition('Reconciling', 'False', 'NoProblems', ago(60)),
    condition('Stalled', 'False', 'NoProblems', ago(60)),
    condition('NeedsAttention', 'False', 'NoProblems', ago(60)),
  ];

  it.each<
    [
      string,
      InstanceStatusSource | undefined,
      Partial<InstanceStatusDisplay> & { since: string | null },
    ]
  >([
    [
      'no metadata and no status',
      undefined,
      {
        tone: 'progress',
        reason: 'Creating',
        sentence: 'Creating',
        message: '',
        since: null,
        stale: false,
      },
    ],
    [
      'reconciled, but no Ready condition yet',
      {
        metadata: { generation: 1, creationTimestamp: ago(60) },
        status: { phase: 'Pending', observedGeneration: 1 },
      },
      {
        tone: 'progress',
        reason: 'Creating',
        sentence: 'Creating',
        message: '',
        since: null,
        stale: false,
      },
    ],
    [
      'no Ready condition on a database that has been up',
      {
        metadata: { generation: 1 },
        status: {
          phase: 'Provisioned',
          observedGeneration: 1,
          conditions: [
            condition('Reconciling', 'True', 'ApplyingChanges', ago(3)),
          ],
        },
      },
      {
        tone: 'progress',
        reason: 'ApplyingChanges',
        sentence: 'Applying changes',
        message: '',
        since: ago(3),
        stale: false,
      },
    ],
    [
      'spec edited a moment ago: the old status decides nothing',
      {
        metadata: {
          generation: 2,
          managedFields: [written('', 1), written('status', 60)],
        },
        status: {
          phase: 'Provisioned',
          observedGeneration: 1,
          conditions: HEALTHY,
        },
      },
      {
        tone: 'progress',
        reason: 'ApplyingChanges',
        sentence: 'Up \u00b7 Applying changes',
        message: '',
        since: null,
        stale: true,
      },
    ],
    [
      'spec edited long ago and never observed: nobody is reconciling',
      {
        metadata: {
          generation: 2,
          managedFields: [written('', 6), written('', 60)],
        },
        status: {
          phase: 'Provisioned',
          observedGeneration: 1,
          conditions: HEALTHY,
        },
      },
      {
        tone: 'unknown',
        reason: 'NotPickedUp',
        sentence: 'Up \u00b7 Not picked up by provider',
        message: '',
        since: null,
        stale: true,
      },
    ],
    [
      'a status write does not start the clock again',
      {
        metadata: {
          generation: 2,
          managedFields: [written('', 6), written('status', 0)],
        },
        status: {
          phase: 'Provisioned',
          observedGeneration: 1,
          conditions: HEALTHY,
        },
      },
      {
        tone: 'unknown',
        reason: 'NotPickedUp',
        sentence: 'Up \u00b7 Not picked up by provider',
        message: '',
        since: null,
        stale: true,
      },
    ],
    [
      // The API leaves the managed fields out, so this is what the UI sees.
      'spec edited, with nothing to time the wait by',
      {
        metadata: { generation: 2, creationTimestamp: ago(40 * 24 * 60) },
        status: {
          phase: 'Provisioned',
          observedGeneration: 1,
          conditions: HEALTHY,
        },
      },
      {
        tone: 'progress',
        reason: 'ApplyingChanges',
        sentence: 'Up \u00b7 Applying changes',
        message: '',
        since: null,
        stale: true,
      },
    ],
    [
      'deletion requested, not observed yet: the old status word is not shown',
      {
        metadata: { generation: 3, deletionTimestamp: ago(1) },
        status: {
          phase: 'Provisioned',
          observedGeneration: 2,
          conditions: HEALTHY,
        },
      },
      {
        tone: 'deleting',
        reason: 'Deleting',
        sentence: 'Deleting',
        message: '',
        since: ago(1),
        stale: true,
      },
    ],
    [
      'being resumed: still stopped, and the change is in progress',
      {
        metadata: { generation: 3 },
        status: {
          phase: 'Provisioned',
          observedGeneration: 2,
          conditions: [
            condition('Available', 'False', 'Suspended', ago(60)),
            condition('Ready', 'True', 'Suspended', ago(60)),
            condition('Suspended', 'True', 'Suspended', ago(60)),
          ],
        },
      },
      {
        tone: 'stopped',
        reason: 'ApplyingChanges',
        sentence: 'Applying changes',
        message: '',
        since: null,
        stale: true,
      },
    ],
    [
      'a stall wins over a problem',
      {
        metadata: { generation: 1 },
        status: {
          phase: 'Provisioned',
          observedGeneration: 1,
          conditions: [
            condition('Available', 'False', 'NotAvailable', ago(60)),
            condition('Ready', 'False', 'DiskFull', ago(60)),
            condition('Stalled', 'True', 'DiskFull', ago(20)),
            condition('NeedsAttention', 'True', 'DiskFull', ago(30)),
          ],
        },
      },
      {
        tone: 'error',
        reason: 'DiskFull',
        sentence: 'Down \u00b7 Disk full',
        message: 'Ready is False',
        since: ago(20),
        stale: false,
      },
    ],
    [
      'a finished operation no longer supplies the time',
      {
        metadata: { generation: 1 },
        status: {
          phase: 'Provisioned',
          observedGeneration: 1,
          conditions: [
            ...HEALTHY,
            condition('Upgrading', 'False', 'Succeeded', ago(1)),
          ],
        },
      },
      {
        tone: 'ok',
        reason: 'Ready',
        sentence: 'Up \u00b7 Ready',
        message: 'Ready is True',
        since: ago(60),
        stale: false,
      },
    ],
  ])('%s', (_, source, want) => {
    expect(getInstanceStatusDisplay(source, NOW)).toMatchObject(want);
  });

  it('takes the tone from condition statuses whatever the reason says', () => {
    const display = getInstanceStatusDisplay(
      instance([
        condition('Ready', 'False', 'Ready'),
        condition('Stalled', 'True', 'Ready'),
      ]),
      NOW
    );

    expect(display).toMatchObject({ tone: 'error', label: 'Ready' });
  });
});

describe('getInstanceStatusExplanation', () => {
  it("is the status's own message when it has one", () => {
    expect(
      getInstanceStatusExplanation({
        message: 'Rolling out version 8.0.12',
        stale: false,
        tone: 'progress',
      })
    ).toBe('Rolling out version 8.0.12');
  });

  it('adds nothing to a current status without a message', () => {
    expect(
      getInstanceStatusExplanation({ message: '', stale: false, tone: 'ok' })
    ).toBe('');
  });

  it.each(['progress', 'stopped', 'unknown', 'deleting'] as const)(
    'says why a %s status that describes an older spec has no message',
    (tone) => {
      expect(
        getInstanceStatusExplanation({ message: '', stale: true, tone })
      ).not.toBe('');
    }
  );
});

describe('getReasonLabel', () => {
  it('has the label of every reason in the shared dictionary', () => {
    Object.entries(INSTANCE_STATUS_FIXTURE_LABELS).forEach(
      ([reason, label]) => {
        expect(getReasonLabel(reason), reason).toBe(label);
      }
    );
  });

  it('knows no reason the shared dictionary lacks', () => {
    expect(REASON_LABELS).toEqual(INSTANCE_STATUS_FIXTURE_LABELS);
  });

  it.each([
    ['CrashLoopBackOff', 'Crash loop'],
    ['RebalancingShards', 'Rebalancing shards'],
    ['WaitingForBackups', 'Waiting for backups'],
    ['PodsReady', 'Pods ready'],
    // A run of capitals is one word.
    ['PVCResizeFailed', 'PVC resize failed'],
    ['S3Unreachable', 'S3 unreachable'],
    ['PlanB', 'Plan B'],
    ['Paused', 'Paused'],
    ['', ''],
    // Not a label just because every object has a property of that name.
    ['constructor', 'constructor'],
  ])('reads %s as "%s", as everestctl does', (reason, label) => {
    expect(getReasonLabel(reason)).toBe(label);
  });
});

describe('compareInstanceStatus', () => {
  it('puts what needs someone first, then the databases that are down', () => {
    const rows = [
      { tone: 'ok', available: 'Yes' },
      { tone: 'deleting', available: 'Unknown' },
      { tone: 'progress', available: 'Yes' },
      { tone: 'stopped', available: 'No' },
      { tone: 'progress', available: 'Unknown' },
      { tone: 'unknown', available: 'Unknown' },
      { tone: 'attention', available: 'Yes' },
      { tone: 'progress', available: 'No' },
      { tone: 'error', available: 'Yes' },
      { tone: 'error', available: 'No' },
    ] as const;

    expect([...rows].sort(compareInstanceStatus)).toEqual([
      { tone: 'error', available: 'No' },
      { tone: 'error', available: 'Yes' },
      { tone: 'attention', available: 'Yes' },
      { tone: 'unknown', available: 'Unknown' },
      { tone: 'progress', available: 'No' },
      { tone: 'progress', available: 'Unknown' },
      { tone: 'progress', available: 'Yes' },
      { tone: 'stopped', available: 'No' },
      { tone: 'deleting', available: 'Unknown' },
      { tone: 'ok', available: 'Yes' },
    ]);
  });
});

describe('what the UI allows', () => {
  it('blocks changes once deletion is requested, whatever the status says', () => {
    expect(isInstanceDeleting(instance([condition('Ready', 'True')]))).toBe(
      false
    );
    expect(
      isInstanceDeleting(
        instance([condition('Ready', 'True')], {
          metadata: { deletionTimestamp: '2026-03-01T11:59:00Z' },
        })
      )
    ).toBe(true);
  });

  it('shows connection details while a change rolls out', () => {
    const upgrading = instance([
      condition('Ready', 'False', 'Upgrading'),
      condition('ConnectionDetailsReady', 'True'),
    ]);

    expect(hasConnectionDetails(upgrading)).toBe(true);
    expect(hasConnectionDetails(instance([condition('Ready', 'True')]))).toBe(
      false
    );
  });

  it('reports a restore from the Restoring condition', () => {
    expect(
      getRunningRestore(instance([condition('Restoring', 'True')]))
    ).toMatchObject({ message: 'Restoring is True' });
    expect(
      getRunningRestore(
        instance([condition('Restoring', 'False', 'Succeeded')])
      )
    ).toBeUndefined();
    expect(
      getRunningRestore(instance([condition('Ready', 'False', 'Restoring')]))
    ).toBeUndefined();
  });

  it.each([
    ['True', true],
    ['Unknown', true],
    ['False', false],
  ] as const)('with Available %s, backing up now is %s', (status, allowed) => {
    expect(canBackUpNow(instance([condition('Available', status)]))).toBe(
      allowed
    );
  });

  it('allows a backup when availability is not reported at all', () => {
    expect(canBackUpNow(instance([]))).toBe(true);
  });
});
