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
  InstanceAvailability,
  InstanceCondition,
  InstanceConditionType,
  InstanceStatusDisplay,
  InstanceStatusSource,
  InstanceTone,
} from 'shared-types/instance.types';
import { Messages, REASON_LABELS } from './instance-status.messages';

const REASON_CREATING = 'Creating';
const REASON_APPLYING_CHANGES = 'ApplyingChanges';
const REASON_DELETING = 'Deleting';
// Not a reason any condition carries: the word for an Instance whose provider
// has left it alone for too long.
const REASON_NOT_PICKED_UP = 'NotPickedUp';

const PHASE_PENDING = 'Pending';

// How long a spec may go unobserved before the Instance reads as abandoned.
const NOT_PICKED_UP_AFTER_MS = 5 * 60 * 1000;

// The most urgent first.
export const INSTANCE_TONES: readonly InstanceTone[] = [
  'error',
  'attention',
  'unknown',
  'progress',
  'stopped',
  'deleting',
  'ok',
];

// A database that is down comes before one nobody can vouch for.
export const INSTANCE_AVAILABILITIES: readonly InstanceAvailability[] = [
  'No',
  'Unknown',
  'Yes',
];

export const NEEDS_ATTENTION_TONES: readonly InstanceTone[] = [
  'error',
  'attention',
];

// The condition whose last transition says when the tone began.
const TONE_CONDITION: Partial<Record<InstanceTone, string>> = {
  error: InstanceConditionType.Stalled,
  attention: InstanceConditionType.NeedsAttention,
  progress: InstanceConditionType.Reconciling,
  ok: InstanceConditionType.Ready,
  stopped: InstanceConditionType.Suspended,
};

export const getInstanceCondition = (
  instance: InstanceStatusSource | undefined,
  type: string
): InstanceCondition | undefined =>
  instance?.status?.conditions?.find((condition) => condition.type === type);

export const isInstanceConditionTrue = (
  instance: InstanceStatusSource | undefined,
  type: string
) => getInstanceCondition(instance, type)?.status === 'True';

const isUpper = (char: string) => /\p{Lu}/u.test(char);
const isLower = (char: string) => /\p{Ll}/u.test(char);
const isLetterOrDigit = (char: string) => /[\p{L}\p{Nd}]/u.test(char);

// Whether the capital at index begins a word: it follows a lowercase letter
// or a digit, or it ends a run of capitals.
const startsWord = (chars: string[], index: number) => {
  if (!isUpper(chars[index])) {
    return false;
  }
  if (!isUpper(chars[index - 1])) {
    return isLetterOrDigit(chars[index - 1]);
  }
  return index + 1 < chars.length && isLower(chars[index + 1]);
};

// Puts a space before every word of a CamelCase name and lowers the first
// letter of every word but the first. A run of capitals is one word and
// keeps its case: "PVCResizeFailed" reads "PVC resize failed". The same
// steps as everestctl takes, so that both spell an unknown reason alike.
const splitWords = (name: string) => {
  const chars = [...name];

  return chars
    .map((char, index) => {
      if (index === 0 || !startsWord(chars, index)) {
        return char;
      }
      const lowered =
        index + 1 < chars.length && isLower(chars[index + 1])
          ? char.toLowerCase()
          : char;
      return ` ${lowered}`;
    })
    .join('');
};

// Never fails: providers add reasons of their own, and one this client has
// not heard of still has to be shown.
export const getReasonLabel = (reason: string) =>
  Object.prototype.hasOwnProperty.call(REASON_LABELS, reason)
    ? REASON_LABELS[reason]
    : splitWords(reason);

const toAvailability = (
  status: InstanceCondition['status'] | undefined
): InstanceAvailability => {
  if (status === 'True') {
    return 'Yes';
  }
  return status === 'False' ? 'No' : 'Unknown';
};

