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

// The cases everestctl's implementation of the status display runs too: one
// file at the repository root keeps the two from drifting apart.
import fixtures from '../../../../../pkg/instancestatus/testdata/display.json';
import { InstanceStatusSource } from 'shared-types/instance.types';
import {
  areInstanceEditsBlocked,
  getInstanceStatusDisplay,
  getInstanceStatusText,
  getReasonLabel,
  hasConnectionDetails,
  isInstanceDeleting,
  isInstanceRestoring,
} from './instance-status';
import { REASON_LABELS } from './instance-status.messages';

const condition = (type: string, status: string, reason = type) => ({
  type,
  status,
  reason,
  message: '',
});

const HEALTHY = [
  condition('Available', 'True'),
  condition('Ready', 'True'),
  condition('ConnectionDetailsReady', 'True'),
];

const instance = (
  conditions: ReturnType<typeof condition>[],
  metadata: InstanceStatusSource['metadata'] = {}
): InstanceStatusSource => ({
  metadata: { generation: 2, ...metadata },
  status: { phase: 'Provisioned', observedGeneration: 2, conditions },
});

describe('getInstanceStatusDisplay', () => {
  it.each(fixtures.cases.map((fixture) => [fixture.name, fixture] as const))(
    '%s',
    (_, { instance: fixtureInstance, want }) => {
      const { tone, reason, label, available } =
        getInstanceStatusDisplay(fixtureInstance);

      expect({ tone, reason, label, available }).toEqual({
        tone: want.tone,
        reason: want.reason,
        label: want.label,
        available: want.available,
      });
    }
  );

  it('uses the label dictionary everestctl uses', () => {
    expect(REASON_LABELS).toEqual(fixtures.labels);
  });

  it('hides the message of a status that describes an older spec', () => {
    const edited = instance(
      [
        { ...condition('Ready', 'False', 'CrashLoopBackOff'), message: 'old' },
        condition('Stalled', 'True'),
      ],
      { generation: 3 }
    );

    expect(getInstanceStatusDisplay(edited)).toMatchObject({
      tone: 'progress',
      label: 'Applying changes',
      message: '',
      stale: true,
    });
  });
});

describe('getInstanceStatusText', () => {
  const textOf = (source: InstanceStatusSource | undefined) =>
    getInstanceStatusText(source, getInstanceStatusDisplay(source));

  it.each([
    ['rolling-upgrade', 'Upgrading \u00b7 online'],
    ['spec-edited-not-observed', 'Applying changes \u00b7 online'],
    ['in-place-restore', 'Restoring \u00b7 offline'],
    // Never been up, so it is not offline either.
    ['seeding-from-backup', 'Restoring'],
    ['change-availability-unknown', 'Applying changes'],
    // Outside of work the status word and its tone say it.
    ['healthy', 'Ready'],
    ['crash-loop-all-replicas-stalled', 'Crash loop'],
  ])('%s reads "%s"', (name, text) => {
    const fixture = fixtures.cases.find((item) => item.name === name);

    expect(textOf(fixture?.instance)).toBe(text);
  });

  it.each(
    fixtures.cases
      .filter(({ want }) => want.tone === 'deleting')
      .map(({ name, instance: fixtureInstance, want }) => [
        name,
        fixtureInstance,
        want.sentence,
      ])
  )('%s reads what everestctl says', (_, fixtureInstance, sentence) => {
    expect(textOf(fixtureInstance)).toBe(sentence);
  });

  it('reads Deleting as soon as the deletion is requested', () => {
    const justDeleted = instance(HEALTHY, {
      deletionTimestamp: '2026-03-01T12:00:00Z',
    });

    expect(textOf(justDeleted)).toBe('Deleting');
  });
});

describe('getReasonLabel', () => {
  it.each([
    ['PausedOutsideOpenEverest', 'Paused outside open everest'],
    ['PVCResizeFailed', 'PVC resize failed'],
    ['hasOwnProperty', 'has own property'],
  ])('spells the unknown reason %s as "%s"', (reason, label) => {
    expect(getReasonLabel(reason)).toBe(label);
  });
});

describe('action gating', () => {
  it.each<[string, InstanceStatusSource | undefined, boolean]>([
    ['healthy', instance(HEALTHY), false],
    [
      'being deleted',
      instance(HEALTHY, { deletionTimestamp: '2026-03-01T12:00:00Z' }),
      true,
    ],
    ['restoring', instance([...HEALTHY, condition('Restoring', 'True')]), true],
    ['upgrading', instance([...HEALTHY, condition('Upgrading', 'True')]), true],
    [
      'resizing its storage',
      instance([...HEALTHY, condition('StorageResizing', 'True')]),
      true,
    ],
    [
      'restore finished',
      instance([...HEALTHY, condition('Restoring', 'False')]),
      false,
    ],
    [
      'converging without a named operation',
      instance([
        condition('Available', 'True'),
        condition('Ready', 'False', 'ApplyingChanges'),
        condition('Reconciling', 'True'),
      ]),
      false,
    ],
    [
      'stalled, so that it can be fixed',
      instance([
        condition('Ready', 'False', 'ValidationFailed'),
        condition('Stalled', 'True'),
        condition('NeedsAttention', 'True'),
      ]),
      false,
    ],
    ['not reported on by a provider', instance([]), true],
    [
      'attended by nobody',
      instance([condition('Unattended', 'True', 'ProviderNotFound')]),
      true,
    ],
    ['not loaded', undefined, true],
  ])('blocks edits of an Instance that is %s: %s', (_, source, blocked) => {
    expect(areInstanceEditsBlocked(source)).toBe(blocked);
  });

  it('leaves an Instance no provider has reported on deletable', () => {
    const neverReconciled: InstanceStatusSource = {
      metadata: { generation: 1 },
      status: { phase: 'Pending', observedGeneration: -1 },
    };

    expect(areInstanceEditsBlocked(neverReconciled)).toBe(true);
    // Deleting is held back only for an Instance already being deleted.
    expect(isInstanceDeleting(neverReconciled)).toBe(false);
  });

  it('tells deletion from the deletion timestamp, not from the phase', () => {
    expect(
      isInstanceDeleting({
        metadata: {},
        status: { phase: 'Terminating' },
      })
    ).toBe(false);
    expect(
      isInstanceDeleting(
        instance(HEALTHY, { deletionTimestamp: '2026-03-01T12:00:00Z' })
      )
    ).toBe(true);
  });

  it('keeps connection details through an upgrade', () => {
    const upgrading = instance([
      condition('Available', 'True'),
      condition('Ready', 'False', 'Upgrading'),
      condition('Upgrading', 'True'),
      condition('ConnectionDetailsReady', 'True'),
    ]);

    expect(hasConnectionDetails(upgrading)).toBe(true);
    expect(hasConnectionDetails(instance([condition('Ready', 'True')]))).toBe(
      false
    );
  });

  it('reads a running restore from its own condition', () => {
    expect(
      isInstanceRestoring(instance([condition('Restoring', 'True')]))
    ).toBe(true);
    expect(
      isInstanceRestoring(instance([condition('Ready', 'False', 'Restoring')]))
    ).toBe(false);
  });
});
