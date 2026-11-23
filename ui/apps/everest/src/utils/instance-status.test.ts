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
import {
  InstanceCondition,
  InstanceStatusSource,
} from 'shared-types/instance.types';
import { PhaseType } from 'shared-types/api.types';
import { getInstanceStatusDisplay, getReasonLabel } from './instance-status';
import { HEALTHY, condition, instance } from './instance-status.test-utils';
import { REASON_LABELS } from './instance-status.messages';

const CONDITION_STATUSES: InstanceCondition['status'][] = [
  'True',
  'False',
  'Unknown',
];
const PHASES: NonNullable<PhaseType>[] = [
  'Pending',
  'Provisioned',
  'Terminating',
];

// JSON gives plain strings; a value the contract doesn't know fails the test
// rather than being let through.
const knownValue = <T extends string>(known: T[], value: string): T => {
  const match = known.find((item) => item === value);
  if (!match) {
    throw new Error(`"${value}" is not one of ${known.join(', ')}`);
  }
  return match;
};

type FixtureInstance = (typeof fixtures.cases)[number]['instance'];

const fromFixture = ({
  metadata,
  status,
}: FixtureInstance): InstanceStatusSource => ({
  metadata,
  status: {
    observedGeneration: status.observedGeneration,
    phase:
      status.phase === undefined ? undefined : knownValue(PHASES, status.phase),
    conditions: status.conditions?.map((item) => ({
      ...item,
      status: knownValue(CONDITION_STATUSES, item.status),
    })),
  },
});

const fixtureNamed = (name: string) => {
  const fixture = fixtures.cases.find((item) => item.name === name);
  if (!fixture) {
    throw new Error(`no fixture named "${name}"`);
  }
  return fromFixture(fixture.instance);
};

describe('getInstanceStatusDisplay', () => {
  it.each(fixtures.cases.map((fixture) => [fixture.name, fixture] as const))(
    '%s',
    (_, { instance: fixtureInstance, want }) => {
      const { tone, reason, label, available } = getInstanceStatusDisplay(
        fromFixture(fixtureInstance)
      );

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

describe('status text', () => {
  const textOf = (source: InstanceStatusSource | undefined) =>
    getInstanceStatusDisplay(source).text;

  it.each([
    ['rolling-upgrade', 'Upgrading \u00b7 available'],
    ['spec-edited-not-observed', 'Applying changes \u00b7 available'],
    ['in-place-restore', 'Restoring \u00b7 unavailable'],
    // Never been up, so it is not unavailable either.
    ['seeding-from-backup', 'Restoring'],
    ['change-availability-unknown', 'Applying changes'],
    // Outside of work the status word and its tone say it.
    ['healthy', 'Ready'],
    ['crash-loop-all-replicas-stalled', 'Crash loop'],
  ])('%s reads "%s"', (name, text) => {
    expect(textOf(fixtureNamed(name))).toBe(text);
  });

  it.each(
    fixtures.cases
      .filter(({ want }) => want.tone === 'deleting')
      .map(({ name, instance: fixtureInstance, want }) => [
        name,
        fromFixture(fixtureInstance),
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