const newest = (times: (string | undefined)[]): number | undefined => {
  const parsed = times
    .map((time) => (time ? Date.parse(time) : NaN))
    .filter((time) => !Number.isNaN(time));
  return parsed.length ? Math.max(...parsed) : undefined;
};

// When the spec the status has not caught up with was written. Creation,
// while no provider has ever reconciled the Instance. After that nothing
// records when the spec last changed, so the newest write that was not to a
// subresource stands in for it; where the API leaves those writes out, the
// wait can't be measured.
const getSpecWrittenAt = (
  instance: InstanceStatusSource | undefined,
  neverReconciled: boolean
) =>
  neverReconciled
    ? newest([instance?.metadata?.creationTimestamp])
    : newest(
        (instance?.metadata?.managedFields ?? [])
          .filter((entry) => !entry.subresource)
          .map((entry) => entry.time)
      );

// Turns an Instance's status into what a person is shown: a tone that says
// whether to wait or act, one status word, and whether the database is up.
//
// It reads the summary conditions only. The tone comes from their statuses,
// never from a reason, so a reason this client has never seen still gets the
// right tone. The current time is an argument so that the result only depends
// on its inputs.
//
// everestctl has the same function in Go (pkg/instancestatus). The two are
// held to the same fixtures, and where those leave a choice open this one
// follows that one step for step.
export const getInstanceStatusDisplay = (
  instance: InstanceStatusSource | undefined,
  now: Date
): InstanceStatusDisplay => {
  const condition = (type: string) => getInstanceCondition(instance, type);
  const isTrue = (type: string) => condition(type)?.status === 'True';

  const ready = condition(InstanceConditionType.Ready);
  // Availability is observed, so it holds whichever spec the rest of the
  // status describes.
  const available = toAvailability(
    condition(InstanceConditionType.Available)?.status
  );
  // The phase is the only record of whether the database has ever been up,
  // and it only picks between wordings.
  const phase = instance?.status?.phase;
  const neverUp = !phase || phase === PHASE_PENDING;
  const deletionTimestamp = instance?.metadata?.deletionTimestamp;

  const generation = instance?.metadata?.generation ?? 0;
  const observedGeneration = instance?.status?.observedGeneration ?? 0;
  // The status describes an older spec, so what it says about convergence
  // and problems no longer holds. An Instance no provider has reconciled is
  // stale too: its observed generation is the API default, below any other.
  const stale = observedGeneration < generation;
  const specWrittenAt = stale
    ? getSpecWrittenAt(instance, observedGeneration < 1)
    : undefined;
  const notPickedUp =
    specWrittenAt !== undefined &&
    now.getTime() - specWrittenAt > NOT_PICKED_UP_AFTER_MS;

  // The first case that matches wins.
  let tone: InstanceTone = 'progress';
  if (deletionTimestamp) {
    tone = 'deleting';
  } else if (notPickedUp) {
    tone = 'unknown';
  } else if (isTrue(InstanceConditionType.Suspended)) {
    tone = 'stopped';
  } else if (stale) {
    // A stall on the older spec says nothing about this one: a fix may just
    // have been pushed.
    tone = 'progress';
  } else if (isTrue(InstanceConditionType.Stalled)) {
    tone = 'error';
  } else if (isTrue(InstanceConditionType.NeedsAttention)) {
    tone = 'attention';
  } else if (isTrue(InstanceConditionType.Ready)) {
    tone = 'ok';
  }

  // The status word is Ready's reason, unless Ready is missing or describes
  // an older spec.
  let reason = neverUp ? REASON_CREATING : REASON_APPLYING_CHANGES;
  if (tone === 'unknown') {
    reason = REASON_NOT_PICKED_UP;
  } else if (ready?.reason && !stale) {
    reason = ready.reason;
  } else if (tone === 'deleting') {
    reason = REASON_DELETING;
  }

  let since: string | null = null;
  if (tone === 'deleting') {
    since = deletionTimestamp ?? null;
  } else if (!stale) {
    // A long operation is a condition named like the status word it
    // supplies, and its own start is the time a person wants: an upgrade
    // began when Upgrading became True, whenever Ready last changed.
    const operation = condition(reason);
    const decidedBy = TONE_CONDITION[tone];
    since =
      (operation?.status === 'True' ? operation : undefined)
        ?.lastTransitionTime ??
      (decidedBy ? condition(decidedBy)?.lastTransitionTime : undefined) ??
      null;
  }

  const label = getReasonLabel(reason);
  let prefix = '';
  if (tone === 'deleting') {
    // "Deleting", or what the deletion is waiting for.
    prefix = reason === REASON_DELETING ? '' : Messages.deleting;
  } else if (!neverUp && tone !== 'stopped' && available !== 'Unknown') {
    // A database that has never been up is not down, and neither is one
    // that was stopped on purpose.
    prefix = available === 'Yes' ? Messages.up : Messages.down;
  }

  return {
    tone,
    reason,
    label,
    available,
    // The only message there is while stale describes an older spec.
    message: ready && !stale ? ready.message : '',
    since,
    sentence: prefix ? `${prefix}${Messages.sentenceSeparator}${label}` : label,
    stale,
  };
};

// What to say about a status that has no message of its own, because the
// provider has not caught up with the Instance.
export const getInstanceStatusExplanation = ({
  message,
  stale,
  tone,
}: Pick<InstanceStatusDisplay, 'message' | 'stale' | 'tone'>) => {
  if (message || !stale) {
    return message;
  }
  if (tone === 'unknown') {
    return Messages.notPickedUp;
  }
  return tone === 'deleting' ? Messages.beingDeleted : Messages.notObserved;
};

// How long ago something happened, in the one unit that says enough.
export const formatElapsed = (since: string, now: Date) => {
  const seconds = Math.max(
    0,
    Math.floor((now.getTime() - Date.parse(since)) / 1000)
  );
  if (seconds < 60) {
    return `${seconds} s`;
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return `${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  return hours < 24 ? `${hours} h` : `${Math.floor(hours / 24)} d`;
};

const rank = <T>(order: readonly T[], value: T) => order.indexOf(value);

// What needs someone first: by tone, then the databases that are down.
export const compareInstanceStatus = (
  a: Pick<InstanceStatusDisplay, 'tone' | 'available'>,
  b: Pick<InstanceStatusDisplay, 'tone' | 'available'>
) =>
  rank(INSTANCE_TONES, a.tone) - rank(INSTANCE_TONES, b.tone) ||
  rank(INSTANCE_AVAILABILITIES, a.available) -
    rank(INSTANCE_AVAILABILITIES, b.available);

export const compareInstanceAvailability = (
  a: InstanceAvailability,
  b: InstanceAvailability
) => rank(INSTANCE_AVAILABILITIES, a) - rank(INSTANCE_AVAILABILITIES, b);

// What the UI allows depends on the condition that makes it possible, never on
// the status word or its tone: those are for people.

// Nothing can be changed on an Instance that is going away.
export const isInstanceDeleting = (instance?: InstanceStatusSource) =>
  !!instance?.metadata?.deletionTimestamp;

// Connection details stay readable through a change: they are published once
// and outlive an upgrade.
export const hasConnectionDetails = (instance?: InstanceStatusSource) =>
  isInstanceConditionTrue(
    instance,
    InstanceConditionType.ConnectionDetailsReady
  );

// The restore that is running, if one is: an operation is its own condition,
// whatever the status word says meanwhile.
export const getRunningRestore = (instance?: InstanceStatusSource) => {
  const restoring = getInstanceCondition(
    instance,
    InstanceConditionType.Restoring
  );
  return restoring?.status === 'True' ? restoring : undefined;
};

// Only a database known to be down can't be backed up: a provider that can't
// tell leaves the decision to the engine.
export const canBackUpNow = (instance?: InstanceStatusSource) =>
  getInstanceCondition(instance, InstanceConditionType.Available)?.status !==
  'False';
